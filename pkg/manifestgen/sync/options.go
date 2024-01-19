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

package sync

import (
	"time"
)

type Options struct {
	Interval          time.Duration
	URL               string
	Name              string
	Namespace         string
	Branch            string
	Tag               string
	SemVer            string
	Commit            string
	Secret            string
	TargetPath        string
	ManifestFile      string
	RecurseSubmodules bool
}

func MakeDefaultOptions() Options {
	return Options{
		Interval:     1 * time.Minute,
		URL:          "",
		Name:         "flux-system",
		Namespace:    "flux-system",
		Branch:       "main",
		Secret:       "flux-system",
		ManifestFile: "gotk-sync.yaml",
		TargetPath:   "",
	}
}
