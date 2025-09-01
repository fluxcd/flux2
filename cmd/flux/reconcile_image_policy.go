package main

import (
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
	"github.com/spf13/cobra"
)

var reconcileImagePolicyCmd = &cobra.Command{
	Use:   "policy [name]",
	Short: "Reconcile an ImagePolicy",
	Long:  `The reconcile image policy command triggers a reconciliation of an ImagePolicy resource and waits for it to finish.`,
	Example: `
	# Trigger a reconciliation for an existing image policy called 'alpine'
	flux reconcile image policy alpine`,
	ValidArgsFunction: resourceNamesCompletionFunc(imagev1.GroupVersion.WithKind(imagev1.ImagePolicyKind)),
	RunE: reconcileCommand{
		apiType: imagePolicyType,
		object:  imagePolicyAdapter{&imagev1.ImagePolicy{}},
	}.run,
}

func init() {
	reconcileImageCmd.AddCommand(reconcileImagePolicyCmd)
}
