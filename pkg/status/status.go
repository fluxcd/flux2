/*
Copyright 2020, 2021 The Flux authors

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

package status

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/cli-utils/pkg/kstatus/polling"
	"github.com/fluxcd/cli-utils/pkg/kstatus/polling/aggregator"
	"github.com/fluxcd/cli-utils/pkg/kstatus/polling/collector"
	"github.com/fluxcd/cli-utils/pkg/kstatus/polling/event"
	"github.com/fluxcd/cli-utils/pkg/kstatus/status"
	"github.com/fluxcd/cli-utils/pkg/object"
	runtimeclient "github.com/fluxcd/pkg/runtime/client"

	"github.com/fluxcd/flux2/v2/pkg/log"
)

type StatusChecker struct {
	pollInterval time.Duration
	timeout      time.Duration
	client       client.Client
	statusPoller *polling.StatusPoller
	logger       log.Logger
}

func NewStatusCheckerWithClient(c client.Client, pollInterval time.Duration, timeout time.Duration, log log.Logger) (*StatusChecker, error) {
	return &StatusChecker{
		pollInterval: pollInterval,
		timeout:      timeout,
		client:       c,
		statusPoller: polling.NewStatusPoller(c, c.RESTMapper(), polling.Options{}),
		logger:       log,
	}, nil
}

func NewStatusChecker(kubeConfig *rest.Config, pollInterval time.Duration, timeout time.Duration, log log.Logger) (*StatusChecker, error) {
	restMapper, err := runtimeclient.NewDynamicRESTMapper(kubeConfig)
	if err != nil {
		return nil, err
	}
	c, err := client.New(kubeConfig, client.Options{Mapper: restMapper})
	if err != nil {
		return nil, err
	}

	return NewStatusCheckerWithClient(c, pollInterval, timeout, log)
}

func (sc *StatusChecker) Assess(identifiers ...object.ObjMetadata) error {
	ctx, cancel := context.WithTimeout(context.Background(), sc.timeout)
	defer cancel()

	opts := polling.PollOptions{PollInterval: sc.pollInterval}
	eventsChan := sc.statusPoller.Poll(ctx, identifiers, opts)

	coll := collector.NewResourceStatusCollector(identifiers)
	done := coll.ListenWithObserver(eventsChan, desiredStatusNotifierFunc(cancel, status.CurrentStatus))

	<-done

	// we use sorted identifiers to loop over the resource statuses because a Go's map is unordered.
	// sorting identifiers by object's name makes sure that the logs look stable for every run
	sort.SliceStable(identifiers, func(i, j int) bool {
		return strings.Compare(identifiers[i].Name, identifiers[j].Name) < 0
	})
	var errs []error
	for _, id := range identifiers {
		rs := coll.ResourceStatuses[id]
		switch rs.Status {
		case status.CurrentStatus:
			sc.logger.Successf("%s: %s ready", rs.Identifier.Name, strings.ToLower(rs.Identifier.GroupKind.Kind))
		case status.NotFoundStatus:
			errMsg := fmt.Sprintf("%s: %s not found", rs.Identifier.Name, strings.ToLower(rs.Identifier.GroupKind.Kind))
			sc.logger.Failuref(errMsg)
			errs = append(errs, errors.New(errMsg))
		default:
			errMsg := fmt.Sprintf("%s: %s not ready", rs.Identifier.Name, strings.ToLower(rs.Identifier.GroupKind.Kind))
			sc.logger.Failuref(errMsg)
			errs = append(errs, errors.New(errMsg))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	if coll.Error != nil || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("timed out waiting for all resources to be ready")
	}
	return nil
}

// desiredStatusNotifierFunc returns an Observer function for the
// ResourceStatusCollector that will cancel the context (using the cancelFunc)
// when all resources have reached the desired status.
func desiredStatusNotifierFunc(cancelFunc context.CancelFunc,
	desired status.Status) collector.ObserverFunc {
	return func(rsc *collector.ResourceStatusCollector, _ event.Event) {
		var rss []*event.ResourceStatus
		for _, rs := range rsc.ResourceStatuses {
			rss = append(rss, rs)
		}
		aggStatus := aggregator.AggregateStatus(rss, desired)
		if aggStatus == desired {
			cancelFunc()
		}
	}
}
