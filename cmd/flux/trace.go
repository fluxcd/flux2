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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/flux2/internal/utils"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

var traceCmd = &cobra.Command{
	Use:   "trace [name]",
	Short: "trace an in-cluster object throughout the GitOps delivery pipeline",
	Long: `The trace command shows how an object is managed by Flux,
from which source and revision it comes, and what's the latest reconciliation status.'`,
	Example: `  # Trace a Kubernetes Deployment
  flux trace my-app --kind=deployment --api-version=apps/v1 --namespace=apps

  # Trace a Kubernetes Pod
  flux trace redis-master-0 --kind=pod --api-version=v1 -n redis

  # Trace a Kubernetes global object
  flux trace redis --kind=namespace --api-version=v1

  # Trace a Kubernetes custom resource
  flux trace redis --kind=helmrelease --api-version=helm.toolkit.fluxcd.io/v2beta1 -n redis`,
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
	if len(args) < 1 {
		return fmt.Errorf("object name is required")
	}
	name := args[0]

	if traceArgs.kind == "" {
		return fmt.Errorf("object kind is required (--kind)")
	}

	if traceArgs.apiVersion == "" {
		return fmt.Errorf("object apiVersion is required (--api-version)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(rootArgs.kubeconfig, rootArgs.kubecontext)
	if err != nil {
		return err
	}

	gv, err := schema.ParseGroupVersion(traceArgs.apiVersion)
	if err != nil {
		return fmt.Errorf("invaild apiVersion: %w", err)
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    traceArgs.kind,
	})

	objName := types.NamespacedName{
		Namespace: rootArgs.namespace,
		Name:      name,
	}

	err = kubeClient.Get(ctx, objName, obj)
	if err != nil {
		return fmt.Errorf("failed to find object: %w", err)
	}

	if ks, ok := isOwnerManagedByFlux(ctx, kubeClient, obj, kustomizev1.GroupVersion.Group); ok {
		report, err := traceKustomization(ctx, kubeClient, ks, obj)
		if err != nil {
			return err
		}
		fmt.Println(report)
		return nil
	}

	if hr, ok := isOwnerManagedByFlux(ctx, kubeClient, obj, helmv2.GroupVersion.Group); ok {
		report, err := traceHelm(ctx, kubeClient, hr, obj)
		if err != nil {
			return err
		}
		fmt.Println(report)
		return nil
	}

	return fmt.Errorf("object not managed by Flux")
}

func traceKustomization(ctx context.Context, kubeClient client.Client, ksName types.NamespacedName, obj *unstructured.Unstructured) (string, error) {
	ks := &kustomizev1.Kustomization{}
	ksReady := &metav1.Condition{}
	err := kubeClient.Get(ctx, ksName, ks)
	if err != nil {
		return "", fmt.Errorf("failed to find kustomization: %w", err)
	}
	ksReady = meta.FindStatusCondition(ks.Status.Conditions, fluxmeta.ReadyCondition)

	var ksRepository *sourcev1.GitRepository
	var ksRepositoryReady *metav1.Condition
	if ks.Spec.SourceRef.Kind == sourcev1.GitRepositoryKind {
		ksRepository = &sourcev1.GitRepository{}
		sourceNamespace := ks.Namespace
		if ks.Spec.SourceRef.Namespace != "" {
			sourceNamespace = ks.Spec.SourceRef.Namespace
		}
		err = kubeClient.Get(ctx, types.NamespacedName{
			Namespace: sourceNamespace,
			Name:      ks.Spec.SourceRef.Name,
		}, ksRepository)
		if err != nil {
			return "", fmt.Errorf("failed to find GitRepository: %w", err)
		}
		ksRepositoryReady = meta.FindStatusCondition(ksRepository.Status.Conditions, fluxmeta.ReadyCondition)
	}

	var traceTmpl = `
Object:        {{.ObjectName}}
{{- if .ObjectNamespace }}
Namespace:     {{.ObjectNamespace}}
{{- end }}
Status:        Managed by Flux
{{- if .Kustomization }}
---
Kustomization: {{.Kustomization.Name}}
Namespace:     {{.Kustomization.Namespace}}
{{- if .Kustomization.Spec.TargetNamespace }}
Target:        {{.Kustomization.Spec.TargetNamespace}}
{{- end }}
Path:          {{.Kustomization.Spec.Path}}
Revision:      {{.Kustomization.Status.LastAppliedRevision}}
{{- if .KustomizationReady }}
Status:        Last reconciled at {{.KustomizationReady.LastTransitionTime}}
Message:       {{.KustomizationReady.Message}}
{{- else }}
Status:        Unknown
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
		ObjectName         string
		ObjectNamespace    string
		Kustomization      *kustomizev1.Kustomization
		KustomizationReady *metav1.Condition
		GitRepository      *sourcev1.GitRepository
		GitRepositoryReady *metav1.Condition
	}{
		ObjectName:         obj.GetKind() + "/" + obj.GetName(),
		ObjectNamespace:    obj.GetNamespace(),
		Kustomization:      ks,
		KustomizationReady: ksReady,
		GitRepository:      ksRepository,
		GitRepositoryReady: ksRepositoryReady,
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
	hrReady := &metav1.Condition{}
	err := kubeClient.Get(ctx, hrName, hr)
	if err != nil {
		return "", fmt.Errorf("failed to find HelmRelease: %w", err)
	}
	hrReady = meta.FindStatusCondition(hr.Status.Conditions, fluxmeta.ReadyCondition)

	var hrChart *sourcev1.HelmChart
	var hrChartReady *metav1.Condition
	if chart := hr.Status.HelmChart; chart != "" {
		hrChart = &sourcev1.HelmChart{}
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

	var hrHelmRepository *sourcev1.HelmRepository
	var hrHelmRepositoryReady *metav1.Condition
	if hr.Spec.Chart.Spec.SourceRef.Kind == sourcev1.HelmRepositoryKind {
		hrHelmRepository = &sourcev1.HelmRepository{}
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
{{- if .GitRepository.Spec.Reference.Tag }}
Tag:           {{.GitRepository.Spec.Reference.Tag}}
{{- else if .GitRepository.Spec.Reference.SemVer }}
Tag:           {{.GitRepository.Spec.Reference.SemVer}}
{{- else if .GitRepository.Spec.Reference.Branch }}
Branch:        {{.GitRepository.Spec.Reference.Branch}}
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
		HelmChart           *sourcev1.HelmChart
		HelmChartReady      *metav1.Condition
		GitRepository       *sourcev1.GitRepository
		GitRepositoryReady  *metav1.Condition
		HelmRepository      *sourcev1.HelmRepository
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
