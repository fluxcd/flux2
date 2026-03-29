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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/kustomize"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestTrimSopsData(t *testing.T) {
	testCases := []struct {
		name     string
		yamlStr  string
		expected string
	}{
		{
			name: "secret with sops token",
			yamlStr: `apiVersion: v1
kind: Secret
metadata:
  name: my-secret
type: Opaque
data:
  token: |
    ewoJImRhdGEiOiAiRU5DW0FFUzI1Nl9HQ00sZGF0YTpvQmU1UGxQbWZRQ1VVYzRzcUtJbW
    p3PT0saXY6TUxMRVcxNVFDOWtSZFZWYWdKbnpMQ1NrMHhaR1dJcEFlVGZIenl4VDEwZz0s
    dGFnOkszR2tCQ0dTK3V0NFRwazZuZGIwQ0E9PSx0eXBlOnN0cl0iLAoJInNvcHMiOiB7Cg
    kJImttcyI6IG51bGwsCgkJImdjcF9rbXMiOiBudWxsLAoJCSJhenVyZV9rdiI6IG51bGws
    CgkJImhjX3ZhdWx0IjogbnVsbCwKCQkiYWdlIjogWwoJCQl7CgkJCQkicmVjaXBpZW50Ij
    ogImFnZTEwbGEyZ2Uwd3R2eDNxcjdkYXRxZjdyczR5bmd4c3pkYWw5MjdmczlydWthbXI4
    dTJwc2hzdnR6N2NlIiwKCQkJCSJlbmMiOiAiLS0tLS1CRUdJTiBBR0UgRU5DUllQVEVEIE
    ZJTEUtLS0tLVxuWVdkbExXVnVZM0o1Y0hScGIyNHViM0puTDNZeENpMCtJRmd5TlRVeE9T
    QTFMMlJwWkhScksxRlNWbVlyZDFWYVxuWTBoeFdGUXpTREJzVDFrM1dqTnRZbVUxUW1saW
    FESnljWGxOQ25GMVlqZE5PVGhWYlZOdk1HOXJOUzlaVVhad1xuTW5WMGJuUlVNR050ZWpG
    UGJ6TTRVMlV6V2tzemVWa0tMUzB0SUdKNlVHaHhNVVYzWW1WSlRIbEpTVUpwUlZSWlxuVm
    pkMFJWUmFkVTh3ZWt4WFRISXJZVXBsWWtOMmFFRUswSS9NQ0V0WFJrK2IvTjJHMUpGM3ZI
    UVQyNGRTaFdZRFxudytKSVVTQTNhTGYyc3YwenIyTWRVRWRWV0JKb004blQ0RDR4VmJCT1
    JEKzY2OVcrOW5EZVN3PT1cbi0tLS0tRU5EIEFHRSBFTkNSWVBURUQgRklMRS0tLS0tXG4i
    CgkJCX0KCQldLAoJCSJsYXN0bW9kaWZpZWQiOiAiMjAyMS0xMS0yNlQxNjozNDo1MVoiLA
    oJCSJtYWMiOiAiRU5DW0FFUzI1Nl9HQ00sZGF0YTpDT0d6ZjVZQ0hOTlA2ejRKYUVLcmpO
    M004ZjUrUTF1S1VLVE1Id2ozODgvSUNtTHlpMnNTclRtajdQUCtYN005alRWd2E4d1ZnWV
    RwTkxpVkp4K0xjeHF2SVhNMFR5bysvQ3UxenJmYW85OGFpQUNQOCtUU0VEaUZRTnRFdXMy
    M0grZC9YMWhxTXdSSERJM2tRKzZzY2dFR25xWTU3cjNSRFNBM0U4RWhIcjQ9LGl2Okx4aX
    RWSVltOHNyWlZxRnVlSmg5bG9DbEE0NFkyWjNYQVZZbXhlc01tT2c9LHRhZzpZOHFGRDhV
    R2xEZndOU3Y3eGxjbjZBPT0sdHlwZTpzdHJdIiwKCQkicGdwIjogbnVsbCwKCQkidW5lbm
    NyeXB0ZWRfc3VmZml4IjogIl91bmVuY3J5cHRlZCIsCgkJInZlcnNpb24iOiAiMy43LjEi
    Cgl9Cn0=
`,
			expected: `apiVersion: v1
data:
  token: KipTT1BTKio=
kind: Secret
metadata:
  name: my-secret
type: Opaque
`,
		},
		{
			name: "secret with basic auth",
			yamlStr: `apiVersion: v1
data:
  password: cGFzc3dvcmQK
  username: YWRtaW4K
kind: Secret
metadata:
  name: secret-basic-auth
type: kubernetes.io/basic-auth
`,
			expected: `apiVersion: v1
data:
  password: cGFzc3dvcmQK
  username: YWRtaW4K
kind: Secret
metadata:
  name: secret-basic-auth
type: kubernetes.io/basic-auth
`,
		},
		{
			name: "secret sops secret",
			yamlStr: `apiVersion: v1
data:
  .dockerconfigjson: ENC[AES256_GCM,data:KHCFH3hNnc+PMfWLFEPjebf3W4z4WXbGFAANRZyZC+07z7wlrTALJM6rn8YslW4tMAWCoAYxblC5WRCszTy0h9rw0U/RGOv5H0qCgnNg/FILFUqhwo9pNfrUH+MEP4M9qxxbLKZwObpHUE7DUsKx1JYAxsI=,iv:q48lqUbUQD+0cbYcjNMZMJLRdGHi78ZmDhNAT2th9tg=,tag:QRI2SZZXQrAcdql3R5AH2g==,type:str]
kind: Secret
metadata:
  name: secret
type: kubernetes.io/dockerconfigjson
sops:
  kms: []
  gcp_kms: []
  azure_kv: []
  hc_vault: []
  age:
    - recipient: age10la2ge0wtvx3qr7datqf7rs4yngxszdal927fs9rukamr8u2pshsvtz7ce
      enc: |
        -----BEGIN AGE ENCRYPTED FILE-----
        YWdlLWVuY3J5cHRpb24ub3JnL3YxCi0+IFgyNTUxOSA3eU1CTEJhVXZ4eEVYYkVV
        OU90TEcrR2pYckttN0pBanJoSUZWSW1RQXlRCkUydFJ3V1NZUTBuVFF0aC9GUEcw
        bUdhNjJWTkoyL1FUVi9Dc1dxUDBkM0UKLS0tIE1sQXkwcWdGaEFuY0RHQTVXM0J6
        dWpJcThEbW15V3dXYXpPZklBdW1Hd1kKoIAdmGNPrEctV8h1w8KuvQ5S+BGmgqN9
        MgpNmUhJjWhgcQpb5BRYpQesBOgU5TBGK7j58A6DMDKlSiYZsdQchQ==
        -----END AGE ENCRYPTED FILE-----
  lastmodified: "2022-02-03T16:03:17Z"
  mac: ENC[AES256_GCM,data:AHdYSawajwgAFwlmDN1IPNmT9vWaYKzyVIra2d6sPcjTbZ8/p+VRSRpVm4XZFFsaNnW5AUJaouwXnKYDTmJDXKlr/rQcu9kXqsssQgdzcXaA6l5uJlgsnml8ba7J3OK+iEKMax23mwQEx2EUskCd9ENOwFDkunP02sxqDNOz20k=,iv:8F5OamHt3fAVorf6p+SoIrWoqkcATSGWVoM0EK87S4M=,tag:E1mxXnc7wWkEX5BxhpLtng==,type:str]
  pgp: []
  encrypted_regex: ^(data|stringData)$
  version: 3.7.1
`,
			expected: `apiVersion: v1
data:
  .dockerconfigjson: eyJtYXNrIjoiKipTT1BTKioifQ==
kind: Secret
metadata:
  name: secret
type: kubernetes.io/dockerconfigjson
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := yaml.Parse(tc.yamlStr)
			if err != nil {
				t.Fatalf("unable to parse yaml: %v", err)
			}

			resource := &resource.Resource{RNode: *r}
			err = maskSopsData(resource)
			if err != nil {
				t.Fatalf("unable to trim sops data: %v", err)
			}

			sYaml, err := resource.AsYAML()
			if err != nil {
				t.Fatalf("unable to convert sanitized resources to yaml: %v", err)
			}
			if diff := cmp.Diff(string(sYaml), tc.expected); diff != "" {
				t.Errorf("unexpected sanitized resources: (-got +want)%v", diff)
			}
		})
	}
}

func Test_unMarshallKustomization(t *testing.T) {
	tests := []struct {
		name        string
		localKsFile string
		wantErr     bool
		errString   string
	}{
		{
			name:        "valid kustomization",
			localKsFile: "testdata/local-kustomization/valid.yaml",
		},
		{
			name:        "Multi-doc yaml containing kustomization and other resources",
			localKsFile: "testdata/local-kustomization/multi-doc-valid.yaml",
		},
		{
			name:        "no namespace",
			localKsFile: "testdata/local-kustomization/no-ns.yaml",
		},
		{
			name:        "kustomization with a different name",
			localKsFile: "testdata/local-kustomization/different-name.yaml",
			wantErr:     true,
			errString:   "failed find kustomization with name",
		},
		{
			name:        "yaml containing other resource with same name as kustomization",
			localKsFile: "testdata/local-kustomization/invalid-resource.yaml",
			wantErr:     true,
			errString:   "failed find kustomization with name",
		},
	}

	b := &Builder{
		name:      "podinfo",
		namespace: "flux-system",
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b.kustomizationFile = tt.localKsFile
			ks, err := b.unMarshallKustomization()
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("unexpected err '%s'", err)
				}

				if ks.Name != b.name && ks.Namespace != b.namespace {
					t.Errorf("expected kustomization '%s/%s' to match '%s/%s'",
						ks.Name, ks.Namespace, b.name, b.namespace)
				}
			} else {
				if err == nil {
					t.Fatal("expected error but got nil")
				}

				if !strings.Contains(err.Error(), tt.errString) {
					t.Errorf("expected error '%s' to contain string '%s'", err.Error(), tt.errString)
				}
			}
		})
	}
	t.Run("correct parsing of multiple documents", func(t *testing.T) {
		b.kustomizationFile = "testdata/local-kustomization/multi-doc-reset.yaml"
		ks, err := b.unMarshallKustomization()
		if err != nil {
			t.Errorf("unexpected err '%s'", err)
		}
		if len(ks.Spec.Components) > 0 {
			t.Errorf("previous Kustomization in file leaked into subsequent Kustomizations")
		}
	})
}

func Test_ResolveKustomization(t *testing.T) {
	tests := []struct {
		name              string
		localKsFile       string
		liveKustomization *kustomizev1.Kustomization
		dryrun            bool
	}{
		{
			name:        "valid kustomization",
			localKsFile: "testdata/local-kustomization/valid.yaml",
		},
		{
			name:        "local and live kustomization",
			localKsFile: "testdata/local-kustomization/valid.yaml",
			liveKustomization: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "podinfo",
					Namespace: "flux-system",
				},
				Spec: kustomizev1.KustomizationSpec{
					Interval: metav1.Duration{Duration: time.Minute * 5},
					Path:     "./testdata/local-kustomization/valid.yaml",
				},
				Status: kustomizev1.KustomizationStatus{
					Conditions: []metav1.Condition{
						{
							Type:   meta.ReadyCondition,
							Status: metav1.ConditionTrue,
						},
					},
					Inventory: &kustomizev1.ResourceInventory{
						Entries: []kustomizev1.ResourceRef{
							{
								ID:      "flux-system_podinfo_v1_service_podinfo",
								Version: "v1",
							},
						},
					},
				},
			},
		},
		{
			name:        "local and live kustomization with dryrun",
			localKsFile: "testdata/local-kustomization/valid.yaml",
			liveKustomization: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "podinfo",
					Namespace: "flux-system",
				},
				Spec: kustomizev1.KustomizationSpec{
					Interval: metav1.Duration{Duration: time.Minute * 5},
					Path:     "./testdata/local-kustomization/valid.yaml",
				},
				Status: kustomizev1.KustomizationStatus{
					Conditions: []metav1.Condition{
						{
							Type:   meta.ReadyCondition,
							Status: metav1.ConditionTrue,
						},
					},
					Inventory: &kustomizev1.ResourceInventory{
						Entries: []kustomizev1.ResourceRef{
							{
								ID:      "flux-system_podinfo_v1_service_podinfo",
								Version: "v1",
							},
						},
					},
				},
			},
			dryrun: true,
		},
		{
			name: "live kustomization",
			liveKustomization: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "podinfo",
					Namespace: "flux-system",
				},
				Spec: kustomizev1.KustomizationSpec{
					Interval: metav1.Duration{Duration: time.Minute * 5},
					Path:     "./testdata/local-kustomization/valid.yaml",
				},
				Status: kustomizev1.KustomizationStatus{
					Conditions: []metav1.Condition{
						{
							Type:   meta.ReadyCondition,
							Status: metav1.ConditionTrue,
						},
					},
					Inventory: &kustomizev1.ResourceInventory{
						Entries: []kustomizev1.ResourceRef{
							{
								ID:      "flux-system_podinfo_v1_service_podinfo",
								Version: "v1",
							},
						},
					},
				},
			},
		},
	}

	b := &Builder{
		name:      "podinfo",
		namespace: "flux-system",
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b.kustomizationFile = tt.localKsFile
			b.dryRun = tt.dryrun
			ks, err := b.resolveKustomization(tt.liveKustomization)
			if err != nil {
				t.Errorf("unexpected err '%s'", err)
			}
			if !tt.dryrun {
				if b.kustomizationFile == "" {
					if cmp.Diff(ks, tt.liveKustomization) != "" {
						t.Errorf("expected kustomization to match live kustomization")
					}
				} else {
					if tt.liveKustomization != nil && cmp.Diff(ks.Status, tt.liveKustomization.Status) != "" {
						t.Errorf("expected kustomization status to match live kustomization status")
					}
				}
			} else {
				if ks.Status.Inventory != nil {
					fmt.Println(ks.Status.Inventory)
					t.Errorf("expected kustomization status to be nil")
				}
			}
		})
	}
}

func Test_isKustomization(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
		object   *unstructured.Unstructured
	}{
		{
			name: "flux kustomization",
			object: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "kustomize.toolkit.fluxcd.io/v1",
					"kind":       "Kustomization",
				},
			},
			expected: true,
		},
		{
			name: "other kustomization",
			object: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "kustomize.config.k8s.io/v1beta1",
					"kind":       "Kustomization",
				},
			},
			expected: false,
		},
		{
			name: "wrong kind",
			object: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "kustomize.toolkit.fluxcd.io/v1",
					"kind":       "ConfigMap",
				},
			},
			expected: false,
		},
		{
			name: "wrong object",
			object: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := isKustomization(tt.object)
			if actual != tt.expected {
				t.Fatalf("got '%v', want '%v'", actual, tt.expected)
			}
		})
	}
}

func Test_kustomizationsEqual(t *testing.T) {
	tests := []struct {
		name           string
		kustomization1 *kustomizev1.Kustomization
		kustomization2 *kustomizev1.Kustomization
		expected       bool
	}{
		{
			name: "equal",
			kustomization1: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "podinfo",
					Namespace: "flux-system",
				},
			},
			kustomization2: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "podinfo",
					Namespace: "flux-system",
				},
			},
			expected: true,
		},
		{
			name: "wrong name",
			kustomization1: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "podinfo",
					Namespace: "flux-system",
				},
			},
			kustomization2: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-app",
					Namespace: "flux-system",
				},
			},
			expected: false,
		},
		{
			name: "wrong namespace",
			kustomization1: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "podinfo",
					Namespace: "flux-system",
				},
			},
			kustomization2: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "podinfo",
					Namespace: "my-ns",
				},
			},
			expected: false,
		},
		{
			name: "wrong name and namespace",
			kustomization1: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "podinfo",
					Namespace: "flux-system",
				},
			},
			kustomization2: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-app",
					Namespace: "my-ns",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := kustomizationsEqual(tt.kustomization1, tt.kustomization2)
			if actual != tt.expected {
				t.Fatalf("got '%v', want '%v'", actual, tt.expected)
			}
		})
	}
}

func Test_kustomizationPath(t *testing.T) {
	tests := []struct {
		name          string
		kustomization *kustomizev1.Kustomization
		expected      string
		wantErr       bool
		errString     string
	}{
		{
			name: "full repo",
			kustomization: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-app",
					Namespace: "flux-system",
				},
				Spec: kustomizev1.KustomizationSpec{
					Path: "my-path",
					SourceRef: kustomizev1.CrossNamespaceSourceReference{
						Kind:      "GitRepository",
						Name:      "my-repo",
						Namespace: "flux-system",
					},
				},
			},
			expected: "path/to/local/git/my-path",
		},
		{
			name: "repo without namespace",
			kustomization: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-app",
					Namespace: "flux-system",
				},
				Spec: kustomizev1.KustomizationSpec{
					Path: "my-path",
					SourceRef: kustomizev1.CrossNamespaceSourceReference{
						Kind:      "GitRepository",
						Name:      "my-repo",
						Namespace: "",
					},
				},
			},
			expected: "path/to/local/git/my-path",
		},
		{
			name: "repo not found",
			kustomization: &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-app",
					Namespace: "flux-system",
				},
				Spec: kustomizev1.KustomizationSpec{
					Path: "my-path",
					SourceRef: kustomizev1.CrossNamespaceSourceReference{
						Kind:      "GitRepository",
						Name:      "my-repo",
						Namespace: "my-ns",
					},
				},
			},
			wantErr:   true,
			errString: "cannot get local path",
		},
	}

	b := &Builder{
		name:      "podinfo",
		namespace: "flux-system",
		localSources: map[string]string{
			"GitRepository/flux-system/my-repo": "./path/to/local/git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := b.kustomizationPath(tt.kustomization)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("unexpected err '%s'", err)
				}

				if actual != tt.expected {
					t.Errorf("got '%v', want '%v'", actual, tt.expected)
				}
			} else {
				if err == nil {
					t.Fatal("expected error but got nil")
				}

				if !strings.Contains(err.Error(), tt.errString) {
					t.Errorf("expected error '%s' to contain string '%s'", err.Error(), tt.errString)
				}
			}

		})
	}
}

// chdirTemp changes to the given directory and restores the original on cleanup.
func chdirTemp(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}

func Test_inMemoryFsBackend_Generate(t *testing.T) {
	srcDir := t.TempDir()
	chdirTemp(t, srcDir)

	kusYAML := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- configmap.yaml
`
	cmYAML := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  key: value
`
	if err := os.WriteFile(filepath.Join(srcDir, "kustomization.yaml"), []byte(kusYAML), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "configmap.yaml"), []byte(cmYAML), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// snapshot source dir
	beforeFiles := map[string]string{}
	filepath.Walk(srcDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, _ := os.ReadFile(p)
		rel, _ := filepath.Rel(srcDir, p)
		beforeFiles[rel] = string(data)
		return nil
	})

	ks := unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "kustomize.toolkit.fluxcd.io/v1",
		"kind":       "Kustomization",
		"metadata":   map[string]interface{}{"name": "test", "namespace": "default"},
		"spec": map[string]interface{}{
			"targetNamespace": "my-ns",
		},
	}}
	gen := kustomize.NewGenerator(srcDir, ks)

	backend := inMemoryFsBackend{}
	fs, dir, action, err := backend.Generate(gen, srcDir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if action != kustomize.UnchangedAction {
		t.Errorf("expected UnchangedAction, got %q", action)
	}

	// kustomization.yaml should contain the merged targetNamespace
	data, err := fs.ReadFile(filepath.Join(dir, "kustomization.yaml"))
	if err != nil {
		t.Fatalf("ReadFile kustomization.yaml: %v", err)
	}
	if !strings.Contains(string(data), "my-ns") {
		t.Errorf("expected kustomization to contain targetNamespace, got:\n%s", data)
	}

	// resource file should be readable from disk through the memory fs
	data, err = fs.ReadFile(filepath.Join(dir, "configmap.yaml"))
	if err != nil {
		t.Fatalf("ReadFile configmap.yaml: %v", err)
	}
	if diff := cmp.Diff(string(data), cmYAML); diff != "" {
		t.Errorf("configmap mismatch: (-got +want)%s", diff)
	}

	// source directory must be unmodified
	afterFiles := map[string]string{}
	filepath.Walk(srcDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, _ := os.ReadFile(p)
		rel, _ := filepath.Rel(srcDir, p)
		afterFiles[rel] = string(data)
		return nil
	})
	if diff := cmp.Diff(afterFiles, beforeFiles); diff != "" {
		t.Errorf("source directory was modified: (-got +want)%s", diff)
	}
}

func Test_inMemoryFsBackend_Generate_parentRef(t *testing.T) {
	// tmpDir/
	//   configmap.yaml                  (referenced as ../../configmap.yaml)
	//   overlay/sub/kustomization.yaml
	tmpDir := t.TempDir()
	chdirTemp(t, tmpDir)

	cmYAML := `apiVersion: v1
kind: ConfigMap
metadata:
  name: parent-cm
data:
  key: value
`
	if err := os.WriteFile(filepath.Join(tmpDir, "configmap.yaml"), []byte(cmYAML), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	overlayDir := filepath.Join(tmpDir, "overlay", "sub")
	if err := os.MkdirAll(overlayDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	kusYAML := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- ../../configmap.yaml
`
	if err := os.WriteFile(filepath.Join(overlayDir, "kustomization.yaml"), []byte(kusYAML), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ks := unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "kustomize.toolkit.fluxcd.io/v1",
		"kind":       "Kustomization",
		"metadata":   map[string]interface{}{"name": "test", "namespace": "default"},
		"spec": map[string]interface{}{
			"targetNamespace": "parent-ns",
		},
	}}
	gen := kustomize.NewGenerator(overlayDir, ks)

	backend := inMemoryFsBackend{}
	fs, dir, _, err := backend.Generate(gen, overlayDir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// ../../configmap.yaml must resolve through the disk layer
	m, err := kustomize.Build(fs, dir)
	if err != nil {
		t.Fatalf("kustomize.Build failed (parent ref not resolved): %v", err)
	}

	resources := m.Resources()
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].GetName() != "parent-cm" {
		t.Errorf("expected resource name parent-cm, got %s", resources[0].GetName())
	}
	if resources[0].GetNamespace() != "parent-ns" {
		t.Errorf("expected namespace parent-ns, got %s", resources[0].GetNamespace())
	}
}

