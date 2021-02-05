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
	"os"

	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/pkg/apis/meta"

	"github.com/fluxcd/flux2/internal/utils"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get sources and resources",
	Long:  "The get sub-commands print the statuses of sources and resources.",
}

type GetFlags struct {
	allNamespaces bool
}

var getArgs GetFlags

func init() {
	getCmd.PersistentFlags().BoolVarP(&getArgs.allNamespaces, "all-namespaces", "A", false,
		"list the requested object(s) across all namespaces")
	rootCmd.AddCommand(getCmd)
}

type summarisable interface {
	listAdapter
	summariseItem(i int, includeNamespace bool) []string
	headers(includeNamespace bool) []string
}

// --- these help with implementations of summarisable

func statusAndMessage(conditions []metav1.Condition) (string, string) {
	if c := apimeta.FindStatusCondition(conditions, meta.ReadyCondition); c != nil {
		return string(c.Status), c.Message
	}
	return string(metav1.ConditionFalse), "waiting to be reconciled"
}

func nameColumns(item named, includeNamespace bool) []string {
	if includeNamespace {
		return []string{item.GetNamespace(), item.GetName()}
	}
	return []string{item.GetName()}
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

	if get.list.len() == 0 {
		logger.Failuref("no %s objects found in %s namespace", get.kind, rootArgs.namespace)
		return nil
	}

	header := get.list.headers(getArgs.allNamespaces)
	var rows [][]string
	for i := 0; i < get.list.len(); i++ {
		row := get.list.summariseItem(i, getArgs.allNamespaces)
		rows = append(rows, row)
	}
	utils.PrintTable(os.Stdout, header, rows)
	return nil
}
