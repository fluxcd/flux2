package main

import (
	"context"
	"fmt"

	sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
)

var deleteSourceGitCmd = &cobra.Command{
	Use:   "git [name]",
	Short: "Delete git source",
	RunE:  deleteSourceGitCmdRun,
}

func init() {
	deleteSourceCmd.AddCommand(deleteSourceGitCmd)
}

func deleteSourceGitCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("git name is required")
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	var git sourcev1.GitRepository
	err = kubeClient.Get(ctx, namespacedName, &git)
	if err != nil {
		return err
	}

	if !deleteSilent {
		prompt := promptui.Prompt{
			Label: fmt.Sprintf(
				"Are you sure you want to delete the %s source from the %s namespace", name, namespace,
			),
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("aborting")
		}
	}

	logAction("deleting source %s in %s namespace", name, namespace)
	err = kubeClient.Delete(ctx, &git)
	if err != nil {
		return err
	}
	logSuccess("source deleted")

	return nil
}
