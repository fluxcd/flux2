/*
Copyright 2020 The Flux CD contributors.

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
	"fmt"
	"github.com/fluxcd/pkg/untar"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the toolkit components",
	Long: `The install command deploys the toolkit components in the specified namespace.
If a previous version is installed, then an in-place upgrade will be performed.`,
	Example: `  # Install the latest version in the gitops-systems namespace
  tk install --version=latest --namespace=gitops-systems

  # Dry-run install for a specific version and a series of components
  tk install --dry-run --version=v0.0.7 --components="source-controller,kustomize-controller"

  # Dry-run install with manifests preview 
  tk install --dry-run --verbose

  # Write install manifests to file 
  tk install --export > gitops-system.yaml
`,
	RunE: installCmdRun,
}

var (
	installExport        bool
	installDryRun        bool
	installManifestsPath string
	installVersion       string
	installComponents    []string
)

func init() {
	installCmd.Flags().BoolVar(&installExport, "export", false,
		"write the install manifests to stdout and exit")
	installCmd.Flags().BoolVarP(&installDryRun, "dry-run", "", false,
		"only print the object that would be applied")
	installCmd.Flags().StringVarP(&installVersion, "version", "v", defaultVersion,
		"toolkit version")
	installCmd.Flags().StringSliceVar(&installComponents, "components", defaultComponents,
		"list of components, accepts comma-separated values")
	installCmd.Flags().StringVarP(&installManifestsPath, "manifests", "", "",
		"path to the manifest directory, dev only")
	rootCmd.AddCommand(installCmd)
}

func installCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var kustomizePath string
	if installVersion == "" && !strings.HasPrefix(installManifestsPath, "github.com/") {
		if _, err := os.Stat(installManifestsPath); err != nil {
			return fmt.Errorf("manifests not found: %w", err)
		}
		kustomizePath = installManifestsPath
	}

	tmpDir, err := ioutil.TempDir("", namespace)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if !installExport {
		logger.Generatef("generating manifests")
	}
	if kustomizePath == "" {
		err = genInstallManifests(installVersion, namespace, installComponents, tmpDir)
		if err != nil {
			return fmt.Errorf("install failed: %w", err)
		}
		kustomizePath = tmpDir
	}

	manifest := path.Join(tmpDir, fmt.Sprintf("%s.yaml", namespace))
	if err := buildKustomization(kustomizePath, manifest); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	command := fmt.Sprintf("cat %s", manifest)
	if yaml, err := utils.execCommand(ctx, ModeCapture, command); err != nil {
		return fmt.Errorf("install failed: %w", err)
	} else {
		if verbose {
			fmt.Print(yaml)
		} else if installExport {
			fmt.Println("---")
			fmt.Println("# GitOps Toolkit revision", installVersion, time.Now().Format(time.RFC3339))
			fmt.Print(yaml)
			fmt.Println("---")
			return nil
		}
	}
	logger.Successf("manifests build completed")

	logger.Actionf("installing components in %s namespace", namespace)
	applyOutput := ModeStderrOS
	if verbose {
		applyOutput = ModeOS
	}
	dryRun := ""
	if installDryRun {
		dryRun = "--dry-run=client"
		applyOutput = ModeOS
	}
	command = fmt.Sprintf("cat %s | kubectl apply -f- %s", manifest, dryRun)
	if _, err := utils.execCommand(ctx, applyOutput, command); err != nil {
		return fmt.Errorf("install failed")
	}

	if installDryRun {
		logger.Successf("install dry-run finished")
		return nil
	} else {
		logger.Successf("install completed")
	}

	logger.Waitingf("verifying installation")
	for _, deployment := range installComponents {
		command = fmt.Sprintf("kubectl -n %s rollout status deployment %s --timeout=%s",
			namespace, deployment, timeout.String())
		if _, err := utils.execCommand(ctx, applyOutput, command); err != nil {
			return fmt.Errorf("install failed")
		} else {
			logger.Successf("%s ready", deployment)
		}
	}

	logger.Successf("install finished")
	return nil
}

var namespaceTmpl = `---
apiVersion: v1
kind: Namespace
metadata:
  name: {{.Namespace}}
`

var labelsTmpl = `---
apiVersion: builtin
kind: LabelTransformer
metadata:
  name: labels
labels:
  app.kubernetes.io/instance: {{.Namespace}}
  app.kubernetes.io/version: "{{.Version}}"
fieldSpecs:
  - path: metadata/labels
    create: true
`

var kustomizationTmpl = `---
{{- $version := .Version }}
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: {{.Namespace}}
transformers:
  - labels.yaml
resources:
  - namespace.yaml
  - policies.yaml
  - roles
{{- range .Components }}
  - {{.}}.yaml
{{- end }}
`

var kustomizationRolesTmpl = `---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - rbac.yaml
nameSuffix: -{{.Namespace}}
`

func downloadManifests(version string, tmpDir string) error {
	ghURL := "https://github.com/fluxcd/toolkit/releases/latest/download/manifests.tar.gz"
	if strings.HasPrefix(version, "v") {
		ghURL = fmt.Sprintf("https://github.com/fluxcd/toolkit/releases/download/%s/manifests.tar.gz", version)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequest("GET", ghURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for %s, error: %w", ghURL, err)
	}

	// download
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to download artifact from %s, error: %w", ghURL, err)
	}
	defer resp.Body.Close()

	// check response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("faild to download artifact from %s, status: %s", ghURL, resp.Status)
	}

	// extract
	if _, err = untar.Untar(resp.Body, tmpDir); err != nil {
		return fmt.Errorf("faild to untar manifests from %s, error: %w", ghURL, err)
	}

	return nil
}

func genInstallManifests(version string, namespace string, components []string, tmpDir string) error {
	model := struct {
		Version    string
		Namespace  string
		Components []string
	}{
		Version:    version,
		Namespace:  namespace,
		Components: components,
	}

	if err := downloadManifests(version, tmpDir); err != nil {
		return err
	}

	if err := utils.execTemplate(model, namespaceTmpl, path.Join(tmpDir, "namespace.yaml")); err != nil {
		return fmt.Errorf("generate namespace failed: %w", err)
	}

	if err := utils.execTemplate(model, labelsTmpl, path.Join(tmpDir, "labels.yaml")); err != nil {
		return fmt.Errorf("generate labels failed: %w", err)
	}

	if err := utils.execTemplate(model, kustomizationTmpl, path.Join(tmpDir, "kustomization.yaml")); err != nil {
		return fmt.Errorf("generate kustomization failed: %w", err)
	}

	if err := os.MkdirAll(path.Join(tmpDir, "roles"), os.ModePerm); err != nil {
		return fmt.Errorf("generate roles failed: %w", err)
	}

	if err := utils.execTemplate(model, kustomizationRolesTmpl, path.Join(tmpDir, "roles/kustomization.yaml")); err != nil {
		return fmt.Errorf("generate roles failed: %w", err)
	}

	if err := utils.copyFile(filepath.Join(tmpDir, "rbac.yaml"), filepath.Join(tmpDir, "roles/rbac.yaml")); err != nil {
		return fmt.Errorf("generate rbac failed: %w", err)
	}

	return nil
}

func buildKustomization(base, manifests string) error {
	kfile := filepath.Join(base, "kustomization.yaml")

	fs := filesys.MakeFsOnDisk()
	if !fs.Exists(kfile) {
		return fmt.Errorf("%s not found", kfile)
	}

	opt := krusty.MakeDefaultOptions()
	k := krusty.MakeKustomizer(fs, opt)
	m, err := k.Run(base)
	if err != nil {
		return err
	}

	resources, err := m.AsYaml()
	if err != nil {
		return err
	}

	if err := fs.WriteFile(manifests, resources); err != nil {
		return err
	}

	return nil
}
