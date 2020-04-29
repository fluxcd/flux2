package main

import (
	"context"

	sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var getSourceGitCmd = &cobra.Command{
	Use:   "git",
	Short: "Get git sources status",
	Long: `
The get sources command prints the status of the git resources.`,
	RunE: getSourceGitCmdRun,
}

func init() {
	getSourceCmd.AddCommand(getSourceGitCmd)
}

func getSourceGitCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	var list sourcev1.GitRepositoryList
	err = kubeClient.List(ctx, &list, client.InNamespace(namespace))
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		logFailure("no sources found in %s namespace", namespace)
		return nil
	}

	for _, source := range list.Items {
		isInitialized := false
		for _, condition := range source.Status.Conditions {
			if condition.Type == sourcev1.ReadyCondition {
				if condition.Status != corev1.ConditionFalse {
					logSuccess("%s last fetched revision %s", source.GetName(), source.Status.Artifact.Revision)
				} else {
					logFailure("%s %s", source.GetName(), condition.Message)
				}
				isInitialized = true
				break
			}
		}
		if !isInitialized {
			logFailure("%s is not ready", source.GetName())
		}
	}
	return nil
}
