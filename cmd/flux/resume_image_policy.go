package main

import (
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
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
