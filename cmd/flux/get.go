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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	watchtools "k8s.io/client-go/tools/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/pkg/apis/meta"

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/printers"
)

type deriveType func(runtime.Object) (summarisable, error)

type typeMap map[string]deriveType

func (m typeMap) registerCommand(t string, f deriveType) error {
	if _, ok := m[t]; ok {
		return fmt.Errorf("duplicate type function %s", t)
	}
	m[t] = f
	return nil
}

func (m typeMap) execute(t string, obj runtime.Object) (summarisable, error) {
	f, ok := m[t]
	if !ok {
		return nil, fmt.Errorf("unsupported type %s", t)
	}
	return f(obj)
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the resources and their status",
	Long:  `The get sub-commands print the statuses of Flux resources.`,
}

type GetFlags struct {
	allNamespaces  bool
	noHeader       bool
	statusSelector string
	labelSelector  string
	watch          bool
}

var getArgs GetFlags

func init() {
	getCmd.PersistentFlags().BoolVarP(&getArgs.allNamespaces, "all-namespaces", "A", false,
		"list the requested object(s) across all namespaces")
	getCmd.PersistentFlags().BoolVarP(&getArgs.noHeader, "no-header", "", false, "skip the header when printing the results")
	getCmd.PersistentFlags().BoolVarP(&getArgs.watch, "watch", "w", false, "After listing/getting the requested object, watch for changes.")
	getCmd.PersistentFlags().StringVar(&getArgs.statusSelector, "status-selector", "",
		"specify the status condition name and the desired state to filter the get result, e.g. ready=false")
	getCmd.PersistentFlags().StringVarP(&getArgs.labelSelector, "label-selector", "l", "",
		"filter objects by label selector")
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
	list    summarisable
	funcMap typeMap
}

func (get getCommand) run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	var listOpts []client.ListOption
	if !getArgs.allNamespaces {
		listOpts = append(listOpts, client.InNamespace(*kubeconfigArgs.Namespace))
	}

	if len(args) > 0 {
		listOpts = append(listOpts, client.MatchingFields{"metadata.name": args[0]})
	}

	if getArgs.labelSelector != "" {
		label, err := metav1.ParseToLabelSelector(getArgs.labelSelector)
		if err != nil {
			return fmt.Errorf("unable to parse label selector: %w", err)
		}

		sel, err := metav1.LabelSelectorAsSelector(label)
		if err != nil {
			return err
		}
		listOpts = append(listOpts, client.MatchingLabelsSelector{
			Selector: sel,
		})
	}

	getAll := cmd.Use == "all"

	if getArgs.watch {
		return get.watch(ctx, kubeClient, cmd, args, listOpts)
	}

	err = kubeClient.List(ctx, get.list.asClientList(), listOpts...)
	if err != nil {
		if getAll && apimeta.IsNoMatchError(err) {
			return nil
		}
		return err
	}

	if get.list.len() == 0 {
		if len(args) > 0 {
			logger.Failuref("%s object '%s' not found in %s namespace",
				get.kind,
				args[0],
				namespaceNameOrAny(getArgs.allNamespaces, *kubeconfigArgs.Namespace),
			)
		} else if !getAll {
			logger.Failuref("no %s objects found in %s namespace",
				get.kind,
				namespaceNameOrAny(getArgs.allNamespaces, *kubeconfigArgs.Namespace),
			)
		}
		return nil
	}

	var header []string
	if !getArgs.noHeader {
		header = get.list.headers(getArgs.allNamespaces)
	}

	rows, err := getRowsToPrint(getAll, get.list)
	if err != nil {
		return err
	}

	err = printers.TablePrinter(header).Print(cmd.OutOrStdout(), rows)
	if err != nil {
		return err
	}

	if getAll {
		fmt.Println()
	}

	return nil
}

func namespaceNameOrAny(allNamespaces bool, namespaceName string) string {
	if allNamespaces {
		return "any"
	}
	return fmt.Sprintf("%q", namespaceName)
}

func getRowsToPrint(getAll bool, list summarisable) ([][]string, error) {
	noFilter := true
	var conditionType, conditionStatus string
	if getArgs.statusSelector != "" {
		parts := strings.SplitN(getArgs.statusSelector, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("expected status selector in type=status format, but found: %s", getArgs.statusSelector)
		}
		conditionType = parts[0]
		conditionStatus = parts[1]
		noFilter = false
	}
	var rows [][]string
	for i := 0; i < list.len(); i++ {
		if noFilter || list.statusSelectorMatches(i, conditionType, conditionStatus) {
			row := list.summariseItem(i, getArgs.allNamespaces, getAll)
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// watch starts a client-side watch of one or more resources.
func (get *getCommand) watch(ctx context.Context, kubeClient client.WithWatch, cmd *cobra.Command, args []string, listOpts []client.ListOption) error {
	w, err := kubeClient.Watch(ctx, get.list.asClientList(), listOpts...)
	if err != nil {
		return err
	}

	_, err = watchUntil(ctx, w, get)
	if err != nil {
		return err
	}

	return nil
}

func watchUntil(ctx context.Context, w watch.Interface, get *getCommand) (bool, error) {
	firstIteration := true
	_, error := watchtools.UntilWithoutRetry(ctx, w, func(e watch.Event) (bool, error) {
		objToPrint := e.Object
		sink, err := get.funcMap.execute(get.apiType.kind, objToPrint)
		if err != nil {
			return false, err
		}

		var header []string
		if !getArgs.noHeader {
			header = sink.headers(getArgs.allNamespaces)
		}
		rows, err := getRowsToPrint(false, sink)
		if err != nil {
			return false, err
		}
		if firstIteration {
			err = printers.TablePrinter(header).Print(os.Stdout, rows)
			if err != nil {
				return false, err
			}
			firstIteration = false
		} else {
			err = printers.TablePrinter([]string{}).Print(os.Stdout, rows)
			if err != nil {
				return false, err
			}
		}

		return false, nil
	})

	return false, error
}

func validateWatchOption(cmd *cobra.Command, toMatch string) error {
	w, _ := cmd.Flags().GetBool("watch")
	if cmd.Use == toMatch && w {
		return fmt.Errorf("expected a single resource type, but found %s", cmd.Use)
	}
	return nil
}
