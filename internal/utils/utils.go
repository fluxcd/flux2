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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	sigyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta2"
	imageautov1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
	imagereflectv1 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notificationv1 "github.com/fluxcd/notification-controller/api/v1"
	notificationv1b3 "github.com/fluxcd/notification-controller/api/v1beta3"
	"github.com/fluxcd/pkg/apis/meta"
	runclient "github.com/fluxcd/pkg/runtime/client"
	"github.com/fluxcd/pkg/version"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
)

type ExecMode string

const (
	ModeOS       ExecMode = "os.stderr|stdout"
	ModeStderrOS ExecMode = "os.stderr"
	ModeCapture  ExecMode = "capture.stderr|stdout"
)

func ExecKubectlCommand(ctx context.Context, mode ExecMode, kubeConfigPath string, kubeContext string, args ...string) (string, error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	if kubeConfigPath != "" && len(filepath.SplitList(kubeConfigPath)) == 1 {
		args = append(args, "--kubeconfig="+kubeConfigPath)
	}

	if kubeContext != "" {
		args = append(args, "--context="+kubeContext)
	}

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

func KubeConfig(rcg genericclioptions.RESTClientGetter, opts *runclient.Options) (*rest.Config, error) {
	cfg, err := rcg.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("kubernetes configuration load failed: %w", err)
	}

	// avoid throttling request when some Flux CRDs are not registered
	cfg.QPS = opts.QPS
	cfg.Burst = opts.Burst

	return cfg, nil
}

// Create the Scheme, methods for serializing and deserializing API objects
// which can be shared by tests.
func NewScheme() *apiruntime.Scheme {
	scheme := apiruntime.NewScheme()
	_ = apiextensionsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = networkingv1.AddToScheme(scheme)
	_ = sourcev1b2.AddToScheme(scheme)
	_ = sourcev1.AddToScheme(scheme)
	_ = kustomizev1.AddToScheme(scheme)
	_ = helmv2.AddToScheme(scheme)
	_ = notificationv1.AddToScheme(scheme)
	_ = notificationv1b3.AddToScheme(scheme)
	_ = imagereflectv1.AddToScheme(scheme)
	_ = imageautov1.AddToScheme(scheme)
	return scheme
}

func KubeClient(rcg genericclioptions.RESTClientGetter, opts *runclient.Options) (client.WithWatch, error) {
	cfg, err := rcg.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	cfg.QPS = opts.QPS
	cfg.Burst = opts.Burst

	scheme := NewScheme()
	kubeClient, err := client.NewWithWatch(cfg, client.Options{
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

func ContainsItemString(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func ContainsEqualFoldItemString(s []string, e string) (string, bool) {
	for _, a := range s {
		if strings.EqualFold(a, e) {
			return a, true
		}
	}
	return "", false
}

// ParseNamespacedName extracts the NamespacedName of a resource
// based on the '<namespace>/<name>' format
func ParseNamespacedName(input string) types.NamespacedName {
	parts := strings.Split(input, "/")
	if len(parts) == 2 {
		return types.NamespacedName{
			Namespace: parts[0],
			Name:      parts[1],
		}
	}
	return types.NamespacedName{
		Name: input,
	}
}

// ParseObjectKindName extracts the kind and name of a resource
// based on the '<kind>/<name>' format
func ParseObjectKindName(input string) (kind, name string) {
	name = input
	parts := strings.Split(input, "/")
	if len(parts) == 2 {
		kind, name = parts[0], parts[1]
	}
	return kind, name
}

// ParseObjectKindNameNamespace extracts the kind, name and namespace of a resource
// based on the '<kind>/<name>.<namespace>' format
func ParseObjectKindNameNamespace(input string) (kind, name, namespace string) {
	kind, name = ParseObjectKindName(input)

	if nn := strings.Split(name, "."); len(nn) > 1 {
		name = strings.Join(nn[:len(nn)-1], ".")
		namespace = nn[len(nn)-1]
	}

	return kind, name, namespace
}

func MakeDependsOn(deps []string) []meta.NamespacedObjectReference {
	refs := []meta.NamespacedObjectReference{}
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
		refs = append(refs, meta.NamespacedObjectReference{
			Namespace: depNamespace,
			Name:      depName,
		})
	}
	return refs
}

func ValidateComponents(components []string) error {
	defaults := install.MakeDefaultOptions()
	bootstrapAllComponents := append(defaults.Components, defaults.ComponentsExtra...)
	for _, component := range components {
		if !ContainsItemString(bootstrapAllComponents, component) {
			return fmt.Errorf("component %s is not available", component)
		}
	}

	return nil
}

// CompatibleVersion returns if the provided binary version is compatible
// with the given target version. At present, this is true if the target
// version is equal to the MINOR range of the binary, or if the binary
// version is a prerelease.
func CompatibleVersion(binary, target string) bool {
	binSv, err := version.ParseVersion(binary)
	if err != nil {
		return false
	}
	// Assume prerelease builds are compatible.
	if binSv.Prerelease() != "" {
		return true
	}
	targetSv, err := version.ParseVersion(target)
	if err != nil {
		return false
	}
	return binSv.Major() == targetSv.Major() && binSv.Minor() == targetSv.Minor()
}

func ExtractCRDs(inManifestPath, outManifestPath string) error {
	manifests, err := os.ReadFile(inManifestPath)
	if err != nil {
		return err
	}

	crds := ""
	reader := sigyaml.NewYAMLOrJSONDecoder(bytes.NewReader(manifests), 2048)

	for {
		var obj unstructured.Unstructured
		err := reader.Decode(&obj)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if obj.GetKind() == "CustomResourceDefinition" {
			b, err := obj.MarshalJSON()
			if err != nil {
				return err
			}
			y, err := yaml.JSONToYAML(b)
			if err != nil {
				return err
			}
			crds += "---\n" + string(y)
		}
	}

	if crds == "" {
		return fmt.Errorf("no CRDs found in %s", inManifestPath)
	}

	return os.WriteFile(outManifestPath, []byte(crds), os.ModePerm)
}
