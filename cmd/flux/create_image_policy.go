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
	"regexp/syntax"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fluxcd/pkg/apis/meta"

	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
)

var createImagePolicyCmd = &cobra.Command{
	Use:   "policy [name]",
	Short: "Create or update an ImagePolicy object",
	Long: withPreviewNote(`The create image policy command generates an ImagePolicy resource.
An ImagePolicy object calculates a "latest image" given an image
repository and a policy, e.g., semver.

The image that sorts highest according to the policy is recorded in
the status of the object.`),
	Example: `  # Create an ImagePolicy to select the latest stable release
  flux create image policy podinfo \
    --image-ref=podinfo \
    --select-semver=">=1.0.0"

  # Create an ImagePolicy to select the latest main branch build tagged as "${GIT_BRANCH}-${GIT_SHA:0:7}-$(date +%s)"
  flux create image policy podinfo \
    --image-ref=podinfo \
    --select-numeric=asc \
	--filter-regex='^main-[a-f0-9]+-(?P<ts>[0-9]+)' \
	--filter-extract='$ts'`,
	RunE: createImagePolicyRun}

type imagePolicyFlags struct {
	imageRef      string
	semver        string
	alpha         string
	numeric       string
	filterRegex   string
	filterExtract string
}

var imagePolicyArgs = imagePolicyFlags{}

func init() {
	flags := createImagePolicyCmd.Flags()
	flags.StringVar(&imagePolicyArgs.imageRef, "image-ref", "", "the name of an image repository object")
	flags.StringVar(&imagePolicyArgs.semver, "select-semver", "", "a semver range to apply to tags; e.g., '1.x'")
	flags.StringVar(&imagePolicyArgs.alpha, "select-alpha", "", "use alphabetical sorting to select image; either \"asc\" meaning select the last, or \"desc\" meaning select the first")
	flags.StringVar(&imagePolicyArgs.numeric, "select-numeric", "", "use numeric sorting to select image; either \"asc\" meaning select the last, or \"desc\" meaning select the first")
	flags.StringVar(&imagePolicyArgs.filterRegex, "filter-regex", "", "regular expression pattern used to filter the image tags")
	flags.StringVar(&imagePolicyArgs.filterExtract, "filter-extract", "", "replacement pattern (using capture groups from --filter-regex) to use for sorting")

	createImageCmd.AddCommand(createImagePolicyCmd)
}

// getObservedGeneration is implemented here, since it's not
// (presently) needed elsewhere.
func (obj imagePolicyAdapter) getObservedGeneration() int64 {
	return obj.ImagePolicy.Status.ObservedGeneration
}

