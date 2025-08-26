/*
Copyright 2025 The Flux authors

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

package main

import (
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1"
	"github.com/spf13/cobra"
)

var resumeImagePolicyCmd = &cobra.Command{
	Use:   "policy [name]",
	Short: "Resume an ImagePolicy",
	Long:  `The resume image policy command resumes a suspended ImagePolicy resource.`,
	Example: `
	# Resume a suspended image policy called 'alpine'
	flux resume image policy alpine`,
	ValidArgsFunction: resourceNamesCompletionFunc(imagev1.GroupVersion.WithKind(imagev1.ImagePolicyKind)),
	RunE: resumeCommand{
		apiType: imagePolicyType,
		list:    imagePolicyListAdapter{&imagev1.ImagePolicyList{}},
	}.run,
}

func init() {
	resumeImageCmd.AddCommand(resumeImagePolicyCmd)
}
