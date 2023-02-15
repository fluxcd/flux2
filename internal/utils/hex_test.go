/*
Copyright 2023 The Flux authors

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
	"testing"

	. "github.com/onsi/gomega"
)

func TestTruncateHex(t *testing.T) {
	tests := []struct {
		name string
		str  string
		want string
	}{
		{
			name: "SHA1 hash",
			str:  "16cfcc0b9066b3234dda29927ac1c19860d9663f",
			want: "16cfcc0b",
		},
		{
			name: "SHA256 hash",
			str:  "c2448c95e262b10f9c1137bf1472f51c04dffca76966ff15eff409d0b300c0b0",
			want: "c2448c95",
		},
		{
			name: "BLAKE3 hash",
			str:  "d7b332559f4e57c01dcbe24e53346b4e47696fd2a07f39639a4017a8c3a1d045",
			want: "d7b33255",
		},
		{
			name: "SHA512 hash",
			str:  "dd81564b7e1e1d5986b166c21963d602f47f8610bf2a6ebbfd2f9c1e5ef05ef134f07e587383cbc049325c43e0e6817b5a282a74c0d569a5e057118484989781",
			want: "dd81564b",
		},
		{
			name: "part of digest",
			str:  "sha256:c2448c95e262b10f9c1137bf1472f51c04dffca76966ff15eff409d0b300c0b0",
			want: "sha256:c2448c95",
		},
		{
			name: "part of revision with digest",
			str:  "tag@sha256:c2448c95e262b10f9c1137bf1472f51c04dffca76966ff15eff409d0b300c0b0",
			want: "tag@sha256:c2448c95",
		},
		{
			name: "legacy revision with hash",
			str:  "HEAD/16cfcc0b9066b3234dda29927ac1c19860d9663f",
			want: "HEAD/16cfcc0b",
		},
		{
			name: "hex exceeding max length",
			str:  "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			want: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			name: "hex under min length",
			str:  "ffffffffffffff",
			want: "ffffffffffffff",
		},
		{
			name: "within string",
			str:  "this is a lengthy string with a hash 16cfcc0b9066b3234dda29927ac1c19860d9663f in it",
			want: "this is a lengthy string with a hash 16cfcc0b in it",
		},
		{
			name: "within string (quoted)",
			str:  "this is a lengthy string with a hash \"c2448c95e262b10f9c1137bf1472f51c04dffca76966ff15eff409d0b300c0b0\" in it",
			want: "this is a lengthy string with a hash \"c2448c95\" in it",
		},
		{
			name: "within string (single quoted)",
			str:  "this is a lengthy string with a hash 'sha256:c2448c95e262b10f9c1137bf1472f51c04dffca76966ff15eff409d0b300c0b0' in it",
			want: "this is a lengthy string with a hash 'sha256:c2448c95' in it",
		},
		{
			name: "arbitrary string",
			str:  "which should not be modified",
			want: "which should not be modified",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			g.Expect(TruncateHex(tt.str)).To(Equal(tt.want))
		})
	}
}
