package main

import (
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
	"github.com/spf13/cobra"
)

var suspendImagePolicyCmd = &cobra.Command{
	Use:               "policy [name]",
	Short:             "Suspend an ImagePolicy",
	Long:              `The suspend image policy command suspends the reconciliation of an ImagePolicy resource.`,
	ValidArgsFunction: resourceNamesCompletionFunc(imagev1.GroupVersion.WithKind(imagev1.ImagePolicyKind)),
	RunE: suspendCommand{
		apiType: imagePolicyType,
		list:    imagePolicyListAdapter{&imagev1.ImagePolicyList{}},
	}.run,
}

func init() {
	suspendImageCmd.AddCommand(suspendImagePolicyCmd)
}
