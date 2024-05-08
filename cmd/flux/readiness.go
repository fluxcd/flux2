/*
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

	kstatus "github.com/fluxcd/cli-utils/pkg/kstatus/status"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/runtime/object"
	"github.com/fluxcd/pkg/runtime/patch"
)

// objectStatusType is the type of object in terms of status when computing the
// readiness of an object. Readiness check method depends on the type of object.
// For a dynamic object, Ready status condition is considered only for the
// latest generation of the object. For a static object that don't have any
// condition, the object generation is not considered.
type objectStatusType int

const (
	objectStatusDynamic objectStatusType = iota
	objectStatusStatic
)

// isObjectReady determines if an object is ready using the kstatus.Compute()
// result. statusType helps differenciate between static and dynamic objects to
// accurately check the object's readiness. A dynamic object may have some extra
// considerations depending on the object.
func isObjectReady(obj client.Object, statusType objectStatusType) (bool, error) {
	observedGen, err := object.GetStatusObservedGeneration(obj)
	if err != nil && err != object.ErrObservedGenerationNotFound {
		return false, err
	}

	if statusType == objectStatusDynamic {
		// Object not reconciled yet.
		if observedGen < 1 {
			return false, nil
		}

		cobj, ok := obj.(meta.ObjectWithConditions)
		if !ok {
			return false, fmt.Errorf("unable to get conditions from object")
		}

		if c := apimeta.FindStatusCondition(cobj.GetConditions(), meta.ReadyCondition); c != nil {
			// Ensure that the ready condition is for the latest generation of
			// the object.
			// NOTE: Some APIs like ImageUpdateAutomation and HelmRelease don't
			// support per condition observed generation yet. Per condition
			// observed generation for them are always zero.
			// There are two strategies used across different object kinds to
			// check the latest ready condition:
			//   - check that the ready condition's generation matches the
			//     object's generation.
			//   - check that the observed generation of the object in the
			//     status matches the object's generation.
			//
			// TODO: Once ImageUpdateAutomation and HelmRelease APIs have per
			// condition observed generation, remove the object's observed
			// generation and object's generation check (the second condition
			// below). Also, try replacing this readiness check function with
			// fluxcd/pkg/ssa's ResourceManager.Wait(), which uses kstatus
			// internally to check readiness of the objects.
			if c.ObservedGeneration != 0 && c.ObservedGeneration != obj.GetGeneration() {
				return false, nil
			}
			if c.ObservedGeneration == 0 && observedGen != obj.GetGeneration() {
				return false, nil
			}
		} else {
			return false, nil
		}
	}

	u, err := patch.ToUnstructured(obj)
	if err != nil {
		return false, err
	}
	result, err := kstatus.Compute(u)
	if err != nil {
		return false, err
	}
	switch result.Status {
	case kstatus.CurrentStatus:
		return true, nil
	case kstatus.InProgressStatus:
		return false, nil
	default:
		return false, fmt.Errorf(result.Message)
	}
}

// isObjectReadyConditionFunc returns a wait.ConditionFunc to be used with
// wait.Poll* while polling for an object with dynamic status to be ready.
func isObjectReadyConditionFunc(kubeClient client.Client, namespaceName types.NamespacedName, obj client.Object) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		err := kubeClient.Get(ctx, namespaceName, obj)
		if err != nil {
			return false, err
		}

		return isObjectReady(obj, objectStatusDynamic)
	}
}

// isStaticObjectReadyConditionFunc returns a wait.ConditionFunc to be used with
// wait.Poll* while polling for an object with static or no status to be
// ready.
func isStaticObjectReadyConditionFunc(kubeClient client.Client, namespaceName types.NamespacedName, obj client.Object) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		err := kubeClient.Get(ctx, namespaceName, obj)
		if err != nil {
			return false, err
		}

		return isObjectReady(obj, objectStatusStatic)
	}
}

// kstatusCompute returns the kstatus computed result of a given object.
func kstatusCompute(obj client.Object) (result *kstatus.Result, err error) {
	u, err := patch.ToUnstructured(obj)
	if err != nil {
		return result, err
	}
	return kstatus.Compute(u)
}
