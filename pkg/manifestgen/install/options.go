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

import "time"

type Options struct {
	BaseURL                 string
	Version                 string
	Namespace               string
	Components              []string
	ComponentsExtra         []string
	EventsAddr              string
	Registry                string
	ImagePullSecret         string
	WatchAllNamespaces      bool
	NetworkPolicy           bool
	LogLevel                string
	NotificationController  string
	ManifestFile            string
	Timeout                 time.Duration
	TargetPath              string
	ClusterDomain           string
	TolerationKeys          []string
	AdditionalNodeSelectors map[string]string
}

func MakeDefaultOptions() Options {
	return Options{
		Version:                "latest",
		Namespace:              "flux-system",
		Components:             []string{"source-controller", "kustomize-controller", "helm-controller", "notification-controller"},
		ComponentsExtra:        []string{"image-reflector-controller", "image-automation-controller"},
		EventsAddr:             "",
		Registry:               "ghcr.io/fluxcd",
		ImagePullSecret:        "",
		WatchAllNamespaces:     true,
		NetworkPolicy:          true,
		LogLevel:               "info",
		BaseURL:                "https://github.com/fluxcd/flux2/releases",
		NotificationController: "notification-controller",
		ManifestFile:           "gotk-components.yaml",
		Timeout:                time.Minute,
		TargetPath:             "",
		ClusterDomain:          "cluster.local",
	}
}

func containsItemString(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
