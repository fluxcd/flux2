/*
Copyright 2022 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package build

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/ytbx"
	"github.com/google/go-cmp/cmp"
	"github.com/homeport/dyff/pkg/dyff"
	"github.com/lucasb-eyer/go-colorful"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/cli-utils/pkg/kstatus/polling"
	"github.com/fluxcd/cli-utils/pkg/object"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/ssa"
	ssautil "github.com/fluxcd/pkg/ssa/utils"

	"github.com/fluxcd/flux2/v2/pkg/printers"
)

func (b *Builder) Manager() (*ssa.ResourceManager, error) {
	statusPoller := polling.NewStatusPoller(b.client, b.restMapper, polling.Options{})
	owner := ssa.Owner{
		Field: controllerName,
		Group: controllerGroup,
	}

	return ssa.NewResourceManager(b.client, statusPoller, owner), nil
}

func (b *Builder) Diff() (string, bool, error) {
	err := b.StartSpinner()
	if err != nil {
		return "", false, err
	}

	output, createdOrDrifted, diffErr := b.diff()

	err = b.StopSpinner()
	if err != nil {
		return "", false, err
	}

	return output, createdOrDrifted, diffErr
}

func (b *Builder) diff() (string, bool, error) {
	output := strings.Builder{}
	createdOrDrifted := false
	objects, err := b.Build()
	if err != nil {
		return "", createdOrDrifted, err
	}

	err = ssa.SetNativeKindsDefaults(objects)
	if err != nil {
		return "", createdOrDrifted, err
	}

	resourceManager, err := b.Manager()
	if err != nil {
		return "", createdOrDrifted, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	var diffErrs []error
	// create an inventory of objects to be reconciled
	newInventory := newInventory()
	for _, obj := range objects {
		diffOptions := ssa.DiffOptions{
			Exclusions: map[string]string{
				"kustomize.toolkit.fluxcd.io/reconcile": "disabled",
				"kustomize.toolkit.fluxcd.io/ssa":       "ignore",
			},
			IfNotPresentSelector: map[string]string{
				"kustomize.toolkit.fluxcd.io/ssa": "ifnotpresent",
			},
		}
		change, liveObject, mergedObject, err := resourceManager.Diff(ctx, obj, diffOptions)
		if err != nil {
			// gather errors and continue, as we want to see all the diffs
			diffErrs = append(diffErrs, err)
			continue
		}

		// if the object is a sops secret, we need to
		// make sure we diff only if the keys are different
		if obj.GetKind() == "Secret" && change.Action == ssa.ConfiguredAction {
			diffSopsSecret(obj, liveObject, mergedObject, change)
		}

		if change.Action == ssa.CreatedAction {
			output.WriteString(writeString(fmt.Sprintf("► %s created\n", change.Subject), bunt.Green))
			createdOrDrifted = true
		}

		if change.Action == ssa.ConfiguredAction {
			output.WriteString(bunt.Sprint(fmt.Sprintf("► %s drifted\n", change.Subject)))
			liveFile, mergedFile, tmpDir, err := writeYamls(liveObject, mergedObject)
			if err != nil {
				return "", createdOrDrifted, err
			}

			err = diff(liveFile, mergedFile, &output)
			if err != nil {
				cleanupDir(tmpDir)
				return "", createdOrDrifted, err
			}

			cleanupDir(tmpDir)
			createdOrDrifted = true
		}

		addObjectsToInventory(newInventory, change)

		if b.recursive && isKustomization(obj) && change.Action != ssa.CreatedAction {
			k, err := toKustomization(obj)
			if err != nil {
				return "", createdOrDrifted, err
			}

			if !kustomizationsEqual(k, b.kustomization) {
				if k.Spec.KubeConfig != nil {
					output.WriteString(writeString(fmt.Sprintf("⚠️ %s skipped: diff not supported for remote clusters\n", ssautil.FmtUnstructured(obj)), bunt.Orange))
				} else {
					subOutput, subCreatedOrDrifted, err := b.kustomizationDiff(k)
					if err != nil {
						diffErrs = append(diffErrs, err)
					}
					if subCreatedOrDrifted {
						createdOrDrifted = true
						output.WriteString(bunt.Sprint(fmt.Sprintf("📁 %s changed\n", ssautil.FmtUnstructured(obj))))
						output.WriteString(subOutput)
					}
				}

				// finished with Kustomization diff
				if b.spinner != nil {
					b.spinner.Message(spinnerDryRunMessage)
				}
			}
		}
	}

	if b.spinner != nil {
		b.spinner.Message("processing inventory")
	}

	if b.kustomization.Spec.Prune && len(diffErrs) == 0 {
		oldStatus := b.kustomization.Status.DeepCopy()
		if oldStatus.Inventory != nil {
			staleObjects, err := diffInventory(oldStatus.Inventory, newInventory)
			if err != nil {
				return "", createdOrDrifted, err
			}
			if len(staleObjects) > 0 {
				createdOrDrifted = true
			}
			for _, object := range staleObjects {
				output.WriteString(writeString(fmt.Sprintf("► %s deleted\n", ssautil.FmtUnstructured(object)), bunt.OrangeRed))
			}
		}
	}

	return output.String(), createdOrDrifted, errors.Reduce(errors.Flatten(errors.NewAggregate(diffErrs)))
}

func (b *Builder) kustomizationDiff(kustomization *kustomizev1.Kustomization) (string, bool, error) {
	if b.spinner != nil {
		b.spinner.Message(fmt.Sprintf("%s in %s", spinnerDryRunMessage, kustomization.Name))
	}

	sourceRef := kustomization.Spec.SourceRef.DeepCopy()
	if sourceRef.Namespace == "" {
		sourceRef.Namespace = kustomization.Namespace
	}

	sourceKey := sourceRef.String()
	localPath, ok := b.localSources[sourceKey]
	if !ok {
		return "", false, fmt.Errorf("cannot get local path for %s of kustomization %s", sourceKey, kustomization.Name)
	}

	resourcesPath := filepath.Join(localPath, kustomization.Spec.Path)
	subBuilder, err := NewBuilder(kustomization.Name, resourcesPath,
		// use same client and spinner
		withClientConfigFrom(b),
		withSpinnerFrom(b),
		WithTimeout(b.timeout),
		WithNamespace(kustomization.Namespace),
		WithIgnore(b.ignore),
		WithStrictSubstitute(b.strictSubst),
		WithRecursive(b.recursive),
		WithLocalSources(b.localSources),
		WithSingleKustomization(),
	)
	if err != nil {
		return "", false, err
	}

	return subBuilder.diff()
}

func writeYamls(liveObject, mergedObject *unstructured.Unstructured) (string, string, string, error) {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", "", "", err
	}

	liveYAML, _ := yaml.Marshal(liveObject)
	liveFile := filepath.Join(tmpDir, "live.yaml")
	if err := os.WriteFile(liveFile, liveYAML, 0o600); err != nil {
		return "", "", "", err
	}

	mergedYAML, _ := yaml.Marshal(mergedObject)
	mergedFile := filepath.Join(tmpDir, "merged.yaml")
	if err := os.WriteFile(mergedFile, mergedYAML, 0o600); err != nil {
		return "", "", "", err
	}

	return liveFile, mergedFile, tmpDir, nil
}

func writeString(t string, color colorful.Color) string {
	return bunt.Style(
		t,
		bunt.EachLine(),
		bunt.Foreground(color),
	)
}

func cleanupDir(dir string) error {
	return os.RemoveAll(dir)
}

func diff(liveFile, mergedFile string, output io.Writer) error {
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

	printer := printers.NewDyffPrinter()

	printer.Print(output, report)

	return nil
}

func diffSopsSecret(obj, liveObject, mergedObject *unstructured.Unstructured, change *ssa.ChangeSetEntry) {
	// get both data and stringdata maps
	data := obj.Object[dataField]

	if m, ok := data.(map[string]interface{}); ok && m != nil {
		applySopsDiff(m, liveObject, mergedObject, change)
	}
}

func applySopsDiff(data map[string]interface{}, liveObject, mergedObject *unstructured.Unstructured, change *ssa.ChangeSetEntry) {
	for _, v := range data {
		v, err := base64.StdEncoding.DecodeString(v.(string))
		if err != nil {
			fmt.Println(err)
		}

		if bytes.Contains(v, []byte(mask)) {
			if liveObject != nil && mergedObject != nil {
				change.Action = ssa.UnchangedAction
				liveKeys, mergedKeys := sopsComparableByKeys(liveObject), sopsComparableByKeys(mergedObject)
				if cmp.Diff(liveKeys, mergedKeys) != "" {
					change.Action = ssa.ConfiguredAction
				}
			}
		}
	}
}

func sopsComparableByKeys(object *unstructured.Unstructured) []string {
	m := object.Object[dataField].(map[string]interface{})
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		// make sure we can compare only on keys
		m[k] = "*****"
		keys[i] = k
		i++
	}

	object.Object[dataField] = m

	sort.Strings(keys)

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
