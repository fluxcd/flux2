/*
Copyright 2026 The Flux authors

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

package printers

import (
	"bytes"
	"testing"

	. "github.com/onsi/gomega"
)

func TestTablePrinterEscapesControlCharacters(t *testing.T) {
	g := NewWithT(t)

	rows := [][]string{
		{"podinfo", "False", "\x1b[2J\x1b[1;1Hspoofed\rhidden"},
	}

	var buf bytes.Buffer
	err := TablePrinter([]string{"name", "ready", "message"}).Print(&buf, rows)
	g.Expect(err).ToNot(HaveOccurred())

	out := buf.String()
	g.Expect(out).ToNot(ContainSubstring("\x1b"))
	g.Expect(out).ToNot(ContainSubstring("\r"))
	g.Expect(out).To(ContainSubstring(`^[[2J^[[1;1Hspoofed\rhidden`))
}
