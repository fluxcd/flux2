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

package flags

import (
	"fmt"
	"strings"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var supportedLogLevels = []string{"debug", "info", "error"}

type LogLevel string

func (l *LogLevel) String() string {
	return string(*l)
}

func (l *LogLevel) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("no log level given, must be one of: %s",
			strings.Join(supportedLogLevels, ", "))
	}
	if !utils.ContainsItemString(supportedLogLevels, str) {
		return fmt.Errorf("unsupported log level '%s', must be one of: %s",
			str, strings.Join(supportedLogLevels, ", "))

	}
	*l = LogLevel(str)
	return nil
}

func (l *LogLevel) Type() string {
	return "logLevel"
}

func (l *LogLevel) Description() string {
	return fmt.Sprintf("log level, available options are: (%s)", strings.Join(supportedLogLevels, ", "))
}
