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

var createReceiverCmd = &cobra.Command{
	Use:     "receiver [name]",
	Aliases: []string{"rcv"},
	Short:   "Create or update a Receiver resource",
	Long:    "The create receiver command generates a Receiver resource.",
	Example: `  # Create a Provider for a Slack channel
  gotk create ap slack \
  --type slack \
  --channel general \
  --address https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK \
  --secret-ref webhook-url

  # Create a Provider for a Github repository
  gotk create ap github-podinfo \
  --type github \
  --address https://github.com/stefanprodan/podinfo \
  --secret-ref github-token
`,
	RunE: createReceiverCmdRun,
}

var (
	rcvType      string
	rcvSecretRef string
	rcvEvents    []string
	rcvResources []string
)

func init() {
	createReceiverCmd.Flags().StringVar(&rcvType, "type", "", "")
	createReceiverCmd.Flags().StringVar(&rcvSecretRef, "secret-ref", "", "")
	createReceiverCmd.Flags().StringArrayVar(&rcvEvents, "events", []string{}, "")
	createReceiverCmd.Flags().StringArrayVar(&rcvResources, "resource", []string{}, "")
	createCmd.AddCommand(createReceiverCmd)
}

func createReceiverCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("receiver name is required")
	}
	name := args[0]

	if rcvType == "" {
		return fmt.Errorf("type is required")
	}

	if rcvSecretRef == "" {
		return fmt.Errorf("secret ref is required")
	}

	resources := []notificationv1.CrossNamespaceObjectReference{}
	for _, resource := range rcvResources {
		kind, name := utils.parseObjectKindName(resource)
		if kind == "" {
			return fmt.Errorf("invalid event source '%s', must be in format <kind>/<name>", resource)
		}

		resources = append(resources, notificationv1.CrossNamespaceObjectReference{
			Kind: kind,
			Name: name,
		})
	}

	if len(resources) == 0 {
		return fmt.Errorf("atleast one resource is required")
	}

	sourceLabels, err := parseLabels()
	if err != nil {
		return err
	}

	if !export {
		logger.Generatef("generating receiver")
	}

	receiver := notificationv1.Receiver{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    sourceLabels,
		},
		Spec: notificationv1.ReceiverSpec{
			Type:      rcvType,
			Events:    rcvEvents,
			Resources: resources,
			SecretRef: corev1.LocalObjectReference{
				Name: rcvSecretRef,
			},
			Suspend: false,
		},
	}

	if export {
		return exportReceiver(receiver)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	logger.Actionf("applying receiver")
	if err := upsertReceiver(ctx, kubeClient, receiver); err != nil {
		return err
	}

	logger.Waitingf("waiting for reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isReceiverReady(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	logger.Successf("receiver %s is ready", name)

	return nil
}

func upsertReceiver(ctx context.Context, kubeClient client.Client, receiver notificationv1.Receiver) error {
	namespacedName := types.NamespacedName{
		Namespace: receiver.GetNamespace(),
		Name:      receiver.GetName(),
	}

	var existing notificationv1.Receiver
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, &receiver); err != nil {
				return err
			} else {
				logger.Successf("receiver created")
				return nil
			}
		}
		return err
	}

	existing.Labels = receiver.Labels
	existing.Spec = receiver.Spec
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return err
	}

	logger.Successf("receiver updated")
	return nil
}

func isReceiverReady(ctx context.Context, kubeClient client.Client, name, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		var receiver notificationv1.Receiver
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		err := kubeClient.Get(ctx, namespacedName, &receiver)
		if err != nil {
			return false, err
		}

		if c := meta.GetCondition(receiver.Status.Conditions, meta.ReadyCondition); c != nil {
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
