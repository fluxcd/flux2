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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta3"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/runtime/conditions"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

func Test_isObjectReady(t *testing.T) {
	// Ready object.
	readyObj := &sourcev1.GitRepository{}
	readyObj.Generation = 1
	readyObj.Status.ObservedGeneration = 1
	conditions.MarkTrue(readyObj, meta.ReadyCondition, "foo1", "bar1")

	// Not ready object.
	notReadyObj := readyObj.DeepCopy()
	conditions.MarkFalse(notReadyObj, meta.ReadyCondition, "foo2", "bar2")

	// Not reconciled object.
	notReconciledObj := readyObj.DeepCopy()
	notReconciledObj.Status = sourcev1.GitRepositoryStatus{ObservedGeneration: -1}

	// No condition.
	noConditionObj := readyObj.DeepCopy()
	noConditionObj.Status = sourcev1.GitRepositoryStatus{ObservedGeneration: 1}

	// Outdated condition.
	readyObjOutdated := readyObj.DeepCopy()
	readyObjOutdated.Generation = 2

	// Object without per condition observed generation.
	oldObj := readyObj.DeepCopy()
	readyTrueCondn := conditions.TrueCondition(meta.ReadyCondition, "foo3", "bar3")
	oldObj.Status.Conditions = []metav1.Condition{*readyTrueCondn}

	// Outdated object without per condition observed generation.
	oldObjOutdated := oldObj.DeepCopy()
	oldObjOutdated.Generation = 2

	// Empty status object.
	staticObj := readyObj.DeepCopy()
	staticObj.Status = sourcev1.GitRepositoryStatus{}

	// No status object.
	noStatusObj := &notificationv1.Provider{}
	noStatusObj.Generation = 1

	type args struct {
		obj        client.Object
		statusType objectStatusType
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "dynamic ready",
			args: args{obj: readyObj, statusType: objectStatusDynamic},
			want: true,
		},
		{
			name: "dynamic not ready",
			args: args{obj: notReadyObj, statusType: objectStatusDynamic},
			want: false,
		},
		{
			name: "dynamic not reconciled",
			args: args{obj: notReconciledObj, statusType: objectStatusDynamic},
			want: false,
		},
		{
			name: "dynamic not condition",
			args: args{obj: noConditionObj, statusType: objectStatusDynamic},
			want: false,
		},
		{
			name: "dynamic ready outdated",
			args: args{obj: readyObjOutdated, statusType: objectStatusDynamic},
			want: false,
		},
		{
			name: "dynamic ready without per condition gen",
			args: args{obj: oldObj, statusType: objectStatusDynamic},
			want: true,
		},
		{
			name: "dynamic outdated ready status without per condition gen",
			args: args{obj: oldObjOutdated, statusType: objectStatusDynamic},
			want: false,
		},
		{
			name: "static empty status",
			args: args{obj: staticObj, statusType: objectStatusStatic},
			want: true,
		},
		{
			name: "static no status",
			args: args{obj: noStatusObj, statusType: objectStatusStatic},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isObjectReady(tt.args.obj, tt.args.statusType)
			if (err != nil) != tt.wantErr {
				t.Errorf("isObjectReady() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("isObjectReady() = %v, want %v", got, tt.want)
			}
		})
	}
}
