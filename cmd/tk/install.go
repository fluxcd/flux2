package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the toolkit components",
	Long: `
The install command deploys the toolkit components
on the configured Kubernetes cluster in ~/.kube/config`,
	Example: `  install --version=master --namespace=gitops-systems`,
	RunE:    installCmdRun,
}

var (
	installDryRun        bool
	installManifestsPath string
	installVersion       string
)

func init() {
	installCmd.Flags().BoolVarP(&installDryRun, "dry-run", "", false,
		"only print the object that would be applied")
	installCmd.Flags().StringVarP(&installVersion, "version", "v", "master",
		"toolkit tag or branch")
	installCmd.Flags().StringVarP(&installManifestsPath, "manifests", "", "",
		"path to the manifest directory, dev only")
	rootCmd.AddCommand(installCmd)
}

func installCmdRun(cmd *cobra.Command, args []string) error {
	kustomizePath := ""
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

	if kustomizePath == "" {
		err = generateInstall(tmpDir)
		if err != nil {
			return err
		}
		kustomizePath = tmpDir
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dryRun := ""
	if installDryRun {
		dryRun = "--dry-run=client"
	}
	command := fmt.Sprintf("kustomize build %s | kubectl apply -f- %s",
		kustomizePath, dryRun)
	c := exec.CommandContext(ctx, "/bin/sh", "-c", command)

	var stdoutBuf, stderrBuf bytes.Buffer
	c.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	c.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	logAction("installing components in %s namespace", namespace)
	err = c.Run()
	if err != nil {
		logFailure("install failed")
		os.Exit(1)
	}

	if installDryRun {
		logSuccess("install dry-run finished")
		return nil
	}

	logAction("verifying installation")
	for _, deployment := range []string{"source-controller", "kustomize-controller"} {
		command = fmt.Sprintf("kubectl -n %s rollout status deployment %s --timeout=%s",
			namespace, deployment, timeout.String())
		c = exec.CommandContext(ctx, "/bin/sh", "-c", command)
		c.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		c.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		err := c.Run()
		if err != nil {
			logFailure("install failed")
			os.Exit(1)
		}
	}

	logSuccess("install finished")
	return nil
}

var namespaceTmpl = `---
apiVersion: v1
kind: Namespace
metadata:
  name: {{.}}
`

var labelsTmpl = `---
apiVersion: builtin
kind: LabelTransformer
metadata:
  name: labels
labels:
  app.kubernetes.io/instance: {{.}}
fieldSpecs:
  - path: metadata/labels
    create: true
`

var kustomizationTmpl = `---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: {{.Namespace}}
transformers:
  - labels.yaml
resources:
  - namespace.yaml
  - roles
  - github.com/fluxcd/toolkit/manifests/bases/source-controller?ref={{.Version}}
  - github.com/fluxcd/toolkit/manifests/bases/kustomize-controller?ref={{.Version}}
  - github.com/fluxcd/toolkit/manifests/policies?ref={{.Version}}
`

var kustomizationRolesTmpl = `---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - github.com/fluxcd/toolkit/manifests/rbac?ref={{.Version}}
nameSuffix: -{{.Namespace}}
`

func generateInstall(tmpDir string) error {
	logAction("generating install manifests for %s namespace", namespace)

	nst, err := template.New("tmpl").Parse(namespaceTmpl)
	if err != nil {
		return fmt.Errorf("template parse error: %w", err)
	}

	ns, err := execTemplate(nst, namespace)
	if err != nil {
		return err
	}

	if err := writeToFile(path.Join(tmpDir, "namespace.yaml"), ns); err != nil {
		return err
	}

	labelst, err := template.New("tmpl").Parse(labelsTmpl)
	if err != nil {
		return fmt.Errorf("template parse error: %w", err)
	}

	labels, err := execTemplate(labelst, namespace)
	if err != nil {
		return err
	}

	if err := writeToFile(path.Join(tmpDir, "labels.yaml"), labels); err != nil {
		return err
	}

	model := struct {
		Version   string
		Namespace string
	}{
		Version:   installVersion,
		Namespace: namespace,
	}

	kt, err := template.New("tmpl").Parse(kustomizationTmpl)
	if err != nil {
		return fmt.Errorf("template parse error: %w", err)
	}

	k, err := execTemplate(kt, model)
	if err != nil {
		return err
	}

	if err := writeToFile(path.Join(tmpDir, "kustomization.yaml"), k); err != nil {
		return err
	}

	krt, err := template.New("tmpl").Parse(kustomizationRolesTmpl)
	if err != nil {
		return fmt.Errorf("template parse error: %w", err)
	}

	kr, err := execTemplate(krt, model)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(path.Join(tmpDir, "roles"), os.ModePerm); err != nil {
		return err
	}

	if err := writeToFile(path.Join(tmpDir, "roles/kustomization.yaml"), kr); err != nil {
		return err
	}

	return nil
}

func execTemplate(t *template.Template, obj interface{}) (string, error) {
	var data bytes.Buffer
	writer := bufio.NewWriter(&data)
	if err := t.Execute(writer, obj); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return "", fmt.Errorf("template flush failed: %w", err)
	}
	return data.String(), nil
}

func writeToFile(filename string, data string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.WriteString(file, data)
	if err != nil {
		return err
	}
	return file.Sync()
}