func Test_inMemoryFsBackend_Generate_outsideCwd(t *testing.T) {
	// Two sibling temp dirs: one for the source tree, one as cwd.
	// The kustomization references a file that exists on disk but is
	// outside cwd, so the secure filesystem must reject it.
	//
	// parentDir/
	//   outside/configmap.yaml      (exists but outside cwd)
	//   cwd/overlay/kustomization.yaml  (references ../../outside/configmap.yaml)
	parentDir := t.TempDir()

	outsideDir := filepath.Join(parentDir, "outside")
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outsideDir, "configmap.yaml"), []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: outside-cm
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cwdDir := filepath.Join(parentDir, "cwd")
	overlayDir := filepath.Join(cwdDir, "overlay")
	if err := os.MkdirAll(overlayDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overlayDir, "kustomization.yaml"), []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- ../../outside/configmap.yaml
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Set cwd to cwdDir so the secure root excludes outsideDir.
	chdirTemp(t, cwdDir)

	ks := unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "kustomize.toolkit.fluxcd.io/v1",
		"kind":       "Kustomization",
		"metadata":   map[string]interface{}{"name": "test", "namespace": "default"},
	}}
	gen := kustomize.NewGenerator(overlayDir, ks)

	backend := inMemoryFsBackend{}
	fs, dir, _, err := backend.Generate(gen, overlayDir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Build must fail because the resource is outside the secure root.
	_, err = kustomize.Build(fs, dir)
	if err == nil {
		t.Fatal("expected error when referencing resource outside cwd, got nil")
	}
}