func createImagePolicyRun(cmd *cobra.Command, args []string) error {
	objectName := args[0]

	if imagePolicyArgs.imageRef == "" {
		return fmt.Errorf("the name of an ImageRepository in the namespace is required (--image-ref)")
	}

	labels, err := parseLabels()
	if err != nil {
		return err
	}

	var policy = imagev1.ImagePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectName,
			Namespace: *kubeconfigArgs.Namespace,
			Labels:    labels,
		},
		Spec: imagev1.ImagePolicySpec{
			ImageRepositoryRef: meta.NamespacedObjectReference{
				Name: imagePolicyArgs.imageRef,
			},
		},
	}

	switch {
	case imagePolicyArgs.semver != "" && imagePolicyArgs.alpha != "":
	case imagePolicyArgs.semver != "" && imagePolicyArgs.numeric != "":
	case imagePolicyArgs.alpha != "" && imagePolicyArgs.numeric != "":
		return fmt.Errorf("only one of --select-semver, --select-alpha or --select-numeric can be specified")
	case imagePolicyArgs.semver != "":
		policy.Spec.Policy.SemVer = &imagev1.SemVerPolicy{
			Range: imagePolicyArgs.semver,
		}
	case imagePolicyArgs.alpha != "":
		if imagePolicyArgs.alpha != "desc" && imagePolicyArgs.alpha != "asc" {
			return fmt.Errorf("--select-alpha must be one of [\"asc\", \"desc\"]")
		}
		policy.Spec.Policy.Alphabetical = &imagev1.AlphabeticalPolicy{
			Order: imagePolicyArgs.alpha,
		}
	case imagePolicyArgs.numeric != "":
		if imagePolicyArgs.numeric != "desc" && imagePolicyArgs.numeric != "asc" {
			return fmt.Errorf("--select-numeric must be one of [\"asc\", \"desc\"]")
		}
		policy.Spec.Policy.Numerical = &imagev1.NumericalPolicy{
			Order: imagePolicyArgs.numeric,
		}
	default:
		return fmt.Errorf("a policy must be provided with either --select-semver or --select-alpha")
	}

	if imagePolicyArgs.filterRegex != "" {
		exp, err := syntax.Parse(imagePolicyArgs.filterRegex, syntax.Perl)
		if err != nil {
			return fmt.Errorf("--filter-regex is an invalid regex pattern")
		}
		policy.Spec.FilterTags = &imagev1.TagFilter{
			Pattern: imagePolicyArgs.filterRegex,
		}

		if imagePolicyArgs.filterExtract != "" {
			if err := validateExtractStr(imagePolicyArgs.filterExtract, exp.CapNames()); err != nil {
				return err
			}
			policy.Spec.FilterTags.Extract = imagePolicyArgs.filterExtract
		}
	} else if imagePolicyArgs.filterExtract != "" {
		return fmt.Errorf("cannot specify --filter-extract without specifying --filter-regex")
	}

	if createArgs.export {
		return printExport(exportImagePolicy(&policy))
	}

	var existing imagev1.ImagePolicy
	copyName(&existing, &policy)
	err = imagePolicyType.upsertAndWait(imagePolicyAdapter{&existing}, func() error {
		existing.Spec = policy.Spec
		existing.SetLabels(policy.Labels)
		return nil
	})
	return err
}

// Performs a dry-run of the extract function in Regexp to validate the template
func validateExtractStr(template string, capNames []string) error {
	for len(template) > 0 {
		i := strings.Index(template, "$")
		if i < 0 {
			return nil
		}
		template = template[i:]
		if len(template) > 1 && template[1] == '$' {
			template = template[2:]
			continue
		}
		name, num, rest, ok := extract(template)
		if !ok {
			// Malformed extract string, assume user didn't want this
			return fmt.Errorf("--filter-extract is malformed")
		}
		template = rest
		if num >= 0 {
			// we won't worry about numbers as we can't validate these
			continue
		} else {
			found := false
			for _, capName := range capNames {
				if name == capName {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("capture group $%s used in --filter-extract not found in --filter-regex", name)
			}
		}

	}
	return nil
}

// extract method from the regexp package
// returns the name or number of the value prepended by $
func extract(str string) (name string, num int, rest string, ok bool) {
	if len(str) < 2 || str[0] != '$' {
		return
	}
	brace := false
	if str[1] == '{' {
		brace = true
		str = str[2:]
	} else {
		str = str[1:]
	}
	i := 0
	for i < len(str) {
		rune, size := utf8.DecodeRuneInString(str[i:])
		if !unicode.IsLetter(rune) && !unicode.IsDigit(rune) && rune != '_' {
			break
		}
		i += size
	}
	if i == 0 {
		// empty name is not okay
		return
	}
	name = str[:i]
	if brace {
		if i >= len(str) || str[i] != '}' {
			// missing closing brace
			return
		}
		i++
	}

	// Parse number.
	num = 0
	for i := 0; i < len(name); i++ {
		if name[i] < '0' || '9' < name[i] || num >= 1e8 {
			num = -1
			break
		}
		num = num*10 + int(name[i]) - '0'
	}
	// Disallow leading zeros.
	if name[0] == '0' && len(name) > 1 {
		num = -1
	}

	rest = str[i:]
	ok = true
	return
}
