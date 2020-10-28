package sync

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

const (
	bootstrapSourceManifest        = "toolkit-source.yaml"
	bootstrapKustomizationManifest = "toolkit-kustomization.yaml"
)

func Generate(options Options) ([]map[string]string, error) {
	files := []map[string]string{}

	gvk := sourcev1.GroupVersion.WithKind(sourcev1.GitRepositoryKind)
	gitRepository := sourcev1.GitRepository{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      options.Name,
			Namespace: options.Namespace,
		},
		Spec: sourcev1.GitRepositorySpec{
			URL: options.URL,
			Interval: metav1.Duration{
				Duration: options.Interval,
			},
			Reference: &sourcev1.GitRepositoryRef{
				Branch: options.Branch,
			},
			SecretRef: &corev1.LocalObjectReference{
				Name: options.Name,
			},
		},
	}

	gitData, err := yaml.Marshal(gitRepository)
	if err != nil {
		return nil, err
	}

	files = append(files, map[string]string{"file_path": filepath.Join(options.TargetPath, options.Namespace, bootstrapSourceManifest), "content": string(gitData)})

	gvk = kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind)
	kustomization := kustomizev1.Kustomization{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      options.Name,
			Namespace: options.Namespace,
		},
		Spec: kustomizev1.KustomizationSpec{
			Interval: metav1.Duration{
				Duration: 10 * time.Minute,
			},
			Path:  fmt.Sprintf("./%s", strings.TrimPrefix(options.TargetPath, "./")),
			Prune: true,
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: sourcev1.GitRepositoryKind,
				Name: options.Name,
			},
			Validation: "client",
		},
	}

	ksData, err := yaml.Marshal(kustomization)
	if err != nil {
		return nil, err
	}

	files = append(files, map[string]string{"file_path": filepath.Join(options.TargetPath, options.Namespace, bootstrapKustomizationManifest), "content": string(ksData)})

	return files, nil
}
