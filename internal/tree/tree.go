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

Derived work from https://github.com/d6o/GoTree
Copyright (c) 2017 Diego Siqueira
*/

package tree

import (
	"strings"

	"github.com/fluxcd/cli-utils/pkg/object"
	ssautil "github.com/fluxcd/pkg/ssa/utils"
)

const (
	newLine      = "\n"
	emptySpace   = "    "
	middleItem   = "├── "
	continueItem = "│   "
	lastItem     = "└── "
)

type (
	objMetadataTree struct {
		Resource     object.ObjMetadata `json:"resource"`
		ResourceTree []ObjMetadataTree  `json:"resources,omitempty"`
	}

	ObjMetadataTree interface {
		Add(objMetadata object.ObjMetadata) ObjMetadataTree
		AddTree(tree ObjMetadataTree)
		Items() []ObjMetadataTree
		Text() string
		Print() string
	}

	printer struct {
	}

	Printer interface {
		Print(ObjMetadataTree) string
	}
)

func New(objMetadata object.ObjMetadata) ObjMetadataTree {
	return &objMetadataTree{
		Resource:     objMetadata,
		ResourceTree: []ObjMetadataTree{},
	}
}

func (t *objMetadataTree) Add(objMetadata object.ObjMetadata) ObjMetadataTree {
	n := New(objMetadata)
	t.ResourceTree = append(t.ResourceTree, n)
	return n
}

func (t *objMetadataTree) AddTree(tree ObjMetadataTree) {
	t.ResourceTree = append(t.ResourceTree, tree)
}

func (t *objMetadataTree) Text() string {
	return ssautil.FmtObjMetadata(t.Resource)
}

func (t *objMetadataTree) Items() []ObjMetadataTree {
	return t.ResourceTree
}

func (t *objMetadataTree) Print() string {
	return newPrinter().Print(t)
}

func newPrinter() Printer {
	return &printer{}
}

func (p *printer) Print(t ObjMetadataTree) string {
	return t.Text() + newLine + p.printItems(t.Items(), []bool{})
}

func (p *printer) printText(text string, spaces []bool, last bool) string {
	var result string
	for _, space := range spaces {
		if space {
			result += emptySpace
		} else {
			result += continueItem
		}
	}

	indicator := middleItem
	if last {
		indicator = lastItem
	}

	var out string
	lines := strings.Split(text, "\n")
	for i := range lines {
		text := lines[i]
		if i == 0 {
			out += result + indicator + text + newLine
			continue
		}
		if last {
			indicator = emptySpace
		} else {
			indicator = continueItem
		}
		out += result + indicator + text + newLine
	}

	return out
}

func (p *printer) printItems(t []ObjMetadataTree, spaces []bool) string {
	var result string
	for i, f := range t {
		last := i == len(t)-1
		result += p.printText(f.Text(), spaces, last)
		if len(f.Items()) > 0 {
			spacesChild := append(spaces, last)
			result += p.printItems(f.Items(), spacesChild)
		}
	}
	return result
}
