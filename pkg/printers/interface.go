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

package printers

import "io"

// Printer is an interface for printing Flux cmd	outputs.
type Printer interface {
	// Print prints the given args to the given writer.
	Print(io.Writer, ...interface{}) error
}

// PrinterFunc is a function that can print args to a writer.
type PrinterFunc func(w io.Writer, args ...interface{}) error

// Print implements Printer
func (fn PrinterFunc) Print(w io.Writer, args ...interface{}) error {
	return fn(w, args)
}
