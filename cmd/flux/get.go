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
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"

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
	Example: `  # List all Helm releases
  flux get helmreleases

  # Get a Helm release using the kubectl-style TYPE/NAME form
  flux get hr/podinfo

  # Get several Kustomizations by name
  flux get ks frontend backend

  # Get a Git repository by its kind, short name or fully qualified type
  flux get gitrepository/podinfo
  flux get gitrepo/podinfo
  flux get gitrepositories.source.toolkit.fluxcd.io podinfo

  # List all OCI repositories using a qualified type
  flux get ocirepositories.v1.source.toolkit.fluxcd.io`,
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
		"specify the status condition name and the desired state to filter the get result, e.g. ready=false or ready!=true")
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

func readyStatusMatches(conditionType, conditionStatus string) bool {
	return strings.EqualFold(conditionType, meta.ReadyCondition) &&
		strings.EqualFold(conditionStatus, string(metav1.ConditionTrue))
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
	// found, when set, records the names found with several names and
	// suppresses their not-found errors; used by the "all" commands.
	found map[string]bool
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

	names := uniqueNames(args)
	if len(names) == 1 {
		listOpts = append(listOpts, client.MatchingFields{"metadata.name": names[0]})
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
		return get.watch(ctx, kubeClient, cmd, names, listOpts)
	}

	var notFoundErr error
	if len(names) > 1 {
		missing, err := get.listByNames(ctx, kubeClient, names, listOpts)
		if err != nil {
			if getAll && apimeta.IsNoMatchError(err) {
				return nil
			}
			return err
		}
		if get.found == nil {
			var errs []error
			for _, name := range missing {
				errs = append(errs, notFoundError(get.kind, name))
			}
			if len(errs) == 1 {
				notFoundErr = errs[0]
			} else {
				notFoundErr = errors.Join(errs...)
			}
		}
	} else {
		err = kubeClient.List(ctx, get.list.asClientList(), listOpts...)
		if err != nil {
			if getAll && apimeta.IsNoMatchError(err) {
				return nil
			}
			return err
		}
	}

	if get.list.len() == 0 {
		if len(names) == 1 {
			return notFoundError(get.kind, names[0])
		} else if len(names) > 1 {
			return notFoundErr
		} else if !getAll {
			return fmt.Errorf("no %s objects found in %s namespace",
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
	if getAll && len(rows) == 0 {
		return notFoundErr
	}

	err = printers.TablePrinter(header).Print(cmd.OutOrStdout(), rows)
	if err != nil {
		return err
	}

	if getAll {
		fmt.Println()
	}

	return notFoundErr
}

func notFoundError(kind, name string) error {
	return fmt.Errorf("%s object '%s' not found in %s namespace",
		kind,
		name,
		namespaceNameOrAny(getArgs.allNamespaces, *kubeconfigArgs.Namespace),
	)
}

// listByNames fills get.list with one name-scoped List per name, in the
// order given, and returns the names that matched no object. When get.found
// is set, the names found are recorded in it.
func (get getCommand) listByNames(ctx context.Context, kubeClient client.Client, names []string, listOpts []client.ListOption) ([]string, error) {
	list := get.list.asClientList()
	var items []runtime.Object
	var missing []string
	for _, name := range names {
		nameList, ok := list.DeepCopyObject().(client.ObjectList)
		if !ok {
			return nil, fmt.Errorf("unexpected list type %T", list)
		}
		opts := append(slices.Clone(listOpts), client.MatchingFields{"metadata.name": name})
		if err := kubeClient.List(ctx, nameList, opts...); err != nil {
			return nil, err
		}
		nameItems, err := apimeta.ExtractList(nameList)
		if err != nil {
			return nil, err
		}
		if len(nameItems) == 0 {
			missing = append(missing, name)
		} else if get.found != nil {
			get.found[name] = true
		}
		items = append(items, nameItems...)
	}
	return missing, apimeta.SetList(list, items)
}

// missingNamesError returns an error for the names in args not recorded in
// found, when args has several names.
func missingNamesError(args []string, found map[string]bool) error {
	names := uniqueNames(args)
	if len(names) < 2 {
		return nil
	}
	var errs []error
	for _, name := range names {
		if !found[name] {
			errs = append(errs, fmt.Errorf("object '%s' not found in %s namespace",
				name, namespaceNameOrAny(getArgs.allNamespaces, *kubeconfigArgs.Namespace)))
		}
	}
	return errors.Join(errs...)
}

func namespaceNameOrAny(allNamespaces bool, namespaceName string) string {
	if allNamespaces {
		return "any"
	}
	return fmt.Sprintf("%q", namespaceName)
}

func getRowsToPrint(getAll bool, list summarisable) ([][]string, error) {
	filter := func(i int) bool { return true }
	var conditionType, conditionStatus string
	if getArgs.statusSelector != "" {
		// Support both type=status (match) and type!=status (negated match).
		// "!=" must be checked first since it also contains "=".
		separator := "="
		filter = func(i int) bool {
			return list.statusSelectorMatches(i, conditionType, conditionStatus)
		}
		if strings.Contains(getArgs.statusSelector, "!=") {
			separator = "!="
			filter = func(i int) bool {
				return !list.statusSelectorMatches(i, conditionType, conditionStatus)
			}
		}
		parts := strings.SplitN(getArgs.statusSelector, separator, 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("expected status selector in type=status or type!=status format, but found: %s", getArgs.statusSelector)
		}
		conditionType = parts[0]
		conditionStatus = parts[1]
	}
	var rows [][]string
	for i := 0; i < list.len(); i++ {
		if filter(i) {
			row := list.summariseItem(i, getArgs.allNamespaces, getAll)
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// watch starts a client-side watch of one or more resources.
func (get *getCommand) watch(ctx context.Context, kubeClient client.WithWatch, cmd *cobra.Command, names []string, listOpts []client.ListOption) error {
	var w watch.Interface
	if len(names) > 1 {
		watches := make([]watch.Interface, 0, len(names))
		for _, name := range names {
			opts := append(slices.Clone(listOpts), client.MatchingFields{"metadata.name": name})
			nw, err := kubeClient.Watch(ctx, get.list.asClientList(), opts...)
			if err != nil {
				for _, w := range watches {
					w.Stop()
				}
				return err
			}
			watches = append(watches, nw)
		}
		w = mergeWatches(watches)
	} else {
		var err error
		w, err = kubeClient.Watch(ctx, get.list.asClientList(), listOpts...)
		if err != nil {
			return err
		}
	}

	_, err := watchUntil(ctx, w, get)
	if err != nil {
		return err
	}

	return nil
}

// mergedWatch merges several watches into one. It closes when any of them
// closes, and Stop stops all of them.
type mergedWatch struct {
	watches  []watch.Interface
	result   chan watch.Event
	done     chan struct{}
	stopOnce sync.Once
}

func mergeWatches(watches []watch.Interface) watch.Interface {
	m := &mergedWatch{
		watches: watches,
		result:  make(chan watch.Event),
		done:    make(chan struct{}),
	}
	var wg sync.WaitGroup
	for _, w := range watches {
		wg.Add(1)
		go func(w watch.Interface) {
			defer wg.Done()
			defer m.Stop()
			for e := range w.ResultChan() {
				select {
				case m.result <- e:
				case <-m.done:
					return
				}
			}
		}(w)
	}
	go func() {
		wg.Wait()
		close(m.result)
	}()
	return m
}

func (m *mergedWatch) Stop() {
	m.stopOnce.Do(func() {
		close(m.done)
		for _, w := range m.watches {
			w.Stop()
		}
	})
}

func (m *mergedWatch) ResultChan() <-chan watch.Event {
	return m.result
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
