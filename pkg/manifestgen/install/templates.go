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

package install

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"text/template"
)

var kustomizationTmpl = `---
{{- $eventsAddr := .EventsAddr }}
{{- $watchAllNamespaces := .WatchAllNamespaces }}
{{- $registry := .Registry }}
{{- $logLevel := .LogLevel }}
{{- $clusterDomain := .ClusterDomain }}
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: {{.Namespace}}

transformers:
  - labels.yaml

resources:
  - namespace.yaml
{{- if .NetworkPolicy }}
  - policies.yaml
{{- end }}
  - roles
{{- range .Components }}
  - {{.}}.yaml
{{- end }}

patches:
- path: node-selector.yaml
  target:
    kind: Deployment

patchesJson6902:
{{- range $i, $component := .Components }}
{{- if eq $component "notification-controller" }}
- target:
    group: apps
    version: v1
    kind: Deployment
    name: {{$component}}
  patch: |-
    - op: replace
      path: /spec/template/spec/containers/0/args/0
      value: --watch-all-namespaces={{$watchAllNamespaces}}
    - op: replace
      path: /spec/template/spec/containers/0/args/1
      value: --log-level={{$logLevel}}
{{- else if eq $component "source-controller" }}
- target:
    group: apps
    version: v1
    kind: Deployment
    name: {{$component}}
  patch: |-
    - op: replace
      path: /spec/template/spec/containers/0/args/0
      value: --events-addr={{$eventsAddr}}
    - op: replace
      path: /spec/template/spec/containers/0/args/1
      value: --watch-all-namespaces={{$watchAllNamespaces}}
    - op: replace
      path: /spec/template/spec/containers/0/args/2
      value: --log-level={{$logLevel}}
    - op: replace
      path: /spec/template/spec/containers/0/args/6
      value: --storage-adv-addr=source-controller.$(RUNTIME_NAMESPACE).svc.{{$clusterDomain}}.
{{- else }}
- target:
    group: apps
    version: v1
    kind: Deployment
    name: {{$component}}
  patch: |-
    - op: replace
      path: /spec/template/spec/containers/0/args/0
      value: --events-addr={{$eventsAddr}}
    - op: replace
      path: /spec/template/spec/containers/0/args/1
      value: --watch-all-namespaces={{$watchAllNamespaces}}
    - op: replace
      path: /spec/template/spec/containers/0/args/2
      value: --log-level={{$logLevel}}
{{- end }}
{{- end }}

{{- if $registry }}
images:
{{- range $i, $component := .Components }}
  - name: fluxcd/{{$component}}
    newName: {{$registry}}/{{$component}}
{{- end }}
{{- end }}
`

var kustomizationRolesTmpl = `---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: {{.Namespace}}
resources:
  - rbac.yaml
nameSuffix: -{{.Namespace}}
`

var nodeSelectorTmpl = `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: all
spec:
  template:
    spec:
      nodeSelector:
        kubernetes.io/os: linux
{{- range $k, $v := .AdditionalNodeSelectors }}
        {{$k}}: "{{$v}}"
{{- end }}
{{- if .ImagePullSecret }}
      imagePullSecrets:
       - name: {{.ImagePullSecret}}
{{- end }}
{{ if gt (len .TolerationKeys) 0 }}
      tolerations:
{{- range $i, $key := .TolerationKeys }}
       - key: "{{$key}}"
         operator: "Exists"
{{- end }}
{{- end }}
`

var labelsTmpl = `---
apiVersion: builtin
kind: LabelTransformer
metadata:
  name: labels
labels:
  app.kubernetes.io/instance: {{.Namespace}}
  app.kubernetes.io/version: "{{.Version}}"
  app.kubernetes.io/part-of: flux
fieldSpecs:
  - path: metadata/labels
    create: true
`

var namespaceTmpl = `---
apiVersion: v1
kind: Namespace
metadata:
  name: {{.Namespace}}
  labels:
    pod-security.kubernetes.io/warn: restricted
    pod-security.kubernetes.io/warn-version: latest
`

func execTemplate(obj interface{}, tmpl, filename string) error {
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

func copyFile(src, dst string) error {
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
