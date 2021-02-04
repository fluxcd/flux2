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
	"strings"
	"time"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/aggregator"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/collector"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/fluxcd/flux2/internal/utils"
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
	pollInterval time.Duration
	timeout      time.Duration
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

func (sc *StatusChecker) New(pollInterval time.Duration, timeout time.Duration) error {
	kubeConfig, err := utils.KubeConfig(rootArgs.kubeconfig, rootArgs.kubecontext)
	if err != nil {
		return err
	}
	restMapper, err := apiutil.NewDynamicRESTMapper(kubeConfig)
	if err != nil {
		return err
	}
	client, err := client.New(kubeConfig, client.Options{Mapper: restMapper})
	if err != nil {
		return err
	}
	statusPoller := polling.NewStatusPoller(client, restMapper)
	sc.statusPoller = statusPoller
	sc.pollInterval = pollInterval
	sc.timeout = timeout
	return err
}

func (sc *StatusChecker) Assess(components ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), sc.timeout)
	defer cancel()

	objRefs, err := sc.getObjectRefs(components)
	if err != nil {
		return err
	}

	opts := polling.Options{PollInterval: sc.pollInterval, UseCache: true}
	eventsChan := sc.statusPoller.Poll(ctx, objRefs, opts)

	coll := collector.NewResourceStatusCollector(objRefs)
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
		return fmt.Errorf("Status check timed out for component(s): [%v]", strings.Join(ids, ", "))
	}
	return nil
}

func (sc *StatusChecker) getObjectRefs(components []string) ([]object.ObjMetadata, error) {
	var objRefs []object.ObjMetadata
	for _, deployment := range components {
		objMeta, err := object.CreateObjMetadata(rootArgs.namespace, deployment, schema.GroupKind{Group: "apps", Kind: "Deployment"})
		if err != nil {
			return nil, err
		}
		objRefs = append(objRefs, objMeta)
	}
	return objRefs, nil
}

func (sc *StatusChecker) objMetadataToString(om object.ObjMetadata) string {
	return fmt.Sprintf("%s '%s/%s'", om.GroupKind.Kind, om.Namespace, om.Name)
}
