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

import (
	"fmt"
	"io"

	"sigs.k8s.io/yaml"
)

// YamlPrinter is a printer that prints Flux cmd outputs as yaml.
func YamlPrinter(header []string) PrinterFunc {
	return func(w io.Writer, args ...interface{}) error {
		// args is 2d array of string
		// headers is 1d array of string
		var rows [][]string
		for _, arg := range args {
			switch arg := arg.(type) {
			case []interface{}:
				for _, v := range arg {
					s, ok := v.([][]string)
					if !ok {
						return fmt.Errorf("unsupported type %T", v)
					}
					rows = append(rows, s...)
				}
			default:
				return fmt.Errorf("unsupported type %T", arg)
			}
		}

		var entries []map[string]string
		for _, row := range rows {
			entry := make(map[string]string)
			for j, h := range header {
				entry[h] = row[j]
			}
			entries = append(entries, entry)
		}

		if len(entries) > 0 {
			y, err := yaml.Marshal(entries)
			if err != nil {
				return fmt.Errorf("error marshalling yaml %w", err)
			}
			fmt.Fprint(w, string(y))
		}

		return nil
	}
}
