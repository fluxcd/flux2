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
	"os"
	"strings"

	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/pkg/apis/meta"

	"github.com/fluxcd/flux2/internal/utils"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the resources and their status",
	Long:  "The get sub-commands print the statuses of Flux resources.",
}

type GetFlags struct {
	allNamespaces  bool
	noHeader       bool
	statusSelector string
}

var getArgs GetFlags

func init() {
	getCmd.PersistentFlags().BoolVarP(&getArgs.allNamespaces, "all-namespaces", "A", false,
		"list the requested object(s) across all namespaces")
	getCmd.PersistentFlags().BoolVarP(&getArgs.noHeader, "no-header", "", false, "skip the header when printing the results")
	getCmd.PersistentFlags().StringVar(&getArgs.statusSelector, "status-selector", "",
		"specify the status condition name and the desired state to filter the get result, e.g. ready=false")
	rootCmd.AddCommand(getCmd)
}

type summarisable interface {
	listAdapter
	summariseItem(i int, includeNamespace bool, includeKind bool) []string
	headers(includeNamespace bool) []string
	statusSelectorMatches(i int, conditionType, conditionStatus string) bool
}

// --- these help with implementations of summarisable

func statusAndMessage(conditions []metav1.Condition) (string, string) {
	if c := apimeta.FindStatusCondition(conditions, meta.ReadyCondition); c != nil {
		return string(c.Status), c.Message
	}
	return string(metav1.ConditionFalse), "waiting to be reconciled"
}

func statusMatches(conditionType, conditionStatus string, conditions []metav1.Condition) bool {
	// we don't use apimeta.FindStatusCondition because we'd like to use EqualFold to compare two strings
	var c *metav1.Condition
	for i := range conditions {
		if strings.EqualFold(conditions[i].Type, conditionType) {
			c = &conditions[i]
		}
	}
	if c != nil {
		return strings.EqualFold(string(c.Status), conditionStatus)
	}
	return false
}

func nameColumns(item named, includeNamespace bool, includeKind bool) []string {
	name := item.GetName()
	if includeKind {
		name = fmt.Sprintf("%s/%s",
			strings.ToLower(item.GetObjectKind().GroupVersionKind().Kind),
			item.GetName())
	}
	if includeNamespace {
		return []string{item.GetNamespace(), name}
	}
	return []string{name}
}

var namespaceHeader = []string{"Namespace"}

type getCommand struct {
	apiType
	list summarisable
}

func (get getCommand) run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(rootArgs.kubeconfig, rootArgs.kubecontext)
	if err != nil {
		return err
	}

	var listOpts []client.ListOption
	if !getArgs.allNamespaces {
		listOpts = append(listOpts, client.InNamespace(rootArgs.namespace))
	}

	if len(args) > 0 {
		listOpts = append(listOpts, client.MatchingFields{"metadata.name": args[0]})
	}

	err = kubeClient.List(ctx, get.list.asClientList(), listOpts...)
	if err != nil {
		return err
	}

	getAll := cmd.Use == "all"

	if get.list.len() == 0 {
		if !getAll {
			logger.Failuref("no %s objects found in %s namespace", get.kind, rootArgs.namespace)
		}
		return nil
	}

	var header []string
	if !getArgs.noHeader {
		header = get.list.headers(getArgs.allNamespaces)
	}
	noFilter := true
	var conditionType, conditionStatus string
	if getArgs.statusSelector != "" {
		parts := strings.SplitN(getArgs.statusSelector, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("expected status selector in type=status format, but found: %s", getArgs.statusSelector)
		}
		conditionType = parts[0]
		conditionStatus = parts[1]
		noFilter = false
	}
	var rows [][]string
	for i := 0; i < get.list.len(); i++ {
		if noFilter || get.list.statusSelectorMatches(i, conditionType, conditionStatus) {
			row := get.list.summariseItem(i, getArgs.allNamespaces, getAll)
			rows = append(rows, row)
		}
	}
	utils.PrintTable(os.Stdout, header, rows)

	if getAll {
		fmt.Println()
	}
	return nil
}
