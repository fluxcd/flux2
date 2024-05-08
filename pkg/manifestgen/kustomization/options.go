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

package kustomization

import "sigs.k8s.io/kustomize/kyaml/filesys"

type Options struct {
	FileSystem filesys.FileSystem
	BaseDir    string
	TargetPath string
}

func MakeDefaultOptions() Options {
	// TODO(hidde): switch MakeFsOnDisk to MakeFsOnDiskSecureBuild when we
	//  break API.
	return Options{
		FileSystem: filesys.MakeFsOnDisk(),
		BaseDir:    "",
		TargetPath: "",
	}
}
