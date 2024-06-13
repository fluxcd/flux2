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
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

var supportedSourceBucketProviders = []string{
	sourcev1.GenericBucketProvider,
	sourcev1.AmazonBucketProvider,
	sourcev1.AzureBucketProvider,
	sourcev1.GoogleBucketProvider,
}

type SourceBucketProvider string

func (p *SourceBucketProvider) String() string {
	return string(*p)
}

func (p *SourceBucketProvider) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("no source bucket provider given, please specify %s",
			p.Description())
	}
	if !utils.ContainsItemString(supportedSourceBucketProviders, str) {
		return fmt.Errorf("source bucket provider '%s' is not supported, must be one of: %v",
			str, strings.Join(supportedSourceBucketProviders, ", "))
	}
	*p = SourceBucketProvider(str)
	return nil
}

func (p *SourceBucketProvider) Type() string {
	return "sourceBucketProvider"
}

func (p *SourceBucketProvider) Description() string {
	return fmt.Sprintf(
		"the S3 compatible storage provider name, available options are: (%s)",
		strings.Join(supportedSourceBucketProviders, ", "),
	)
}
