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

package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1"
	notificationv1b3 "github.com/fluxcd/notification-controller/api/v1beta3"
	"github.com/fluxcd/pkg/apis/meta"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var createAlertCmd = &cobra.Command{
	Use:   "alert [name]",
	Short: "Create or update a Alert resource",
	Long:  withPreviewNote(`The create alert command generates a Alert resource.`),
	Example: `  # Create an Alert for kustomization events
  flux create alert \
  --event-severity info \
  --event-source Kustomization/flux-system \
  --provider-ref slack \
  flux-system`,
	RunE: createAlertCmdRun,
}

type alertFlags struct {
	providerRef   string
	eventSeverity string
	eventSources  []string
}

var alertArgs alertFlags

func init() {
	createAlertCmd.Flags().StringVar(&alertArgs.providerRef, "provider-ref", "", "reference to provider")
	createAlertCmd.Flags().StringVar(&alertArgs.eventSeverity, "event-severity", "", "severity of events to send alerts for")
	createAlertCmd.Flags().StringSliceVar(&alertArgs.eventSources, "event-source", []string{}, "sources that should generate alerts (<kind>/<name>), also accepts comma-separated values")
	createCmd.AddCommand(createAlertCmd)
}

func createAlertCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	if alertArgs.providerRef == "" {
		return fmt.Errorf("provider ref is required")
	}

	eventSources := []notificationv1.CrossNamespaceObjectReference{}
	for _, eventSource := range alertArgs.eventSources {
		kind, name, namespace := utils.ParseObjectKindNameNamespace(eventSource)
		if kind == "" {
			return fmt.Errorf("invalid event source '%s', must be in format <kind>/<name>", eventSource)
		}

		eventSources = append(eventSources, notificationv1.CrossNamespaceObjectReference{
			Kind:      kind,
			Name:      name,
			Namespace: namespace,
		})
	}

	if len(eventSources) == 0 {
		return fmt.Errorf("at least one event source is required")
	}

	sourceLabels, err := parseLabels()
	if err != nil {
		return err
	}

	if !createArgs.export {
		logger.Generatef("generating Alert")
	}

	alert := notificationv1b3.Alert{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: *kubeconfigArgs.Namespace,
			Labels:    sourceLabels,
		},
		Spec: notificationv1b3.AlertSpec{
			ProviderRef: meta.LocalObjectReference{
				Name: alertArgs.providerRef,
			},
			EventSeverity: alertArgs.eventSeverity,
			EventSources:  eventSources,
			Suspend:       false,
		},
	}

	if createArgs.export {
		return printExport(exportAlert(&alert))
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	logger.Actionf("applying Alert")
	namespacedName, err := upsertAlert(ctx, kubeClient, &alert)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for Alert reconciliation")
	if err := wait.PollUntilContextTimeout(ctx, rootArgs.pollInterval, rootArgs.timeout, true,
		isStaticObjectReadyConditionFunc(kubeClient, namespacedName, &alert)); err != nil {
		return err
	}
	logger.Successf("Alert %s is ready", name)
	return nil
}

func upsertAlert(ctx context.Context, kubeClient client.Client,
	alert *notificationv1b3.Alert) (types.NamespacedName, error) {
	namespacedName := types.NamespacedName{
		Namespace: alert.GetNamespace(),
		Name:      alert.GetName(),
	}

	var existing notificationv1b3.Alert
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, alert); err != nil {
				return namespacedName, err
			} else {
				logger.Successf("Alert created")
				return namespacedName, nil
			}
		}
		return namespacedName, err
	}

	existing.Labels = alert.Labels
	existing.Spec = alert.Spec
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return namespacedName, err
	}
	alert = &existing
	logger.Successf("Alert updated")
	return namespacedName, nil
}
