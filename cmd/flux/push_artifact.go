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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/pkg/oci"
	"github.com/fluxcd/pkg/oci/auth/login"
	"github.com/fluxcd/pkg/oci/client"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/v2/internal/flags"
)

var pushArtifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Push artifact",
	Long: withPreviewNote(`The push artifact command creates a tarball from the given directory or the single file and uploads the artifact to an OCI repository.
The command can read the credentials from '~/.docker/config.json' but they can also be passed with --creds. It can also login to a supported provider with the --provider flag.`),
	Example: `  # Push manifests to GHCR using the short Git SHA as the OCI artifact tag
  echo $GITHUB_PAT | docker login ghcr.io --username flux --password-stdin
  flux push artifact oci://ghcr.io/org/config/app:$(git rev-parse --short HEAD) \
	--path="./path/to/local/manifests" \
	--source="$(git config --get remote.origin.url)" \
	--revision="$(git branch --show-current)@sha1:$(git rev-parse HEAD)"

  # Push and sign artifact with cosign
  digest_url = $(flux push artifact \
	oci://ghcr.io/org/config/app:$(git rev-parse --short HEAD) \
	--source="$(git config --get remote.origin.url)" \
	--revision="$(git branch --show-current)@sha1:$(git rev-parse HEAD)" \
	--path="./path/to/local/manifest.yaml" \
	--output json | \
	jq -r '. | .repository + "@" + .digest')
  cosign sign $digest_url

  # Push manifests passed into stdin to GHCR and set custom OCI annotations
  kustomize build . | flux push artifact oci://ghcr.io/org/config/app:$(git rev-parse --short HEAD) -f - \ 
    --source="$(git config --get remote.origin.url)" \
    --revision="$(git branch --show-current)@sha1:$(git rev-parse HEAD)" \
    --annotations='org.opencontainers.image.licenses=Apache-2.0' \
    --annotations='org.opencontainers.image.documentation=https://app.org/docs' \
    --annotations='org.opencontainers.image.description=Production config.'

  # Push single manifest file to GHCR using the short Git SHA as the OCI artifact tag
  echo $GITHUB_PAT | docker login ghcr.io --username flux --password-stdin
  flux push artifact oci://ghcr.io/org/config/app:$(git rev-parse --short HEAD) \
	--path="./path/to/local/manifest.yaml" \
	--source="$(git config --get remote.origin.url)" \
	--revision="$(git branch --show-current)@sha1:$(git rev-parse HEAD)"

  # Push manifests to Docker Hub using the Git tag as the OCI artifact tag
  echo $DOCKER_PAT | docker login --username flux --password-stdin
  flux push artifact oci://docker.io/org/app-config:$(git tag --points-at HEAD) \
	--path="./path/to/local/manifests" \
	--source="$(git config --get remote.origin.url)" \
	--revision="$(git tag --points-at HEAD)@sha1:$(git rev-parse HEAD)"

  # Login directly to the registry provider
  # You might need to export the following variable if you use local config files for AWS:
  # export AWS_SDK_LOAD_CONFIG=1
  flux push artifact oci://<account>.dkr.ecr.<region>.amazonaws.com/app-config:$(git tag --points-at HEAD) \
	--path="./path/to/local/manifests" \
	--source="$(git config --get remote.origin.url)" \
	--revision="$(git tag --points-at HEAD)@sha1:$(git rev-parse HEAD)" \
	--provider aws

  # Login by passing credentials directly
  flux push artifact oci://docker.io/org/app-config:$(git tag --points-at HEAD) \
	--path="./path/to/local/manifests" \
	--source="$(git config --get remote.origin.url)" \
	--revision="$(git tag --points-at HEAD)@sha1:$(git rev-parse HEAD)" \
	--creds flux:$DOCKER_PAT
`,
	RunE: pushArtifactCmdRun,
}

type pushArtifactFlags struct {
	path        string
	source      string
	revision    string
	creds       string
	provider    flags.SourceOCIProvider
	ignorePaths []string
	annotations []string
	output      string
	debug       bool
}

var pushArtifactArgs = newPushArtifactFlags()

func newPushArtifactFlags() pushArtifactFlags {
	return pushArtifactFlags{
		provider: flags.SourceOCIProvider(sourcev1.GenericOCIProvider),
	}
}

func init() {
	pushArtifactCmd.Flags().StringVarP(&pushArtifactArgs.path, "path", "f", "", "path to the directory where the Kubernetes manifests are located")
	pushArtifactCmd.Flags().StringVar(&pushArtifactArgs.source, "source", "", "the source address, e.g. the Git URL")
	pushArtifactCmd.Flags().StringVar(&pushArtifactArgs.revision, "revision", "", "the source revision in the format '<branch|tag>@sha1:<commit-sha>'")
	pushArtifactCmd.Flags().StringVar(&pushArtifactArgs.creds, "creds", "", "credentials for OCI registry in the format <username>[:<password>] if --provider is generic")
	pushArtifactCmd.Flags().Var(&pushArtifactArgs.provider, "provider", pushArtifactArgs.provider.Description())
	pushArtifactCmd.Flags().StringSliceVar(&pushArtifactArgs.ignorePaths, "ignore-paths", excludeOCI, "set paths to ignore in .gitignore format")
	pushArtifactCmd.Flags().StringArrayVarP(&pushArtifactArgs.annotations, "annotations", "a", nil, "Set custom OCI annotations in the format '<key>=<value>'")
	pushArtifactCmd.Flags().StringVarP(&pushArtifactArgs.output, "output", "o", "",
		"the format in which the artifact digest should be printed, can be 'json' or 'yaml'")
	pushArtifactCmd.Flags().BoolVarP(&pushArtifactArgs.debug, "debug", "", false, "display logs from underlying library")

	pushCmd.AddCommand(pushArtifactCmd)
}

func pushArtifactCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("artifact URL is required")
	}
	ociURL := args[0]

	if pushArtifactArgs.source == "" {
		return fmt.Errorf("--source is required")
	}

	if pushArtifactArgs.revision == "" {
		return fmt.Errorf("--revision is required")
	}

	if pushArtifactArgs.path == "" {
		return fmt.Errorf("invalid path %q", pushArtifactArgs.path)
	}

	url, err := client.ParseArtifactURL(ociURL)
	if err != nil {
		return err
	}

	ref, err := name.ParseReference(url)
	if err != nil {
		return err
	}

	path := pushArtifactArgs.path
	if pushArtifactArgs.path == "-" {
		path, err = saveReaderToFile(os.Stdin)
		if err != nil {
			return err
		}

		defer os.Remove(path)
	}

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("invalid path '%s', must point to an existing directory or file: %w", path, err)
	}

	annotations := map[string]string{}
	for _, annotation := range pushArtifactArgs.annotations {
		kv := strings.Split(annotation, "=")
		if len(kv) != 2 {
			return fmt.Errorf("invalid annotation %s, must be in the format key=value", annotation)
		}
		annotations[kv[0]] = kv[1]
	}

	if pushArtifactArgs.debug {
		// direct logs from crane library to stderr
		// this can be useful to figure out things happening underneath e.g when the library is retrying a request
		logs.Warn.SetOutput(os.Stderr)
	}

	meta := client.Metadata{
		Source:      pushArtifactArgs.source,
		Revision:    pushArtifactArgs.revision,
		Annotations: annotations,
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	var auth authn.Authenticator
	opts := client.DefaultOptions()
	if pushArtifactArgs.provider.String() == sourcev1.GenericOCIProvider && pushArtifactArgs.creds != "" {
		logger.Actionf("logging in to registry with credentials")
		auth, err = client.GetAuthFromCredentials(pushArtifactArgs.creds)
		if err != nil {
			return fmt.Errorf("could not login with credentials: %w", err)
		}
		opts = append(opts, crane.WithAuth(auth))
	}

	if pushArtifactArgs.provider.String() != sourcev1.GenericOCIProvider {
		logger.Actionf("logging in to registry with provider credentials")
		ociProvider, err := pushArtifactArgs.provider.ToOCIProvider()
		if err != nil {
			return fmt.Errorf("provider not supported: %w", err)
		}

		auth, err = login.NewManager().Login(ctx, url, ref, getProviderLoginOption(ociProvider))
		if err != nil {
			return fmt.Errorf("error during login with provider: %w", err)
		}
		opts = append(opts, crane.WithAuth(auth))
	}

	if rootArgs.timeout != 0 {
		backoff := remote.Backoff{
			Duration: 1.0 * time.Second,
			Factor:   3,
			Jitter:   0.1,
			// timeout happens when the cap is exceeded or number of steps is reached
			// 10 steps is big enough that most reasonable cap(under 30min) will be exceeded before
			// the number of steps are completed.
			Steps: 10,
			Cap:   rootArgs.timeout,
		}

		if auth == nil {
			auth, err = authn.DefaultKeychain.Resolve(ref.Context())
			if err != nil {
				return err
			}
		}
		transportOpts, err := client.WithRetryTransport(ctx, ref, auth, backoff, []string{ref.Context().Scope(transport.PushScope)})
		if err != nil {
			return fmt.Errorf("error setting up transport: %w", err)
		}
		opts = append(opts, transportOpts, client.WithRetryBackOff(backoff))
	}

	if pushArtifactArgs.output == "" {
		logger.Actionf("pushing artifact to %s", url)
	}

	ociClient := client.NewClient(opts)
	digestURL, err := ociClient.Push(ctx, url, path,
		client.WithPushMetadata(meta),
		client.WithPushIgnorePaths(pushArtifactArgs.ignorePaths...),
	)
	if err != nil {
		return fmt.Errorf("pushing artifact failed: %w", err)
	}

	digest, err := name.NewDigest(digestURL)
	if err != nil {
		return fmt.Errorf("artifact digest parsing failed: %w", err)
	}

	tag, err := name.NewTag(url)
	if err != nil {
		return fmt.Errorf("artifact tag parsing failed: %w", err)
	}

	info := struct {
		URL        string `json:"url"`
		Repository string `json:"repository"`
		Tag        string `json:"tag"`
		Digest     string `json:"digest"`
	}{
		URL:        fmt.Sprintf("oci://%s", digestURL),
		Repository: digest.Repository.Name(),
		Tag:        tag.TagStr(),
		Digest:     digest.DigestStr(),
	}

	switch pushArtifactArgs.output {
	case "json":
		marshalled, err := json.MarshalIndent(&info, "", "  ")
		if err != nil {
			return fmt.Errorf("artifact digest JSON conversion failed: %w", err)
		}
		marshalled = append(marshalled, "\n"...)
		rootCmd.Print(string(marshalled))
	case "yaml":
		marshalled, err := yaml.Marshal(&info)
		if err != nil {
			return fmt.Errorf("artifact digest YAML conversion failed: %w", err)
		}
		rootCmd.Print(string(marshalled))
	default:
		logger.Successf("artifact successfully pushed to %s", digestURL)
	}

	return nil
}

func getProviderLoginOption(provider oci.Provider) login.ProviderOptions {
	var opts login.ProviderOptions
	switch provider {
	case oci.ProviderAzure:
		opts.AzureAutoLogin = true
	case oci.ProviderAWS:
		opts.AwsAutoLogin = true
	case oci.ProviderGCP:
		opts.GcpAutoLogin = true
	}
	return opts
}
