package kustomization

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	"github.com/fluxcd/pkg/ssa"
	"github.com/gonvenience/bunt"
	"github.com/gonvenience/ytbx"
	"github.com/google/go-cmp/cmp"
	"github.com/homeport/dyff/pkg/dyff"
	"github.com/lucasb-eyer/go-colorful"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/yaml"
)

const (
	controllerName  = "kustomize-controller"
	controllerGroup = "kustomize.toolkit.fluxcd.io"
)

func (b *Builder) manager() (*ssa.ResourceManager, error) {
	statusPoller := polling.NewStatusPoller(b.client, b.restMapper)
	owner := ssa.Owner{
		Field: controllerName,
		Group: controllerGroup,
	}

	return ssa.NewResourceManager(b.client, statusPoller, owner), nil
}

func (b *Builder) Diff() error {
	res, err := b.Build()
	if err != nil {
		return err
	}
	// convert the build result into Kubernetes unstructured objects
	objects, err := ssa.ReadObjects(bytes.NewReader(res))
	if err != nil {
		return err
	}

	resourceManager, err := b.manager()
	if err != nil {
		return err
	}

	resourceManager.SetOwnerLabels(objects, b.kustomization.GetName(), b.kustomization.GetNamespace())

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	if err := ssa.SetNativeKindsDefaults(objects); err != nil {
		return err
	}

	// create an inventory of objects to be reconciled
	newInventory := newInventory()
	for _, obj := range objects {
		change, liveObject, mergedObject, err := resourceManager.Diff(ctx, obj)
		if err != nil {
			if b.kustomization.Spec.Force && strings.Contains(err.Error(), "immutable") {
				writeString(fmt.Sprintf("► %s created", obj.GetName()), bunt.Green)
			} else {
				writeString(fmt.Sprint(`✗`, err), bunt.Red)
			}
			continue
		}

		// if the object is a sops secret, we need to
		// make sure we diff only if the keys are different
		if obj.GetKind() == "Secret" && change.Action == string(ssa.ConfiguredAction) {
			diffSopsSecret(obj, liveObject, mergedObject, change)
		}

		if change.Action == string(ssa.CreatedAction) {
			writeString(fmt.Sprintf("► %s created", change.Subject), bunt.Green)
		}

		if change.Action == string(ssa.ConfiguredAction) {
			writeString(fmt.Sprintf("► %s drifted", change.Subject), bunt.WhiteSmoke)
			liveFile, mergedFile, tmpDir, err := writeYamls(liveObject, mergedObject)
			if err != nil {
				return err
			}
			defer cleanupDir(tmpDir)

			err = diff(liveFile, mergedFile)
			if err != nil {
				return err
			}
		}

		addObjectsToInventory(newInventory, change)
	}

	if b.kustomization.Spec.Prune {
		oldStatus := b.kustomization.Status.DeepCopy()
		if oldStatus.Inventory != nil {
			diffObjects, err := diffInventory(oldStatus.Inventory, newInventory)
			if err != nil {
				return err
			}
			for _, object := range diffObjects {
				writeString(fmt.Sprintf("► %s deleted", ssa.FmtUnstructured(object)), bunt.OrangeRed)
			}
		}
	}

	return nil
}

func writeYamls(liveObject, mergedObject *unstructured.Unstructured) (string, string, string, error) {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", "", "", err
	}

	liveYAML, _ := yaml.Marshal(liveObject)
	liveFile := filepath.Join(tmpDir, "live.yaml")
	if err := os.WriteFile(liveFile, liveYAML, 0644); err != nil {
		return "", "", "", err
	}

	mergedYAML, _ := yaml.Marshal(mergedObject)
	mergedFile := filepath.Join(tmpDir, "merged.yaml")
	if err := os.WriteFile(mergedFile, mergedYAML, 0644); err != nil {
		return "", "", "", err
	}

	return liveFile, mergedFile, tmpDir, nil
}

func writeString(t string, color colorful.Color) {
	fmt.Println(bunt.Style(
		t,
		bunt.EachLine(),
		bunt.Foreground(color),
	))
}

