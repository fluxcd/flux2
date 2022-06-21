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

package oci

import (
	"fmt"
)

const (
	SourceAnnotation   = "source.toolkit.fluxcd.io/url"
	RevisionAnnotation = "source.toolkit.fluxcd.io/revision"
)

type Metadata struct {
	Source   string `json:"source_url"`
	Revision string `json:"source_revision"`
	Digest   string `json:"digest"`
}

func (m *Metadata) ToAnnotations() map[string]string {
	annotations := map[string]string{
		SourceAnnotation:   m.Source,
		RevisionAnnotation: m.Revision,
	}

	return annotations
}

func GetMetadata(annotations map[string]string) (*Metadata, error) {
	source, ok := annotations[SourceAnnotation]
	if !ok {
		return nil, fmt.Errorf("'%s' annotation not found", SourceAnnotation)
	}

	revision, ok := annotations[RevisionAnnotation]
	if !ok {
		return nil, fmt.Errorf("'%s' annotation not found", RevisionAnnotation)
	}

	m := Metadata{
		Source:   source,
		Revision: revision,
	}

	return &m, nil
}
