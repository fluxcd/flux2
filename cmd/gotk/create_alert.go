/*
Copyright 2020 The Flux CD contributors.

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
	"github.com/fluxcd/pkg/apis/meta"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
)

var createAlertCmd = &cobra.Command{
	Use:   "alert [name]",
	Short: "Create or update a Alert resource",
	Long:  "The create alert command generates a Alert resource.",
	Example: `  # Create an Alert for kustomization events
  gotk create alert \
  --event-severity info \
  --event-source Kustomization/gotk-system \
  --provider-ref slack \
  gotk-system
`,
	RunE: createAlertCmdRun,
}

var (
	aProviderRef   string
	aEventSeverity string
	aEventSources  []string
)

func init() {
	createAlertCmd.Flags().StringVar(&aProviderRef, "provider-ref", "", "reference to provider")
	createAlertCmd.Flags().StringVar(&aEventSeverity, "event-severity", "", "severity of events to send alerts for")
	createAlertCmd.Flags().StringArrayVar(&aEventSources, "event-source", []string{}, "sources that should generate alerts (<kind>/<name>)")
	createCmd.AddCommand(createAlertCmd)
}

func createAlertCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Alert name is required")
	}
	name := args[0]

	if aProviderRef == "" {
		return fmt.Errorf("provider ref is required")
	}

	eventSources := []notificationv1.CrossNamespaceObjectReference{}
	for _, eventSource := range aEventSources {
		kind, name := utils.parseObjectKindName(eventSource)
		if kind == "" {
			return fmt.Errorf("invalid event source '%s', must be in format <kind>/<name>", eventSource)
		}

		eventSources = append(eventSources, notificationv1.CrossNamespaceObjectReference{
			Kind: kind,
			Name: name,
		})
	}

	if len(eventSources) == 0 {
		return fmt.Errorf("at least one event source is required")
	}

	sourceLabels, err := parseLabels()
	if err != nil {
		return err
	}

	if !export {
		logger.Generatef("generating Alert")
	}

	alert := notificationv1.Alert{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    sourceLabels,
		},
		Spec: notificationv1.AlertSpec{
			ProviderRef: corev1.LocalObjectReference{
				Name: aProviderRef,
			},
			EventSeverity: aEventSeverity,
			EventSources:  eventSources,
			Suspend:       false,
		},
	}

	if export {
		return exportAlert(alert)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	logger.Actionf("applying Alert")
	namespacedName, err := upsertAlert(ctx, kubeClient, &alert)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for Alert reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isAlertReady(ctx, kubeClient, namespacedName, &alert)); err != nil {
		return err
	}
	logger.Successf("Alert %s is ready", name)
	return nil
}

func upsertAlert(ctx context.Context, kubeClient client.Client,
	alert *notificationv1.Alert) (types.NamespacedName, error) {
	namespacedName := types.NamespacedName{
		Namespace: alert.GetNamespace(),
		Name:      alert.GetName(),
	}

	var existing notificationv1.Alert
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

func isAlertReady(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, alert *notificationv1.Alert) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, alert)
		if err != nil {
			return false, err
		}

		if c := meta.GetCondition(alert.Status.Conditions, meta.ReadyCondition); c != nil {
			switch c.Status {
			case corev1.ConditionTrue:
				return true, nil
			case corev1.ConditionFalse:
				return false, fmt.Errorf(c.Message)
			}
		}
		return false, nil
	}
}
