/*
Copyright 2023 The Kubernetes Authors.
Copyright 2023 The Flux authors

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
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/apimachinery/pkg/watch"
	runtimeresource "k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta2"
	autov1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notificationv1 "github.com/fluxcd/notification-controller/api/v1"
	notificationv1b3 "github.com/fluxcd/notification-controller/api/v1beta3"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/printers"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Display Kubernetes events for Flux resources",
	Long:  withPreviewNote("The events sub-command shows Kubernetes events from Flux resources"),
	Example: ` # Display events for flux resources in default namespace
	flux events -n default
	
	# Display events for flux resources in all namespaces
	flux events -A

	# Display events for a Kustomization named podinfo
	flux events --for Kustomization/podinfo

	# Display events for all Kustomizations in default namespace
	flux events --for Kustomization -n default

	# Display warning events for alert resources
	flux events --for Alert/podinfo --types warning
`,
	RunE: eventsCmdRun,
}

type eventFlags struct {
	allNamespaces bool
	watch         bool
	forSelector   string
	filterTypes   []string
}

var eventArgs eventFlags

func init() {
	eventsCmd.Flags().BoolVarP(&eventArgs.allNamespaces, "all-namespaces", "A", false,
		"display events from Flux resources across all namespaces")
	eventsCmd.Flags().BoolVarP(&eventArgs.watch, "watch", "w", false,
		"indicate if the events should be streamed")
	eventsCmd.Flags().StringVar(&eventArgs.forSelector, "for", "",
		"get events for a particular object")
	eventsCmd.Flags().StringSliceVar(&eventArgs.filterTypes, "types", []string{}, "filter events for certain types (valid types are: Normal, Warning)")
	rootCmd.AddCommand(eventsCmd)
}

func eventsCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	if err := validateEventTypes(eventArgs.filterTypes); err != nil {
		return err
	}

	kubeclient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	namespace := *kubeconfigArgs.Namespace
	if eventArgs.allNamespaces {
		namespace = ""
	}

	var diffRefNs bool
	clientListOpts := []client.ListOption{client.InNamespace(*kubeconfigArgs.Namespace)}
	var refListOpts [][]client.ListOption
	if eventArgs.forSelector != "" {
		kind, name := getKindNameFromSelector(eventArgs.forSelector)
		if kind == "" {
			return fmt.Errorf("--for selector must be of format <kind>[/<name>]")
		}

		refInfoKind, err := fluxKindMap.getRefInfo(kind)
		if err != nil {
			return err
		}
		clientListOpts = append(clientListOpts, getListOpt(refInfoKind.gvk.Kind, name))
		if name != "" {
			refs, err := getObjectRef(ctx, kubeclient, refInfoKind, name, *kubeconfigArgs.Namespace)
			if err != nil {
				return err
			}

			for _, ref := range refs {
				refKind, refName, refNs := utils.ParseObjectKindNameNamespace(ref)
				if refNs != namespace {
					diffRefNs = true
				}
				refOpt := []client.ListOption{getListOpt(refKind, refName), client.InNamespace(refNs)}
				refListOpts = append(refListOpts, refOpt)
			}
		}
	}

	showNamespace := namespace == "" || diffRefNs
	if eventArgs.watch {
		return eventsCmdWatchRun(ctx, kubeclient, clientListOpts, refListOpts, showNamespace)
	}

	rows, err := getRows(ctx, kubeclient, clientListOpts, refListOpts, showNamespace)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		if eventArgs.allNamespaces {
			logger.Failuref("No events found.")
		} else {
			logger.Failuref("No events found in %s namespace.", *kubeconfigArgs.Namespace)
		}

		return nil
	}
	headers := getHeaders(showNamespace)
	return printers.TablePrinter(headers).Print(cmd.OutOrStdout(), rows)
}

func getRows(ctx context.Context, kubeclient client.Client, clientListOpts []client.ListOption, refListOpts [][]client.ListOption, showNs bool) ([][]string, error) {
	el := &corev1.EventList{}
	if err := addEventsToList(ctx, kubeclient, el, clientListOpts); err != nil {
		return nil, err
	}

	for _, refOpts := range refListOpts {
		if err := addEventsToList(ctx, kubeclient, el, refOpts); err != nil {
			return nil, err
		}
	}

	sort.Sort(SortableEvents(el.Items))

	var rows [][]string
	for _, item := range el.Items {
		if ignoreEvent(item) {
			continue
		}
		rows = append(rows, getEventRow(item, showNs))
	}

	return rows, nil
}

func addEventsToList(ctx context.Context, kubeclient client.Client, el *corev1.EventList, clientListOpts []client.ListOption) error {
	listOpts := &metav1.ListOptions{}
	clientListOpts = append(clientListOpts, client.Limit(cmdutil.DefaultChunkSize))
	err := runtimeresource.FollowContinue(listOpts,
		func(options metav1.ListOptions) (runtime.Object, error) {
			newEvents := &corev1.EventList{}
			if err := kubeclient.List(ctx, newEvents, clientListOpts...); err != nil {
				return nil, fmt.Errorf("error getting events: %w", err)
			}
			el.Items = append(el.Items, newEvents.Items...)
			return newEvents, nil
		})

	return err
}

func getListOpt(kind, name string) client.ListOption {
	var sel fields.Selector
	if name == "" {
		sel = fields.OneTermEqualSelector("involvedObject.kind", kind)
	} else {
		sel = fields.AndSelectors(
			fields.OneTermEqualSelector("involvedObject.kind", kind),
			fields.OneTermEqualSelector("involvedObject.name", name))
	}

	return client.MatchingFieldsSelector{Selector: sel}
}

func eventsCmdWatchRun(ctx context.Context, kubeclient client.WithWatch, listOpts []client.ListOption, refListOpts [][]client.ListOption, showNs bool) error {
	event := &corev1.EventList{}
	listOpts = append(listOpts, client.Limit(cmdutil.DefaultChunkSize))
	eventWatch, err := kubeclient.Watch(ctx, event, listOpts...)
	if err != nil {
		return err
	}

	firstIteration := true

	handleEvent := func(e watch.Event) error {
		if e.Type == watch.Deleted {
			return nil
		}

		event, ok := e.Object.(*corev1.Event)
		if !ok {
			return nil
		}
		if ignoreEvent(*event) {
			return nil
		}
		rows := getEventRow(*event, showNs)
		var hdr []string
		if firstIteration {
			hdr = getHeaders(showNs)
			firstIteration = false
		}
		return printers.TablePrinter(hdr).Print(os.Stdout, [][]string{rows})
	}

	for _, refOpts := range refListOpts {
		refEventWatch, err := kubeclient.Watch(ctx, event, refOpts...)
		if err != nil {
			return err
		}
		go func() {
			if err := receiveEventChan(ctx, refEventWatch, handleEvent); err != nil {
				logger.Failuref("error watching events: %s", err.Error())
			}
		}()
	}

	return receiveEventChan(ctx, eventWatch, handleEvent)
}

func receiveEventChan(ctx context.Context, eventWatch watch.Interface, f func(e watch.Event) error) error {
	defer eventWatch.Stop()
	for {
		select {
		case e, ok := <-eventWatch.ResultChan():
			if !ok {
				return nil
			}
			err := f(e)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func getHeaders(showNs bool) []string {
	headers := []string{"Last seen", "Type", "Reason", "Object", "Message"}
	if showNs {
		headers = append(namespaceHeader, headers...)
	}

	return headers
}

func getEventRow(e corev1.Event, showNs bool) []string {
	var row []string
	if showNs {
		row = []string{e.Namespace}
	}
	row = append(row, getLastSeen(e), e.Type, e.Reason, fmt.Sprintf("%s/%s", e.InvolvedObject.Kind, e.InvolvedObject.Name), e.Message)

	return row
}

// getObjectRef is used to get the metadata of a resource that the selector(in the format <kind/name>) references.
// It returns an empty string if the resource doesn't reference any resource
// and a string with the format `<kind>/<name>.<namespace>` if it does.
func getObjectRef(ctx context.Context, kubeclient client.Client, ref refInfo, name, ns string) ([]string, error) {
	// the resource has no source ref
	if len(ref.field) == 0 {
		return nil, nil
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    ref.gvk.Kind,
		Version: ref.gvk.Version,
		Group:   ref.gvk.Group,
	})
	objName := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}

	if err := kubeclient.Get(ctx, objName, obj); err != nil {
		return nil, err
	}

	refKind := ref.kind
	if refKind == "" {
		kindField := append(ref.field, "kind")
		specKind, ok, err := unstructured.NestedString(obj.Object, kindField...)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("field '%s' for '%s' not found", strings.Join(kindField, "."), objName)
		}
		refKind = specKind
	}

	nameField := append(ref.field, "name")
	refName, ok, err := unstructured.NestedString(obj.Object, nameField...)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("field '%s' for '%s' not found", strings.Join(nameField, "."), objName)
	}

	var allRefs []string
	refNamespace := ns
	if ref.crossNamespaced {
		namespaceField := append(ref.field, "namespace")
		namespace, ok, err := unstructured.NestedString(obj.Object, namespaceField...)
		if err != nil {
			return nil, err
		}
		if ok {
			refNamespace = namespace
		}
	}

	allRefs = append(allRefs, fmt.Sprintf("%s/%s.%s", refKind, refName, refNamespace))
	if ref.otherRefs != nil {
		for _, otherRef := range ref.otherRefs(ns, name) {
			allRefs = append(allRefs, fmt.Sprintf("%s.%s", otherRef, refNamespace))
		}
	}
	return allRefs, nil
}

type refMap map[string]refInfo

func (r refMap) getRefInfo(kind string) (refInfo, error) {
	for key, ref := range r {
		if strings.EqualFold(key, kind) {
			return ref, nil
		}
	}
	return refInfo{}, fmt.Errorf("'%s' is not a recognized Flux kind", kind)
}

func (r refMap) hasKind(kind string) bool {
	_, err := r.getRefInfo(kind)
	return err == nil
}

// validateEventTypes checks that the event types passed into the function
// is either equal to `Normal` or `Warning` which are currently the two supported types.
// https://github.com/kubernetes/kubernetes/blob/a8a1abc25cad87333840cd7d54be2efaf31a3177/staging/src/k8s.io/api/core/v1/types.go#L6212
func validateEventTypes(eventTypes []string) error {
	for _, t := range eventTypes {
		if !strings.EqualFold(corev1.EventTypeWarning, t) && !strings.EqualFold(corev1.EventTypeNormal, t) {
			return fmt.Errorf("type '%s' not supported. Supported types are Normal, Warning", t)
		}
	}

	return nil
}

type refInfo struct {
	// gvk is the group version kind of the resource
	gvk schema.GroupVersionKind
	// kind is the kind that the resource references if it's not static
	kind string
	// crossNamespaced indicates if this resource uses cross namespaced references
	crossNamespaced bool
	// otherRefs returns other reference that might not be directly accessible
	// from the spec of the object
	otherRefs func(namespace, name string) []string
	field     []string
}

var fluxKindMap = refMap{
	kustomizev1.KustomizationKind: {
		gvk:             kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind),
		crossNamespaced: true,
		field:           []string{"spec", "sourceRef"},
	},
	helmv2.HelmReleaseKind: {
		gvk:             helmv2.GroupVersion.WithKind(helmv2.HelmReleaseKind),
		crossNamespaced: true,
		otherRefs: func(namespace, name string) []string {
			return []string{fmt.Sprintf("%s/%s-%s", sourcev1b2.HelmChartKind, namespace, name)}
		},
		field: []string{"spec", "chart", "spec", "sourceRef"},
	},
	notificationv1b3.AlertKind: {
		gvk:             notificationv1b3.GroupVersion.WithKind(notificationv1b3.AlertKind),
		kind:            notificationv1b3.ProviderKind,
		crossNamespaced: false,
		field:           []string{"spec", "providerRef"},
	},
	notificationv1.ReceiverKind:   {gvk: notificationv1.GroupVersion.WithKind(notificationv1.ReceiverKind)},
	notificationv1b3.ProviderKind: {gvk: notificationv1b3.GroupVersion.WithKind(notificationv1b3.ProviderKind)},
	imagev1.ImagePolicyKind: {
		gvk:             imagev1.GroupVersion.WithKind(imagev1.ImagePolicyKind),
		kind:            imagev1.ImageRepositoryKind,
		crossNamespaced: true,
		field:           []string{"spec", "imageRepositoryRef"},
	},
	sourcev1b2.HelmChartKind: {
		gvk:             sourcev1b2.GroupVersion.WithKind(sourcev1b2.HelmChartKind),
		crossNamespaced: true,
		field:           []string{"spec", "sourceRef"},
	},
	sourcev1.GitRepositoryKind:       {gvk: sourcev1.GroupVersion.WithKind(sourcev1.GitRepositoryKind)},
	sourcev1b2.OCIRepositoryKind:     {gvk: sourcev1b2.GroupVersion.WithKind(sourcev1b2.OCIRepositoryKind)},
	sourcev1b2.BucketKind:            {gvk: sourcev1b2.GroupVersion.WithKind(sourcev1b2.BucketKind)},
	sourcev1b2.HelmRepositoryKind:    {gvk: sourcev1b2.GroupVersion.WithKind(sourcev1b2.HelmRepositoryKind)},
	autov1.ImageUpdateAutomationKind: {gvk: autov1.GroupVersion.WithKind(autov1.ImageUpdateAutomationKind)},
	imagev1.ImageRepositoryKind:      {gvk: imagev1.GroupVersion.WithKind(imagev1.ImageRepositoryKind)},
}

func ignoreEvent(e corev1.Event) bool {
	if !fluxKindMap.hasKind(e.InvolvedObject.Kind) {
		return true
	}

	if len(eventArgs.filterTypes) > 0 {
		_, equal := utils.ContainsEqualFoldItemString(eventArgs.filterTypes, e.Type)
		if !equal {
			return true
		}
	}

	return false
}

func getKindNameFromSelector(selector string) (string, string) {
	kind, name := utils.ParseObjectKindName(selector)
	// if there's no slash in the selector utils.ParseObjectKindName returns the
	// input string as the name but here we want it as the kind instead
	if kind == "" && name != "" {
		kind = name
		name = ""
	}

	return kind, name
}

// The functions below are copied from: https://github.com/kubernetes/kubectl/blob/4ecd7bd0f0799f191335a331ca3c6a397a888233/pkg/cmd/events/events.go#L294

// SortableEvents implements sort.Interface for []api.Event by time
type SortableEvents []corev1.Event

func (list SortableEvents) Len() int {
	return len(list)
}

func (list SortableEvents) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

// Return the time that should be used for sorting, which can come from
// various places in corev1.Event.
func eventTime(event corev1.Event) time.Time {
	if event.Series != nil {
		return event.Series.LastObservedTime.Time
	}
	if !event.LastTimestamp.Time.IsZero() {
		return event.LastTimestamp.Time
	}
	return event.EventTime.Time
}

func (list SortableEvents) Less(i, j int) bool {
	return eventTime(list[i]).Before(eventTime(list[j]))
}

func getLastSeen(e corev1.Event) string {
	var interval string
	firstTimestampSince := translateMicroTimestampSince(e.EventTime)
	if e.EventTime.IsZero() {
		firstTimestampSince = translateTimestampSince(e.FirstTimestamp)
	}
	if e.Series != nil {
		interval = fmt.Sprintf("%s (x%d over %s)", translateMicroTimestampSince(e.Series.LastObservedTime), e.Series.Count, firstTimestampSince)
	} else if e.Count > 1 {
		interval = fmt.Sprintf("%s (x%d over %s)", translateTimestampSince(e.LastTimestamp), e.Count, firstTimestampSince)
	} else {
		interval = firstTimestampSince
	}

	return interval
}

// translateMicroTimestampSince returns the elapsed time since timestamp in
// human-readable approximation.
func translateMicroTimestampSince(timestamp metav1.MicroTime) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}

// translateTimestampSince returns the elapsed time since timestamp in
// human-readable approximation.
func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}
