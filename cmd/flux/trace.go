/*
Copyright 2021 The Flux authors

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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/oci"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var traceCmd = &cobra.Command{
	Use:   "trace <resource> <name> [<name> ...]",
	Short: "Trace in-cluster objects throughout the GitOps delivery pipeline",
	Long: withPreviewNote(`The trace command shows how one or more objects are managed by Flux,
from which source and revision they come, and what the latest reconciliation status is.

You can also trace multiple objects with different resource kinds using <resource>/<name> multiple times.`),
	Example: `  # Trace a Kubernetes Deployment
  flux trace -n apps deployment my-app

  # Trace a Kubernetes Pod and a config map
  flux trace -n redis pod/redis-master-0 cm/redis

  # Trace a Kubernetes global object
  flux trace namespace redis

  # Trace a Kubernetes custom resource
  flux trace -n redis helmrelease redis
  
  # API Version and Kind can also be specified explicitly
  # Note that either both, kind and api-version, or neither have to be specified.
  flux trace redis --kind=helmrelease --api-version=helm.toolkit.fluxcd.io/v2beta2 -n redis`,
	RunE: traceCmdRun,
}

type traceFlags struct {
	apiVersion string
	kind       string
}

var traceArgs = traceFlags{}

func init() {
	traceCmd.Flags().StringVar(&traceArgs.kind, "kind", "",
		"the Kubernetes object kind, e.g. Deployment'")
	traceCmd.Flags().StringVar(&traceArgs.apiVersion, "api-version", "",
		"the Kubernetes object API version, e.g. 'apps/v1'")
	rootCmd.AddCommand(traceCmd)
}

func traceCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	var objects []*unstructured.Unstructured
	if traceArgs.kind != "" || traceArgs.apiVersion != "" {
		var obj *unstructured.Unstructured
		obj, err = getObjectStatic(ctx, kubeClient, args)
		objects = []*unstructured.Unstructured{obj}
	} else {
		objects, err = getObjectDynamic(args)
	}
	if err != nil {
		return err
	}

	return traceObjects(ctx, kubeClient, objects)
}

func traceObjects(ctx context.Context, kubeClient client.Client, objects []*unstructured.Unstructured) error {
	for i, obj := range objects {
		err := traceObject(ctx, kubeClient, obj)
		if err != nil {
			rootCmd.PrintErrf("failed to trace %v/%v in namespace %v: %v", obj.GetKind(), obj.GetName(), obj.GetNamespace(), err)
		}
		if i < len(objects)-1 {
			rootCmd.Println("---")
		}
	}
	return nil
}

func traceObject(ctx context.Context, kubeClient client.Client, obj *unstructured.Unstructured) error {
	if ks, ok := isOwnerManagedByFlux(ctx, kubeClient, obj, kustomizev1.GroupVersion.Group); ok {
		report, err := traceKustomization(ctx, kubeClient, ks, obj)
		if err != nil {
			return err
		}
		rootCmd.Print(report)
		return nil
	}

	if hr, ok := isOwnerManagedByFlux(ctx, kubeClient, obj, helmv2.GroupVersion.Group); ok {
		report, err := traceHelm(ctx, kubeClient, hr, obj)
		if err != nil {
			return err
		}
		rootCmd.Print(report)
		return nil
	}

	return fmt.Errorf("object not managed by Flux")
}

func getObjectStatic(ctx context.Context, kubeClient client.Client, args []string) (*unstructured.Unstructured, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("object name is required")
	}

	if traceArgs.kind == "" {
		return nil, fmt.Errorf("object kind is required (--kind)")
	}

	if traceArgs.apiVersion == "" {
		return nil, fmt.Errorf("object apiVersion is required (--api-version)")
	}

	gv, err := schema.ParseGroupVersion(traceArgs.apiVersion)
	if err != nil {
		return nil, fmt.Errorf("invaild apiVersion: %w", err)
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    traceArgs.kind,
	})

	objName := types.NamespacedName{
		Namespace: *kubeconfigArgs.Namespace,
		Name:      args[0],
	}

	if err = kubeClient.Get(ctx, objName, obj); err != nil {
		return nil, fmt.Errorf("failed to find object: %w", err)
	}
	return obj, nil
}

func getObjectDynamic(args []string) ([]*unstructured.Unstructured, error) {
	r := resource.NewBuilder(kubeconfigArgs).
		Unstructured().
		NamespaceParam(*kubeconfigArgs.Namespace).DefaultNamespace().
		ResourceTypeOrNameArgs(false, args...).
		ContinueOnError().
		Latest().
		Do()

	if err := r.Err(); err != nil {
		if resource.IsUsageError(err) {
			return nil, fmt.Errorf("either `<resource>/<name>` or `<resource> <name>` is required as an argument")
		}
		return nil, err
	}

	infos, err := r.Infos()
	if err != nil {
		return nil, fmt.Errorf("x: %v", err)
	}
	if len(infos) == 0 {
		return nil, fmt.Errorf("failed to find object: %w", err)
	}

	objects := []*unstructured.Unstructured{}
	for _, info := range infos {
		obj := &unstructured.Unstructured{}
		obj.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(info.Object)
		if err != nil {
			return objects, err
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

func traceKustomization(ctx context.Context, kubeClient client.Client, ksName types.NamespacedName, obj *unstructured.Unstructured) (string, error) {
	ks := &kustomizev1.Kustomization{}
	err := kubeClient.Get(ctx, ksName, ks)
	if err != nil {
		return "", fmt.Errorf("failed to find kustomization: %w", err)
	}
	ksReady := meta.FindStatusCondition(ks.Status.Conditions, fluxmeta.ReadyCondition)

	var gitRepository *sourcev1.GitRepository
	var ociRepository *sourcev1b2.OCIRepository
	var ksRepositoryReady *metav1.Condition
	switch ks.Spec.SourceRef.Kind {
	case sourcev1.GitRepositoryKind:
		gitRepository = &sourcev1.GitRepository{}
		sourceNamespace := ks.Namespace
		if ks.Spec.SourceRef.Namespace != "" {
			sourceNamespace = ks.Spec.SourceRef.Namespace
		}
		err = kubeClient.Get(ctx, types.NamespacedName{
			Namespace: sourceNamespace,
			Name:      ks.Spec.SourceRef.Name,
		}, gitRepository)
		if err != nil {
			return "", fmt.Errorf("failed to find GitRepository: %w", err)
		}
		ksRepositoryReady = meta.FindStatusCondition(gitRepository.Status.Conditions, fluxmeta.ReadyCondition)
	case sourcev1b2.OCIRepositoryKind:
		ociRepository = &sourcev1b2.OCIRepository{}
		sourceNamespace := ks.Namespace
		if ks.Spec.SourceRef.Namespace != "" {
			sourceNamespace = ks.Spec.SourceRef.Namespace
		}
		err = kubeClient.Get(ctx, types.NamespacedName{
			Namespace: sourceNamespace,
			Name:      ks.Spec.SourceRef.Name,
		}, ociRepository)
		if err != nil {
			return "", fmt.Errorf("failed to find OCIRepository: %w", err)
		}
		ksRepositoryReady = meta.FindStatusCondition(ociRepository.Status.Conditions, fluxmeta.ReadyCondition)
	}

	var traceTmpl = `
Object:          {{.ObjectName}}
{{- if .ObjectNamespace }}
Namespace:       {{.ObjectNamespace}}
{{- end }}
Status:          Managed by Flux
{{- if .Kustomization }}
---
Kustomization:   {{.Kustomization.Name}}
Namespace:       {{.Kustomization.Namespace}}
{{- if .Kustomization.Spec.TargetNamespace }}
Target:          {{.Kustomization.Spec.TargetNamespace}}
{{- end }}
Path:            {{.Kustomization.Spec.Path}}
Revision:        {{.Kustomization.Status.LastAppliedRevision}}
{{- if .KustomizationReady }}
Status:          Last reconciled at {{.KustomizationReady.LastTransitionTime}}
Message:         {{.KustomizationReady.Message}}
{{- else }}
Status:          Unknown
{{- end }}
{{- end }}
{{- if .GitRepository }}
---
GitRepository:   {{.GitRepository.Name}}
Namespace:       {{.GitRepository.Namespace}}
URL:             {{.GitRepository.Spec.URL}}
{{- if .GitRepository.Spec.Reference }}
{{- if .GitRepository.Spec.Reference.Tag }}
Tag:             {{.GitRepository.Spec.Reference.Tag}}
{{- else if .GitRepository.Spec.Reference.SemVer }}
Tag:             {{.GitRepository.Spec.Reference.SemVer}}
{{- else if .GitRepository.Spec.Reference.Branch }}
Branch:          {{.GitRepository.Spec.Reference.Branch}}
{{- end }}
{{- end }}
{{- if .GitRepository.Status.Artifact }}
Revision:        {{.GitRepository.Status.Artifact.Revision}}
{{- end }}
{{- if .RepositoryReady }}
{{- if eq .RepositoryReady.Status "False" }}
Status:          Last reconciliation failed at {{.RepositoryReady.LastTransitionTime}}
{{- else }}
Status:          Last reconciled at {{.RepositoryReady.LastTransitionTime}}
{{- end }}
Message:         {{.RepositoryReady.Message}}
{{- else }}
Status:          Unknown
{{- end }}
{{- end }}
{{- if .OCIRepository }}
---
OCIRepository:   {{.OCIRepository.Name}}
Namespace:       {{.OCIRepository.Namespace}}
URL:             {{.OCIRepository.Spec.URL}}
{{- if .OCIRepository.Spec.Reference }}
{{- if .OCIRepository.Spec.Reference.Tag }}
Tag:             {{.OCIRepository.Spec.Reference.Tag}}
{{- else if .OCIRepository.Spec.Reference.SemVer }}
Tag:             {{.OCIRepository.Spec.Reference.SemVer}}
{{- else if .OCIRepository.Spec.Reference.Digest }}
Digest:          {{.OCIRepository.Spec.Reference.Digest}}
{{- end }}
{{- end }}
{{- if .OCIRepository.Status.Artifact }}
Revision:        {{.OCIRepository.Status.Artifact.Revision}}
{{- if .OCIRepository.Status.Artifact.Metadata }}
{{- $metadata := .OCIRepository.Status.Artifact.Metadata }}
{{- range $k, $v := .Annotations }}
{{ with (index $metadata $v) }}{{ $k }}{{ . }}{{ end }}
{{- end }}
{{- end }}
{{- end }}
{{- if .RepositoryReady }}
{{- if eq .RepositoryReady.Status "False" }}
Status:          Last reconciliation failed at {{.RepositoryReady.LastTransitionTime}}
{{- else }}
Status:          Last reconciled at {{.RepositoryReady.LastTransitionTime}}
{{- end }}
Message:         {{.RepositoryReady.Message}}
{{- else }}
Status:          Unknown
{{- end }}
{{- end }}
`

	traceResult := struct {
		ObjectName         string
		ObjectNamespace    string
		Kustomization      *kustomizev1.Kustomization
		KustomizationReady *metav1.Condition
		GitRepository      *sourcev1.GitRepository
		OCIRepository      *sourcev1b2.OCIRepository
		RepositoryReady    *metav1.Condition
		Annotations        map[string]string
	}{
		ObjectName:         obj.GetKind() + "/" + obj.GetName(),
		ObjectNamespace:    obj.GetNamespace(),
		Kustomization:      ks,
		KustomizationReady: ksReady,
		GitRepository:      gitRepository,
		OCIRepository:      ociRepository,
		RepositoryReady:    ksRepositoryReady,
		Annotations:        map[string]string{"Origin Source:   ": oci.SourceAnnotation, "Origin Revision: ": oci.RevisionAnnotation},
	}

	t, err := template.New("tmpl").Parse(traceTmpl)
	if err != nil {
		return "", err
	}

	var data bytes.Buffer
	writer := bufio.NewWriter(&data)
	if err := t.Execute(writer, traceResult); err != nil {
		return "", err
	}

	if err := writer.Flush(); err != nil {
		return "", err
	}

	return data.String(), nil
}

func traceHelm(ctx context.Context, kubeClient client.Client, hrName types.NamespacedName, obj *unstructured.Unstructured) (string, error) {
	hr := &helmv2.HelmRelease{}
	err := kubeClient.Get(ctx, hrName, hr)
	if err != nil {
		return "", fmt.Errorf("failed to find HelmRelease: %w", err)
	}
	hrReady := meta.FindStatusCondition(hr.Status.Conditions, fluxmeta.ReadyCondition)

	var hrChart *sourcev1b2.HelmChart
	var hrChartReady *metav1.Condition
	if chart := hr.Status.HelmChart; chart != "" {
		hrChart = &sourcev1b2.HelmChart{}
		err = kubeClient.Get(ctx, utils.ParseNamespacedName(chart), hrChart)
		if err != nil {
			return "", fmt.Errorf("failed to find HelmChart: %w", err)
		}
		hrChartReady = meta.FindStatusCondition(hrChart.Status.Conditions, fluxmeta.ReadyCondition)
	}

	var hrGitRepository *sourcev1.GitRepository
	var hrGitRepositoryReady *metav1.Condition
	if hr.Spec.Chart.Spec.SourceRef.Kind == sourcev1.GitRepositoryKind {
		hrGitRepository = &sourcev1.GitRepository{}
		sourceNamespace := hr.Namespace
		if hr.Spec.Chart.Spec.SourceRef.Namespace != "" {
			sourceNamespace = hr.Spec.Chart.Spec.SourceRef.Namespace
		}
		err = kubeClient.Get(ctx, types.NamespacedName{
			Namespace: sourceNamespace,
			Name:      hr.Spec.Chart.Spec.SourceRef.Name,
		}, hrGitRepository)
		if err != nil {
			return "", fmt.Errorf("failed to find GitRepository: %w", err)
		}
		hrGitRepositoryReady = meta.FindStatusCondition(hrGitRepository.Status.Conditions, fluxmeta.ReadyCondition)
	}

	var hrHelmRepository *sourcev1b2.HelmRepository
	var hrHelmRepositoryReady *metav1.Condition
	if hr.Spec.Chart.Spec.SourceRef.Kind == sourcev1b2.HelmRepositoryKind {
		hrHelmRepository = &sourcev1b2.HelmRepository{}
		sourceNamespace := hr.Namespace
		if hr.Spec.Chart.Spec.SourceRef.Namespace != "" {
			sourceNamespace = hr.Spec.Chart.Spec.SourceRef.Namespace
		}
		err = kubeClient.Get(ctx, types.NamespacedName{
			Namespace: sourceNamespace,
			Name:      hr.Spec.Chart.Spec.SourceRef.Name,
		}, hrHelmRepository)
		if err != nil {
			return "", fmt.Errorf("failed to find HelmRepository: %w", err)
		}
		hrHelmRepositoryReady = meta.FindStatusCondition(hrHelmRepository.Status.Conditions, fluxmeta.ReadyCondition)
	}

	var traceTmpl = `
Object:         {{.ObjectName}}
{{- if .ObjectNamespace }}
Namespace:      {{.ObjectNamespace}}
{{- end }}
Status:         Managed by Flux
{{- if .HelmRelease }}
---
HelmRelease:    {{.HelmRelease.Name}}
Namespace:      {{.HelmRelease.Namespace}}
{{- if .HelmRelease.Spec.TargetNamespace }}
Target:         {{.HelmRelease.Spec.TargetNamespace}}
{{- end }}
Revision:       {{.HelmRelease.Status.LastAppliedRevision}}
{{- if .HelmReleaseReady }}
Status:         Last reconciled at {{.HelmReleaseReady.LastTransitionTime}}
Message:        {{.HelmReleaseReady.Message}}
{{- else }}
Status:         Unknown
{{- end }}
{{- end }}
{{- if .HelmChart }}
---
HelmChart:      {{.HelmChart.Name}}
Namespace:      {{.HelmChart.Namespace}}
Chart:          {{.HelmChart.Spec.Chart}}
Version:        {{.HelmChart.Spec.Version}}
{{- if .HelmChart.Status.Artifact }}
Revision:       {{.HelmChart.Status.Artifact.Revision}}
{{- end }}
{{- if .HelmChartReady }}
Status:         Last reconciled at {{.HelmChartReady.LastTransitionTime}}
Message:        {{.HelmChartReady.Message}}
{{- else }}
Status:         Unknown
{{- end }}
{{- end }}
{{- if .HelmRepository }}
---
HelmRepository: {{.HelmRepository.Name}}
Namespace:      {{.HelmRepository.Namespace}}
URL:            {{.HelmRepository.Spec.URL}}
{{- if .HelmRepository.Status.Artifact }}
Revision:       {{.HelmRepository.Status.Artifact.Revision}}
{{- end }}
{{- if .HelmRepositoryReady }}
Status:         Last reconciled at {{.HelmRepositoryReady.LastTransitionTime}}
Message:        {{.HelmRepositoryReady.Message}}
{{- else }}
Status:         Unknown
{{- end }}
{{- end }}
{{- if .GitRepository }}
---
GitRepository: {{.GitRepository.Name}}
Namespace:     {{.GitRepository.Namespace}}
URL:           {{.GitRepository.Spec.URL}}
{{- if .GitRepository.Spec.Reference }}
{{- if .GitRepository.Spec.Reference.Tag }}
Tag:           {{.GitRepository.Spec.Reference.Tag}}
{{- else if .GitRepository.Spec.Reference.SemVer }}
Tag:           {{.GitRepository.Spec.Reference.SemVer}}
{{- else if .GitRepository.Spec.Reference.Branch }}
Branch:        {{.GitRepository.Spec.Reference.Branch}}
{{- end }}
{{- end }}
{{- if .GitRepository.Status.Artifact }}
Revision:      {{.GitRepository.Status.Artifact.Revision}}
{{- end }}
{{- if .GitRepositoryReady }}
{{- if eq .GitRepositoryReady.Status "False" }}
Status:        Last reconciliation failed at {{.GitRepositoryReady.LastTransitionTime}}
{{- else }}
Status:        Last reconciled at {{.GitRepositoryReady.LastTransitionTime}}
{{- end }}
Message:       {{.GitRepositoryReady.Message}}
{{- else }}
Status:        Unknown
{{- end }}
{{- end }}
`

	traceResult := struct {
		ObjectName          string
		ObjectNamespace     string
		HelmRelease         *helmv2.HelmRelease
		HelmReleaseReady    *metav1.Condition
		HelmChart           *sourcev1b2.HelmChart
		HelmChartReady      *metav1.Condition
		GitRepository       *sourcev1.GitRepository
		GitRepositoryReady  *metav1.Condition
		HelmRepository      *sourcev1b2.HelmRepository
		HelmRepositoryReady *metav1.Condition
	}{
		ObjectName:          obj.GetKind() + "/" + obj.GetName(),
		ObjectNamespace:     obj.GetNamespace(),
		HelmRelease:         hr,
		HelmReleaseReady:    hrReady,
		HelmChart:           hrChart,
		HelmChartReady:      hrChartReady,
		GitRepository:       hrGitRepository,
		GitRepositoryReady:  hrGitRepositoryReady,
		HelmRepository:      hrHelmRepository,
		HelmRepositoryReady: hrHelmRepositoryReady,
	}

	t, err := template.New("tmpl").Parse(traceTmpl)
	if err != nil {
		return "", err
	}

	var data bytes.Buffer
	writer := bufio.NewWriter(&data)
	if err := t.Execute(writer, traceResult); err != nil {
		return "", err
	}

	if err := writer.Flush(); err != nil {
		return "", err
	}

	return data.String(), nil
}

func isManagedByFlux(obj *unstructured.Unstructured, group string) (types.NamespacedName, bool) {
	nameKey := fmt.Sprintf("%s/name", group)
	namespaceKey := fmt.Sprintf("%s/namespace", group)
	namespacedName := types.NamespacedName{}

	for k, v := range obj.GetLabels() {
		if k == nameKey {
			namespacedName.Name = v
		}
		if k == namespaceKey {
			namespacedName.Namespace = v
		}
	}

	if namespacedName.Name == "" {
		return namespacedName, false
	}
	return namespacedName, true
}

func isOwnerManagedByFlux(ctx context.Context, kubeClient client.Client, obj *unstructured.Unstructured, group string) (types.NamespacedName, bool) {
	if n, ok := isManagedByFlux(obj, group); ok {
		return n, true
	}

	namespacedName := types.NamespacedName{}
	for _, reference := range obj.GetOwnerReferences() {
		owner := &unstructured.Unstructured{}
		gv, err := schema.ParseGroupVersion(reference.APIVersion)
		if err != nil {
			return namespacedName, false
		}

		owner.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   gv.Group,
			Version: gv.Version,
			Kind:    reference.Kind,
		})

		ownerName := types.NamespacedName{
			Namespace: obj.GetNamespace(),
			Name:      reference.Name,
		}

		err = kubeClient.Get(ctx, ownerName, owner)
		if err != nil {
			return namespacedName, false
		}

		if n, ok := isManagedByFlux(owner, group); ok {
			return n, true
		}

		if len(owner.GetOwnerReferences()) > 0 {
			return isOwnerManagedByFlux(ctx, kubeClient, owner, group)
		}
	}

	return namespacedName, false
}
