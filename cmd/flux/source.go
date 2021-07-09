/*
Copyright 2021 The Flux authors

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
	"sigs.k8s.io/controller-runtime/pkg/client"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

// These are general-purpose adapters for attaching methods to, for
// the various commands. The *List adapters implement len(), since
// it's used in at least a couple of commands.

// sourcev1.Bucket

var bucketType = apiType{
	kind:      sourcev1.BucketKind,
	humanKind: "source bucket",
}

type bucketAdapter struct {
	*sourcev1.Bucket
}

func (a bucketAdapter) asClientObject() client.Object {
	return a.Bucket
}

func (a bucketAdapter) deepCopyClientObject() client.Object {
	return a.Bucket.DeepCopy()
}

// sourcev1.BucketList

type bucketListAdapter struct {
	*sourcev1.BucketList
}

func (a bucketListAdapter) asClientList() client.ObjectList {
	return a.BucketList
}

func (a bucketListAdapter) len() int {
	return len(a.BucketList.Items)
}

// sourcev1.HelmChart

var helmChartType = apiType{
	kind:      sourcev1.HelmChartKind,
	humanKind: "source chart",
}

type helmChartAdapter struct {
	*sourcev1.HelmChart
}

func (a helmChartAdapter) asClientObject() client.Object {
	return a.HelmChart
}

func (a helmChartAdapter) deepCopyClientObject() client.Object {
	return a.HelmChart.DeepCopy()
}

// sourcev1.HelmChartList

type helmChartListAdapter struct {
	*sourcev1.HelmChartList
}

func (a helmChartListAdapter) asClientList() client.ObjectList {
	return a.HelmChartList
}

func (a helmChartListAdapter) len() int {
	return len(a.HelmChartList.Items)
}

// sourcev1.GitRepository

var gitRepositoryType = apiType{
	kind:      sourcev1.GitRepositoryKind,
	humanKind: "source git",
}

type gitRepositoryAdapter struct {
	*sourcev1.GitRepository
}

func (a gitRepositoryAdapter) asClientObject() client.Object {
	return a.GitRepository
}

func (a gitRepositoryAdapter) deepCopyClientObject() client.Object {
	return a.GitRepository.DeepCopy()
}

// sourcev1.GitRepositoryList

type gitRepositoryListAdapter struct {
	*sourcev1.GitRepositoryList
}

func (a gitRepositoryListAdapter) asClientList() client.ObjectList {
	return a.GitRepositoryList
}

func (a gitRepositoryListAdapter) len() int {
	return len(a.GitRepositoryList.Items)
}

// sourcev1.HelmRepository

var helmRepositoryType = apiType{
	kind:      sourcev1.HelmRepositoryKind,
	humanKind: "source helm",
}

type helmRepositoryAdapter struct {
	*sourcev1.HelmRepository
}

func (a helmRepositoryAdapter) asClientObject() client.Object {
	return a.HelmRepository
}

func (a helmRepositoryAdapter) deepCopyClientObject() client.Object {
	return a.HelmRepository.DeepCopy()
}

// sourcev1.HelmRepositoryList

type helmRepositoryListAdapter struct {
	*sourcev1.HelmRepositoryList
}

func (a helmRepositoryListAdapter) asClientList() client.ObjectList {
	return a.HelmRepositoryList
}

func (a helmRepositoryListAdapter) len() int {
	return len(a.HelmRepositoryList.Items)
}
