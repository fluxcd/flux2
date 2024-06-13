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
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create or update sources and resources",
	Long:  `The create sub-commands generate sources and resources.`,
}

type createFlags struct {
	interval time.Duration
	export   bool
	labels   []string
}

var createArgs createFlags

func init() {
	createCmd.PersistentFlags().DurationVarP(&createArgs.interval, "interval", "", time.Minute, "source sync interval")
	createCmd.PersistentFlags().BoolVar(&createArgs.export, "export", false, "export in YAML format to stdout")
	createCmd.PersistentFlags().StringSliceVar(&createArgs.labels, "label", nil,
		"set labels on the resource (can specify multiple labels with commas: label1=value1,label2=value2)")
	createCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("name is required")
		}

		name := args[0]
		if !validateObjectName(name) {
			return fmt.Errorf("name '%s' is invalid, it should adhere to standard defined in RFC 1123, the name can only contain alphanumeric characters or '-'", name)
		}

		return nil
	}
	rootCmd.AddCommand(createCmd)
}

// upsertable is an interface for values that can be used in `upsert`.
type upsertable interface {
	adapter
	named
}

// upsert updates or inserts an object. Instead of providing the
// object itself, you provide a named (as in Name and Namespace)
// template value, and a mutate function which sets the values you
// want to update. The mutate function is nullary -- you mutate a
// value in the closure, e.g., by doing this:
//
//	var existing Value
//	existing.Name = name
//	existing.Namespace = ns
//	upsert(ctx, client, valueAdapter{&value}, func() error {
//	  value.Spec = onePreparedEarlier
//	})
func (names apiType) upsert(ctx context.Context, kubeClient client.Client, object upsertable, mutate func() error) (types.NamespacedName, error) {
	nsname := types.NamespacedName{
		Namespace: object.GetNamespace(),
		Name:      object.GetName(),
	}

	op, err := controllerutil.CreateOrUpdate(ctx, kubeClient, object.asClientObject(), mutate)
	if err != nil {
		return nsname, err
	}

	switch op {
	case controllerutil.OperationResultCreated:
		logger.Successf("%s created", names.kind)
	case controllerutil.OperationResultUpdated:
		logger.Successf("%s updated", names.kind)
	}
	return nsname, nil
}

type upsertWaitable interface {
	upsertable
	statusable
}

// upsertAndWait encodes the pattern of creating or updating a
// resource, then waiting for it to reconcile. See the note on
// `upsert` for how to work with the `mutate` argument.
func (names apiType) upsertAndWait(object upsertWaitable, mutate func() error) error {
	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions) // NB globals
	if err != nil {
		return err
	}

	logger.Generatef("generating %s", names.kind)
	logger.Actionf("applying %s", names.kind)

	namespacedName, err := imageRepositoryType.upsert(ctx, kubeClient, object, mutate)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for %s reconciliation", names.kind)
	if err := wait.PollUntilContextTimeout(ctx, rootArgs.pollInterval, rootArgs.timeout, true,
		isObjectReadyConditionFunc(kubeClient, namespacedName, object.asClientObject())); err != nil {
		return err
	}
	logger.Successf("%s reconciliation completed", names.kind)
	return nil
}

func parseLabels() (map[string]string, error) {
	result := make(map[string]string)
	for _, label := range createArgs.labels {
		// validate key value pair
		parts := strings.Split(label, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid label format '%s', must be key=value", label)
		}

		// validate label name
		if errors := validation.IsQualifiedName(parts[0]); len(errors) > 0 {
			return nil, fmt.Errorf("invalid label '%s': %v", parts[0], errors)
		}

		// validate label value
		if errors := validation.IsValidLabelValue(parts[1]); len(errors) > 0 {
			return nil, fmt.Errorf("invalid label value '%s': %v", parts[1], errors)
		}

		result[parts[0]] = parts[1]
	}

	return result, nil
}

func validateObjectName(name string) bool {
	r := regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]){0,61}[a-z0-9]$`)
	return r.MatchString(name)
}
