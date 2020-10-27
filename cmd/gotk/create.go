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
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create or update sources and resources",
	Long:  "The create sub-commands generate sources and resources.",
}

var (
	interval time.Duration
	export   bool
	labels   []string
)

func init() {
	createCmd.PersistentFlags().DurationVarP(&interval, "interval", "", time.Minute, "source sync interval")
	createCmd.PersistentFlags().BoolVar(&export, "export", false, "export in YAML format to stdout")
	createCmd.PersistentFlags().StringSliceVar(&labels, "label", nil,
		"set labels on the resource (can specify multiple labels with commas: label1=value1,label2=value2)")
	rootCmd.AddCommand(createCmd)
}

func parseLabels() (map[string]string, error) {
	result := make(map[string]string)
	for _, label := range labels {
		// validate key value pair
		parts := strings.Split(label, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid label format '%s', must be key=value", label)
		}

		// validate label name
		if errors := validation.IsQualifiedName(parts[0]); len(errors) > 0 {
			return nil, fmt.Errorf("invalid label '%s': %v", parts[0], errors)
		}

		// validate label value
		if errors := validation.IsValidLabelValue(parts[1]); len(errors) > 0 {
			return nil, fmt.Errorf("invalid label value '%s': %v", parts[1], errors)
		}

		result[parts[0]] = parts[1]
	}

	return result, nil
}
