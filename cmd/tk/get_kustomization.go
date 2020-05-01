package main

import (
	"context"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1alpha1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var getKsCmd = &cobra.Command{
	Use:     "kustomizations",
	Aliases: []string{"ks"},
	Short:   "Get kustomizations status",
	Long: `
The get kustomizations command prints the status of the resources.`,
	RunE: getKsCmdRun,
}

func init() {
	getCmd.AddCommand(getKsCmd)
}

func getKsCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	var list kustomizev1.KustomizationList
	err = kubeClient.List(ctx, &list, client.InNamespace(namespace))
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		logFailure("no kustomizations found in %s namespace", namespace)
		return nil
	}

	for _, kustomization := range list.Items {
		if kustomization.Spec.Suspend {
			logSuccess("%s is suspended", kustomization.GetName())
			continue
		}
		isInitialized := false
		for _, condition := range kustomization.Status.Conditions {
			if condition.Type == kustomizev1.ReadyCondition {
				if condition.Status != corev1.ConditionFalse {
					logSuccess("%s last applied revision %s", kustomization.GetName(), kustomization.Status.LastAppliedRevision)
				} else {
					logFailure("%s %s", kustomization.GetName(), condition.Message)
				}
				isInitialized = true
				break
			}
		}
		if !isInitialized {
			logFailure("%s is not ready", kustomization.GetName())
		}
	}
	return nil
}
