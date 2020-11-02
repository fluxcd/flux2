/*
Copyright 2020 The Flux authors

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

package utils

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/api/konfig"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
	"github.com/fluxcd/pkg/runtime/dependency"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/olekukonko/tablewriter"
)

type Utils struct {
}

type ExecMode string

const (
	ModeOS       ExecMode = "os.stderr|stdout"
	ModeStderrOS ExecMode = "os.stderr"
	ModeCapture  ExecMode = "capture.stderr|stdout"
)

func ExecKubectlCommand(ctx context.Context, mode ExecMode, args ...string) (string, error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	c := exec.CommandContext(ctx, "kubectl", args...)

	if mode == ModeStderrOS {
		c.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	}
	if mode == ModeOS {
		c.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		c.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	}

	if mode == ModeStderrOS || mode == ModeOS {
		if err := c.Run(); err != nil {
			return "", err
		} else {
			return "", nil
		}
	}

	if mode == ModeCapture {
		c.Stdout = &stdoutBuf
		c.Stderr = &stderrBuf
		if err := c.Run(); err != nil {
			return stderrBuf.String(), err
		} else {
			return stdoutBuf.String(), nil
		}
	}

	return "", nil
}

func ExecTemplate(obj interface{}, tmpl, filename string) error {
	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return err
	}

	var data bytes.Buffer
	writer := bufio.NewWriter(&data)
	if err := t.Execute(writer, obj); err != nil {
		return err
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.WriteString(file, data.String())
	if err != nil {
		return err
	}

	return file.Sync()
}

func KubeClient(kubeConfigPath string, kubeContext string) (client.Client, error) {
	configFiles := SplitKubeConfigPath(kubeConfigPath)
	configOverrides := clientcmd.ConfigOverrides{}

	if len(kubeContext) > 0 {
		configOverrides.CurrentContext = kubeContext
	}

	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{Precedence: configFiles},
		&configOverrides,
	).ClientConfig()

	if err != nil {
		return nil, fmt.Errorf("kubernetes client initialization failed: %w", err)
	}

	scheme := apiruntime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)
	_ = sourcev1.AddToScheme(scheme)
	_ = kustomizev1.AddToScheme(scheme)
	_ = helmv2.AddToScheme(scheme)
	_ = notificationv1.AddToScheme(scheme)

	kubeClient, err := client.New(cfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("kubernetes client initialization failed: %w", err)
	}

	return kubeClient, nil
}

// SplitKubeConfigPath splits the given KUBECONFIG path based on the runtime OS
// target.
//
// Ref: https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/#the-kubeconfig-environment-variable
func SplitKubeConfigPath(path string) []string {
	var sep string
	switch runtime.GOOS {
	case "windows":
		sep = ";"
	default:
		sep = ":"
	}
	return strings.Split(path, sep)
}

func WriteFile(content, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.WriteString(file, content)
	if err != nil {
		return err
	}

	return file.Sync()
}

func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func ContainsItemString(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func ParseObjectKindName(input string) (string, string) {
	kind := ""
	name := input
	parts := strings.Split(input, "/")
	if len(parts) == 2 {
		kind, name = parts[0], parts[1]
	}

	return kind, name
}

func MakeDependsOn(deps []string) []dependency.CrossNamespaceDependencyReference {
	refs := []dependency.CrossNamespaceDependencyReference{}
	for _, dep := range deps {
		parts := strings.Split(dep, "/")
		depNamespace := ""
		depName := ""
		if len(parts) > 1 {
			depNamespace = parts[0]
			depName = parts[1]
		} else {
			depName = parts[0]
		}
		refs = append(refs, dependency.CrossNamespaceDependencyReference{
			Namespace: depNamespace,
			Name:      depName,
		})
	}
	return refs
}

// GenerateKustomizationYaml is the equivalent of running
// 'kustomize create --autodetect' in the specified dir
func GenerateKustomizationYaml(dirPath string) error {
	fs := filesys.MakeFsOnDisk()
	kfile := filepath.Join(dirPath, "kustomization.yaml")

	scan := func(base string) ([]string, error) {
		var paths []string
		uf := kunstruct.NewKunstructuredFactoryImpl()
		err := fs.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path == base {
				return nil
			}
			if info.IsDir() {
				// If a sub-directory contains an existing kustomization file add the
				// directory as a resource and do not decend into it.
				for _, kfilename := range konfig.RecognizedKustomizationFileNames() {
					if fs.Exists(filepath.Join(path, kfilename)) {
						paths = append(paths, path)
						return filepath.SkipDir
					}
				}
				return nil
			}
			fContents, err := fs.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := uf.SliceFromBytes(fContents); err != nil {
				return nil
			}
			paths = append(paths, path)
			return nil
		})
		return paths, err
	}

	if _, err := os.Stat(kfile); err != nil {
		abs, err := filepath.Abs(dirPath)
		if err != nil {
			return err
		}

		files, err := scan(abs)
		if err != nil {
			return err
		}

		f, err := fs.Create(kfile)
		if err != nil {
			return err
		}
		f.Close()

		kus := kustypes.Kustomization{
			TypeMeta: kustypes.TypeMeta{
				APIVersion: kustypes.KustomizationVersion,
				Kind:       kustypes.KustomizationKind,
			},
		}

		var resources []string
		for _, file := range files {
			resources = append(resources, strings.Replace(file, abs, ".", 1))
		}

		kus.Resources = resources
		kd, err := yaml.Marshal(kus)
		if err != nil {
			return err
		}

		return ioutil.WriteFile(kfile, kd, os.ModePerm)
	}
	return nil
}

func PrintTable(writer io.Writer, header []string, rows [][]string) {
	table := tablewriter.NewWriter(writer)
	table.SetHeader(header)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)
	table.AppendBulk(rows)
	table.Render()
}
