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

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/pkg/apis/meta"
)

// statusable is used to see if a resource is considered ready in the usual way
type statusable interface {
	adapter
	// this is implemented by ObjectMeta
	GetGeneration() int64
	getObservedGeneration() int64
	// this is usually implemented by GOTK API objects because it's used by pkg/apis/meta
	GetStatusConditions() *[]metav1.Condition
}

type StatusChecker struct {
	client client.Client
	timeout time.Duration
	objRefs []object.ObjMetadata
	statusPoller *polling.StatusPoller
}

func isReady(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, object statusable) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, object.asClientObject())
		if err != nil {
			return false, err
		}

		// Confirm the state we are observing is for the current generation
		if object.GetGeneration() != object.getObservedGeneration() {
			return false, nil
		}

		if c := apimeta.FindStatusCondition(*object.GetStatusConditions(), meta.ReadyCondition); c != nil {
			switch c.Status {
			case metav1.ConditionTrue:
				return true, nil
			case metav1.ConditionFalse:
				return false, fmt.Errorf(c.Message)
			}
		}
		return false, nil
	}
}

func (sc *StatusChecker) New(kubeConfig *rest.Config, timeout time.Duration) error {
	restMapper, err := apiutil.NewDynamicRESTMapper(kubeConfig)
	if err != nil {
		return err
	}
	client, err := client.New(kubeConfig, client.Options{Mapper: restMapper})
	if err != nil {
		return err
	}
	statusPoller := polling.NewStatusPoller(client, restMapper)
	sc.client = client
	sc.statusPoller = statusPoller
	sc.timeout = timeout
	return err
}

func (sc *StatusChecker) AddChecks(components []string) error {
	var componentRefs []object.ObjMetadata
	for _, deployment := range components {
		objMeta, err := object.CreateObjMetadata(rootArgs.namespace, deployment, schema.GroupKind{Group: "apps", Kind: "Deployment"})
		if err != nil {
			return err
		}
		componentRefs = append(componentRefs, objMeta)
	}
	sc.objRefs = componentRefs
	return nil
}

func (sc *StatusChecker) Assess(pollInterval time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), sc.timeout)
	defer cancel()

	opts := polling.Options{PollInterval: pollInterval, UseCache: true}
	eventsChan := sc.statusPoller.Poll(ctx, sc.objRefs, opts)
	coll := collector.NewResourceStatusCollector(sc.objRefs)
	done := coll.ListenWithObserver(eventsChan, collector.ObserverFunc(
		func(statusCollector *collector.ResourceStatusCollector, e event.Event) {
			var rss []*event.ResourceStatus
			for _, rs := range statusCollector.ResourceStatuses {
				rss = append(rss, rs)
			}
			desired := status.CurrentStatus
			aggStatus := aggregator.AggregateStatus(rss, desired)
			if aggStatus == desired {
				cancel()
				return
			}
		}),
	)
	<-done


	if coll.Error != nil {
		return coll.Error
	}

	if ctx.Err() == context.DeadlineExceeded {
		ids := []string{}
		for _, rs := range coll.ResourceStatuses {
			if rs.Status != status.CurrentStatus {
				id := sc.objMetadataToString(rs.Identifier)
				ids = append(ids, id)
			}
		}
		return fmt.Errorf("Health check timed out for [%v]", strings.Join(ids, ", "))
	}
	return nil
}

func (sc *StatusChecker) toObjMetadata(cr []meta.NamespacedObjectKindReference) ([]object.ObjMetadata, error) {
	oo := []object.ObjMetadata{}
	for _, c := range cr {
		// For backwards compatibility
		if c.APIVersion == "" {
			c.APIVersion = "apps/v1"
		}

		gv, err := schema.ParseGroupVersion(c.APIVersion)
		if err != nil {
			return []object.ObjMetadata{}, err
		}

		gk := schema.GroupKind{Group: gv.Group, Kind: c.Kind}
		o, err := object.CreateObjMetadata(c.Namespace, c.Name, gk)
		if err != nil {
			return []object.ObjMetadata{}, err
		}

		oo = append(oo, o)
	}
	return oo, nil
}

func (sc *StatusChecker) objMetadataToString(om object.ObjMetadata) string {
	return fmt.Sprintf("%s '%s/%s'", om.GroupKind.Kind, om.Namespace, om.Name)
} 