func cleanupDir(dir string) error {
	return os.RemoveAll(dir)
}

func diff(liveFile, mergedFile string) error {
	from, to, err := ytbx.LoadFiles(liveFile, mergedFile)
	if err != nil {
		return fmt.Errorf("failed to load input files: %w", err)
	}

	report, err := dyff.CompareInputFiles(from, to,
		dyff.IgnoreOrderChanges(false),
		dyff.KubernetesEntityDetection(true),
	)
	if err != nil {
		return fmt.Errorf("failed to compare input files: %w", err)
	}

	reportWriter := &dyff.HumanReport{
		Report:     report,
		OmitHeader: true,
	}

	if err := reportWriter.WriteReport(os.Stdout); err != nil {
		return fmt.Errorf("failed to print report: %w", err)
	}

	return nil
}

func diffSopsSecret(obj, liveObject, mergedObject *unstructured.Unstructured, change *ssa.ChangeSetEntry) {
	data := obj.Object["data"]
	for _, v := range data.(map[string]interface{}) {
		v, err := base64.StdEncoding.DecodeString(v.(string))
		if err != nil {
			fmt.Println(err)
		}
		if bytes.Contains(v, []byte(mask)) {
			if liveObject != nil && mergedObject != nil {
				change.Action = string(ssa.UnchangedAction)
				dataLive := liveObject.Object["data"].(map[string]interface{})
				dataMerged := mergedObject.Object["data"].(map[string]interface{})
				if cmp.Diff(keys(dataLive), keys(dataMerged)) != "" {
					change.Action = string(ssa.ConfiguredAction)
				}
			}
		}
	}
}

func keys(m map[string]interface{}) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}

// diffInventory returns the slice of objects that do not exist in the target inventory.
func diffInventory(inv *kustomizev1.ResourceInventory, target *kustomizev1.ResourceInventory) ([]*unstructured.Unstructured, error) {
	versionOf := func(i *kustomizev1.ResourceInventory, objMetadata object.ObjMetadata) string {
		for _, entry := range i.Entries {
			if entry.ID == objMetadata.String() {
				return entry.Version
			}
		}
		return ""
	}

	objects := make([]*unstructured.Unstructured, 0)
	aList, err := listMetaInInventory(inv)
	if err != nil {
		return nil, err
	}

	bList, err := listMetaInInventory(target)
	if err != nil {
		return nil, err
	}

	list := aList.Diff(bList)
	if len(list) == 0 {
		return objects, nil
	}

	for _, metadata := range list {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   metadata.GroupKind.Group,
			Kind:    metadata.GroupKind.Kind,
			Version: versionOf(inv, metadata),
		})
		u.SetName(metadata.Name)
		u.SetNamespace(metadata.Namespace)
		objects = append(objects, u)
	}

	sort.Sort(ssa.SortableUnstructureds(objects))
	return objects, nil
}

// listMetaInInventory returns the inventory entries as object.ObjMetadata objects.
func listMetaInInventory(inv *kustomizev1.ResourceInventory) (object.ObjMetadataSet, error) {
	var metas []object.ObjMetadata
	for _, e := range inv.Entries {
		m, err := object.ParseObjMetadata(e.ID)
		if err != nil {
			return metas, err
		}
		metas = append(metas, m)
	}

	return metas, nil
}

func newInventory() *kustomizev1.ResourceInventory {
	return &kustomizev1.ResourceInventory{
		Entries: []kustomizev1.ResourceRef{},
	}
}

// addObjectsToInventory extracts the metadata from the given objects and adds it to the inventory.
func addObjectsToInventory(inv *kustomizev1.ResourceInventory, entry *ssa.ChangeSetEntry) error {
	if entry == nil {
		return nil
	}

	inv.Entries = append(inv.Entries, kustomizev1.ResourceRef{
		ID:      entry.ObjMetadata.String(),
		Version: entry.GroupVersion,
	})

	return nil
}
