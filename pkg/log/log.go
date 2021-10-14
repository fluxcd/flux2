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

package log

type Logger interface {
	// Actionf logs a formatted action message.
	Actionf(format string, a ...interface{})
	// Generatef logs a formatted generate message.
	Generatef(format string, a ...interface{})
	// Waitingf logs a formatted waiting message.
	Waitingf(format string, a ...interface{})
	// Successf logs a formatted success message.
	Successf(format string, a ...interface{})
	// Warningf logs a formatted warning message.
	Warningf(format string, a ...interface{})
	// Failuref logs a formatted failure message.
	Failuref(format string, a ...interface{})
}
