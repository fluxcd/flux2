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
	"strings"
	"testing"
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
