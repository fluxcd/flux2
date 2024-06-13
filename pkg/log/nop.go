/*
Copyright 2022 The Flux authors

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

package log

type NopLogger struct{}

func (NopLogger) Actionf(format string, a ...interface{}) {}

func (NopLogger) Generatef(format string, a ...interface{}) {}

func (NopLogger) Waitingf(format string, a ...interface{}) {}

func (NopLogger) Successf(format string, a ...interface{}) {}

func (NopLogger) Warningf(format string, a ...interface{}) {}

func (NopLogger) Failuref(format string, a ...interface{}) {}
