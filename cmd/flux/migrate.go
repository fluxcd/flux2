/*
Copyright 2025 The Flux authors

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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fluxcd/pkg/ssa"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	imageautov1 "github.com/fluxcd/image-automation-controller/api/v1"
	imageautov1b2 "github.com/fluxcd/image-automation-controller/api/v1beta2"
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1"
	imagev1b2 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notificationv1 "github.com/fluxcd/notification-controller/api/v1"
	notificationv1b3 "github.com/fluxcd/notification-controller/api/v1beta3"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	swv1b1 "github.com/fluxcd/source-watcher/api/v2/v1beta1"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

// APIVersions holds the mapping of GroupKinds to their respective
// latest API versions for a specific Flux version.
type APIVersions struct {
	FluxVersion    string
	LatestVersions map[schema.GroupKind]string
}

// TODO: Update this mapping when new Flux minor versions are released!
// latestAPIVersions contains the latest API versions for each GroupKind
// for each supported Flux version. We maintain the latest two minor versions.
var latestAPIVersions = []APIVersions{
	{
		FluxVersion: "2.7",
		LatestVersions: map[schema.GroupKind]string{
			// source-controller
			{Group: sourcev1.GroupVersion.Group, Kind: sourcev1.BucketKind}:           sourcev1.GroupVersion.Version,
			{Group: sourcev1.GroupVersion.Group, Kind: sourcev1.GitRepositoryKind}:    sourcev1.GroupVersion.Version,
			{Group: sourcev1.GroupVersion.Group, Kind: sourcev1.OCIRepositoryKind}:    sourcev1.GroupVersion.Version,
			{Group: sourcev1.GroupVersion.Group, Kind: sourcev1.HelmRepositoryKind}:   sourcev1.GroupVersion.Version,
			{Group: sourcev1.GroupVersion.Group, Kind: sourcev1.HelmChartKind}:        sourcev1.GroupVersion.Version,
			{Group: sourcev1.GroupVersion.Group, Kind: sourcev1.ExternalArtifactKind}: sourcev1.GroupVersion.Version,

			// kustomize-controller
			{Group: kustomizev1.GroupVersion.Group, Kind: kustomizev1.KustomizationKind}: kustomizev1.GroupVersion.Version,

			// helm-controller
			{Group: helmv2.GroupVersion.Group, Kind: helmv2.HelmReleaseKind}: helmv2.GroupVersion.Version,

			// notification-controller
			{Group: notificationv1.GroupVersion.Group, Kind: notificationv1.ReceiverKind}:     notificationv1.GroupVersion.Version,
			{Group: notificationv1b3.GroupVersion.Group, Kind: notificationv1b3.AlertKind}:    notificationv1b3.GroupVersion.Version,
			{Group: notificationv1b3.GroupVersion.Group, Kind: notificationv1b3.ProviderKind}: notificationv1b3.GroupVersion.Version,

			// image-reflector-controller
			{Group: imagev1.GroupVersion.Group, Kind: imagev1.ImageRepositoryKind}: imagev1.GroupVersion.Version,
			{Group: imagev1.GroupVersion.Group, Kind: imagev1.ImagePolicyKind}:     imagev1.GroupVersion.Version,

			// image-automation-controller
			{Group: imageautov1.GroupVersion.Group, Kind: imageautov1.ImageUpdateAutomationKind}: imageautov1.GroupVersion.Version,

			// source-watcher
			{Group: swv1b1.GroupVersion.Group, Kind: swv1b1.ArtifactGeneratorKind}: swv1b1.GroupVersion.Version,
		},
	},
	{
		FluxVersion: "2.6",
		LatestVersions: map[schema.GroupKind]string{
			// source-controller
			{Group: sourcev1.GroupVersion.Group, Kind: sourcev1.BucketKind}:           sourcev1.GroupVersion.Version,
			{Group: sourcev1.GroupVersion.Group, Kind: sourcev1.GitRepositoryKind}:    sourcev1.GroupVersion.Version,
			{Group: sourcev1.GroupVersion.Group, Kind: sourcev1.OCIRepositoryKind}:    sourcev1.GroupVersion.Version,
			{Group: sourcev1.GroupVersion.Group, Kind: sourcev1.HelmRepositoryKind}:   sourcev1.GroupVersion.Version,
			{Group: sourcev1.GroupVersion.Group, Kind: sourcev1.HelmChartKind}:        sourcev1.GroupVersion.Version,
			{Group: sourcev1.GroupVersion.Group, Kind: sourcev1.ExternalArtifactKind}: sourcev1.GroupVersion.Version,

			// kustomize-controller
			{Group: kustomizev1.GroupVersion.Group, Kind: kustomizev1.KustomizationKind}: kustomizev1.GroupVersion.Version,

			// helm-controller
			{Group: helmv2.GroupVersion.Group, Kind: helmv2.HelmReleaseKind}: helmv2.GroupVersion.Version,

			// notification-controller
			{Group: notificationv1.GroupVersion.Group, Kind: notificationv1.ReceiverKind}:     notificationv1.GroupVersion.Version,
			{Group: notificationv1b3.GroupVersion.Group, Kind: notificationv1b3.AlertKind}:    notificationv1b3.GroupVersion.Version,
			{Group: notificationv1b3.GroupVersion.Group, Kind: notificationv1b3.ProviderKind}: notificationv1b3.GroupVersion.Version,

			// image-reflector-controller
			{Group: imagev1b2.GroupVersion.Group, Kind: imagev1b2.ImageRepositoryKind}: imagev1b2.GroupVersion.Version,
			{Group: imagev1b2.GroupVersion.Group, Kind: imagev1b2.ImagePolicyKind}:     imagev1b2.GroupVersion.Version,

			// image-automation-controller
			{Group: imageautov1b2.GroupVersion.Group, Kind: imageautov1b2.ImageUpdateAutomationKind}: imageautov1b2.GroupVersion.Version,
		},
	},
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Args:  cobra.NoArgs,
	Short: "Migrate the Flux custom resources to their latest API version",
	Long: `The migrate command must be run before a Flux minor version upgrade.

The command has two modes of operation:

- Cluster mode (default): migrates all the Flux custom resources stored in Kubernetes etcd to their latest API version.
- File system mode (-f): migrates the Flux custom resources defined in the manifests located in the specified path.
`,
	Example: `  # Migrate all the Flux custom resources in the cluster.
  # This uses the current kubeconfig context and requires cluster-admin permissions.
  flux migrate

  # Migrate all the Flux custom resources in a Git repository
  # checked out in the current working directory.
  flux migrate -f .

  # Migrate all Flux custom resources defined in YAML and Helm YAML template files.
  flux migrate -f . --extensions=.yml,.yaml,.tpl

  # Migrate the Flux custom resources to the latest API versions of Flux 2.6.
  flux migrate -f . --version=2.6

  # Migrate the Flux custom resources defined in a multi-document YAML manifest file.
  flux migrate -f path/to/manifest.yaml

  # Simulate the migration without making any changes.
  flux migrate -f . --dry-run

  # Run the migration skipping confirmation prompts.
  flux migrate -f . --yes
`,
	RunE: runMigrateCmd,
}

var migrateFlags struct {
	yes        bool
	dryRun     bool
	path       string
	version    string
	extensions []string
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().StringVarP(&migrateFlags.path, "path", "f", "",
		"the path to the directory containing the manifests to migrate")
	migrateCmd.Flags().StringSliceVarP(&migrateFlags.extensions, "extensions", "e", []string{".yaml", ".yml"},
		"the file extensions to consider when migrating manifests, only applicable with --path")
	migrateCmd.Flags().StringVarP(&migrateFlags.version, "version", "v", "",
		"the target Flux minor version to migrate manifests to, only applicable with --path (defaults to the version of the CLI)")
	migrateCmd.Flags().BoolVarP(&migrateFlags.yes, "yes", "y", false,
		"skip confirmation prompts when migrating manifests, only applicable with --path")
	migrateCmd.Flags().BoolVar(&migrateFlags.dryRun, "dry-run", false,
		"simulate the migration of manifests without making any changes, only applicable with --path")
}

func runMigrateCmd(*cobra.Command, []string) error {
	if migrateFlags.path == "" {
		return migrateCluster()
	}
	return migrateFileSystem()
}

func migrateCluster() error {
	logger.Actionf("starting migration of custom resources")
	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	cfg, err := utils.KubeConfig(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return fmt.Errorf("the Kubernetes client initialization failed: %w", err)
	}

	kubeClient, err := client.New(cfg, client.Options{Scheme: utils.NewScheme()})
	if err != nil {
		return err
	}

	migrator := NewClusterMigrator(kubeClient, client.MatchingLabels{
		"app.kubernetes.io/part-of": "flux",
	})

	if err := migrator.Run(ctx); err != nil {
		return err
	}

	logger.Successf("custom resources migrated successfully")
	return nil
}

func migrateFileSystem() error {
	pathRoot, err := os.OpenRoot(".")
	if err != nil {
		return fmt.Errorf("failed to open filesystem at the current working directory: %w", err)
	}
	defer pathRoot.Close()

	fileSystem := &osFS{pathRoot.FS()}
	yes := migrateFlags.yes
	dryRun := migrateFlags.dryRun
	path := migrateFlags.path
	extensions := migrateFlags.extensions
	var latestVersions map[schema.GroupKind]string

	// Determine latest API versions based on the Flux version.
	if migrateFlags.version == "" {
		latestVersions = latestAPIVersions[0].LatestVersions
	} else {
		supportedVersions := make([]string, 0, len(latestAPIVersions))
		for _, v := range latestAPIVersions {
			if v.FluxVersion == migrateFlags.version {
				latestVersions = v.LatestVersions
				break
			}
			supportedVersions = append(supportedVersions, v.FluxVersion)
		}
		if latestVersions == nil {
			return fmt.Errorf("version %s is not supported, supported versions are: %s",
				migrateFlags.version, strings.Join(supportedVersions, ", "))
		}
	}

	return NewFileSystemMigrator(fileSystem, yes, dryRun, path, extensions, latestVersions).Run()
}

// ClusterMigrator migrates all the CRs in the cluster for the CRDs matching the label selector.
type ClusterMigrator struct {
	labelSelector client.MatchingLabels
	kubeClient    client.Client
}

// NewClusterMigrator creates a new ClusterMigrator instance with the specified label selector.
func NewClusterMigrator(kubeClient client.Client, labelSelector client.MatchingLabels) *ClusterMigrator {
	return &ClusterMigrator{
		labelSelector: labelSelector,
		kubeClient:    kubeClient,
	}
}

func (c *ClusterMigrator) Run(ctx context.Context) error {
	crdList := &apiextensionsv1.CustomResourceDefinitionList{}

	if err := c.kubeClient.List(ctx, crdList, c.labelSelector); err != nil {
		return fmt.Errorf("failed to list CRDs: %w", err)
	}

	for _, crd := range crdList.Items {
		if err := c.migrateCRD(ctx, crd.Name); err != nil {
			return err
		}
	}

	return nil
}

func (c *ClusterMigrator) migrateCRD(ctx context.Context, name string) error {
	crd := &apiextensionsv1.CustomResourceDefinition{}

	if err := c.kubeClient.Get(ctx, client.ObjectKey{Name: name}, crd); err != nil {
		return fmt.Errorf("failed to get CRD %s: %w", name, err)
	}

	// get the latest storage version for the CRD
	storageVersion := c.getStorageVersion(crd)
	if storageVersion == "" {
		return fmt.Errorf("no storage version found for CRD %s", name)
	}

	// migrate all the resources for the CRD
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return c.migrateCR(ctx, crd, storageVersion)
	})
	if err != nil {
		return fmt.Errorf("failed to migrate resources for CRD %s: %w", name, err)
	}

	// set the CRD status to contain only the latest storage version
	if len(crd.Status.StoredVersions) > 1 || crd.Status.StoredVersions[0] != storageVersion {
		crd.Status.StoredVersions = []string{storageVersion}
		if err := c.kubeClient.Status().Update(ctx, crd); err != nil {
			return fmt.Errorf("failed to update CRD %s status: %w", crd.Name, err)
		}
		logger.Successf("%s migrated to storage version %s", crd.Name, storageVersion)
	}
	return nil
}

// migrateCR migrates all CRs for the given CRD to the specified version by patching them.
func (c *ClusterMigrator) migrateCR(ctx context.Context, crd *apiextensionsv1.CustomResourceDefinition, version string) error {
	list := &unstructured.UnstructuredList{}

	apiVersion := crd.Spec.Group + "/" + version
	listKind := crd.Spec.Names.ListKind

	list.SetAPIVersion(apiVersion)
	list.SetKind(listKind)

	err := c.kubeClient.List(ctx, list, client.InNamespace(""))
	if err != nil {
		return fmt.Errorf("failed to list resources for CRD %s: %w", crd.Name, err)
	}

	if len(list.Items) == 0 {
		return nil
	}

	for _, item := range list.Items {
		patches, err := ssa.PatchMigrateToVersion(&item, apiVersion)
		if err != nil {
			return fmt.Errorf("failed to create migration patch for %s/%s/%s: %w",
				item.GetKind(), item.GetNamespace(), item.GetName(), err)
		}

		if len(patches) == 0 {
			// patch the resource with an empty patch to update the version
			if err := c.kubeClient.Patch(
				ctx,
				&item,
				client.RawPatch(client.Merge.Type(), []byte("{}")),
			); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf(" %s/%s/%s failed to migrate: %w",
					item.GetKind(), item.GetNamespace(), item.GetName(), err)
			}
		} else {
			// patch the resource to migrate the managed fields to the latest apiVersion
			rawPatch, err := json.Marshal(patches)
			if err != nil {
				return fmt.Errorf("failed to marshal migration patch for %s/%s/%s: %w",
					item.GetKind(), item.GetNamespace(), item.GetName(), err)
			}
			if err := c.kubeClient.Patch(
				ctx,
				&item,
				client.RawPatch(types.JSONPatchType, rawPatch),
			); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf(" %s/%s/%s failed to migrate managed fields: %w",
					item.GetKind(), item.GetNamespace(), item.GetName(), err)
			}
		}

		logger.Successf("%s/%s/%s migrated to version %s",
			item.GetKind(), item.GetNamespace(), item.GetName(), version)
	}

	return nil
}

// getStorageVersion retrieves the storage version of a CustomResourceDefinition.
func (c *ClusterMigrator) getStorageVersion(crd *apiextensionsv1.CustomResourceDefinition) string {
	var version string
	for _, v := range crd.Spec.Versions {
		if v.Storage {
			version = v.Name
			break
		}
	}

	return version
}

// WritableFS extends fs.FS with a WriteFile method.
type WritableFS interface {
	fs.FS
	WriteFile(name string, data []byte, perm os.FileMode) error
}

// osFS is a WritableFS implementation that uses the file system of the OS.
type osFS struct {
	fs.FS
}

func (o *osFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

// FileSystemMigrator migrates all the CRs found in the manifests located in the specified path.
type FileSystemMigrator struct {
	fileSystem     WritableFS
	yes            bool
	dryRun         bool
	path           string
	extensions     []string
	latestVersions map[schema.GroupKind]string
}

// FileAPIUpgrades represents the API upgrades detected in a specific manifest file.
type FileAPIUpgrades struct {
	File     string
	Upgrades []APIUpgrade
}

// APIUpgrade represents an upgrade of a specific API version in a manifest file.
type APIUpgrade struct {
	Line       int
	Kind       string
	OldVersion string
	NewVersion string
}

// NewFileSystemMigrator creates a new FileSystemMigrator instance with the specified flags.
func NewFileSystemMigrator(fileSystem WritableFS, yes, dryRun bool, path string,
	extensions []string, latestVersions map[schema.GroupKind]string) *FileSystemMigrator {
	return &FileSystemMigrator{
		fileSystem:     fileSystem,
		yes:            yes,
		dryRun:         dryRun,
		path:           filepath.Clean(path), // convert dir/ to dir to avoid error when walking
		extensions:     extensions,
		latestVersions: latestVersions,
	}
}

func (f *FileSystemMigrator) Run() error {
	logger.Actionf("starting migration of custom resources")

	// List and filter files.
	files, err := f.listFiles()
	if err != nil {
		return err
	}

	// Detect upgrades.
	upgrades, err := f.detectUpgrades(files)
	if err != nil {
		return err
	}
	if len(upgrades) == 0 {
		logger.Successf("no custom resources found that require migration")
		return nil
	}
	if f.dryRun {
		logger.Successf("dry-run mode enabled, no changes will be made")
		return nil
	}

	// Confirm upgrades.
	if !f.yes {
		prompt := promptui.Prompt{
			Label:     "Are you sure you want to proceed with the above upgrades", // Already prints "? [y/N]"
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			return err
		}
	}

	// Migrate files.
	for _, fileUpgrades := range upgrades {
		if err := f.migrateFile(&fileUpgrades); err != nil {
			return err
		}
		logger.Successf("file %s migrated successfully", fileUpgrades.File)
	}

	logger.Successf("custom resources migrated successfully")
	return nil
}

func (f *FileSystemMigrator) listFiles() ([]string, error) {
	fileInfo, err := fs.Stat(f.fileSystem, f.path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path %s: %w", f.path, err)
	}
	if fileInfo.IsDir() {
		return f.listDirectoryFiles()
	}
	if err := f.validateSingleFile(); err != nil {
		return nil, err
	}
	return []string{f.path}, nil
}

func (f *FileSystemMigrator) listDirectoryFiles() ([]string, error) {
	var files []string
	err := fs.WalkDir(f.fileSystem, f.path, func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !f.matchesExtensions(path) {
			return nil
		}
		fileInfo, err := dirEntry.Info()
		if err != nil {
			return err
		}
		if fileInfo.Mode().IsRegular() {
			files = append(files, path)
		} else if !fileInfo.IsDir() {
			logger.Warningf("skipping irregular file %s", path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", f.path, err)
	}
	return files, nil
}

func (f *FileSystemMigrator) validateSingleFile() error {
	if !f.matchesExtensions(f.path) {
		return fmt.Errorf("file %s does not match the specified extensions: %v",
			f.path, strings.Join(f.extensions, ", "))
	}

	// Check if it's irregular by walking the parent directory.
	var irregular bool
	err := fs.WalkDir(f.fileSystem, filepath.Dir(f.path), func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != f.path {
			return nil
		}
		fileInfo, err := dirEntry.Info()
		if err != nil {
			return err
		}
		if !fileInfo.Mode().IsRegular() {
			irregular = true
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to validate file %s: %w", f.path, err)
	}
	if irregular {
		return fmt.Errorf("file %s is irregular", f.path)
	}
	return nil
}

func (f *FileSystemMigrator) matchesExtensions(file string) bool {
	for _, ext := range f.extensions {
		if strings.HasSuffix(file, ext) {
			return true
		}
	}
	return false
}

func (f *FileSystemMigrator) detectUpgrades(files []string) ([]FileAPIUpgrades, error) {
	var upgrades []FileAPIUpgrades
	for _, file := range files {
		fileUpgrades, err := f.detectFileUpgrades(file)
		if err != nil {
			return nil, err
		}
		if len(fileUpgrades) == 0 {
			continue
		}
		fu := FileAPIUpgrades{
			File:     file,
			Upgrades: fileUpgrades,
		}
		upgrades = append(upgrades, fu)
		f.printDetectedUpgrades(&fu)
	}
	return upgrades, nil
}

func (f *FileSystemMigrator) detectFileUpgrades(file string) ([]APIUpgrade, error) {
	b, err := fs.ReadFile(f.fileSystem, file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", file, err)
	}
	lines := strings.Split(string(b), "\n")

	var fileUpgrades []APIUpgrade
	for line, apiVersionLine := range lines {
		// Parse apiVersion.
		const apiVersionPrefix = "apiVersion: "
		idx := strings.Index(apiVersionLine, apiVersionPrefix)
		if idx == -1 {
			continue
		}
		apiVersionValuePrefix := strings.TrimSpace(apiVersionLine[idx+len(apiVersionPrefix):])
		apiVersion := strings.Split(apiVersionValuePrefix, " ")[0]
		gv, err := schema.ParseGroupVersion(apiVersion)
		if err != nil {
			logger.Warningf("%s:%d: %v", file, line+1, err)
			continue
		}

		// Parse kind.
		if line+1 >= len(lines) {
			continue
		}
		kindLine := lines[line+1]
		const kindPrefix = "kind: "
		idx = strings.Index(kindLine, kindPrefix)
		if idx == -1 {
			continue
		}
		kind := strings.TrimSpace(kindLine[idx+len(kindPrefix):])

		// Build GroupKind.
		gk := schema.GroupKind{
			Group: gv.Group,
			Kind:  kind,
		}

		// Check if there's a newer version for the GroupKind.
		latestVersion, ok := f.latestVersions[gk]
		if !ok || latestVersion == gv.Version {
			continue
		}

		// Record the upgrade.
		fileUpgrades = append(fileUpgrades, APIUpgrade{
			Line:       line,
			Kind:       kind,
			OldVersion: gv.Version,
			NewVersion: latestVersion,
		})
	}
	return fileUpgrades, nil
}

func (f *FileSystemMigrator) printDetectedUpgrades(fileUpgrades *FileAPIUpgrades) {
	for _, upgrade := range fileUpgrades.Upgrades {
		logger.Generatef("%s:%d: %s %s -> %s",
			fileUpgrades.File,
			upgrade.Line+1,
			upgrade.Kind,
			upgrade.OldVersion,
			upgrade.NewVersion)
	}
}

func (f *FileSystemMigrator) migrateFile(fileUpgrades *FileAPIUpgrades) error {
	// Read file and map lines.
	b, err := fs.ReadFile(f.fileSystem, fileUpgrades.File)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", fileUpgrades.File, err)
	}
	lines := strings.Split(string(b), "\n")

	// Apply upgrades to lines.
	for _, upgrade := range fileUpgrades.Upgrades {
		line := lines[upgrade.Line]
		line = strings.Replace(line, upgrade.OldVersion, upgrade.NewVersion, 1)
		lines[upgrade.Line] = line
	}

	// Read file info to preserve permissions.
	fileInfo, err := fs.Stat(f.fileSystem, fileUpgrades.File)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", fileUpgrades.File, err)
	}

	// Write file with preserved permissions.
	b = []byte(strings.Join(lines, "\n"))
	if err := f.fileSystem.WriteFile(fileUpgrades.File, b, fileInfo.Mode()); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fileUpgrades.File, err)
	}

	return nil
}
