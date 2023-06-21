/*
Copyright 2023 The Flux authors

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

package integration

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/test-infra/tftestenv"
	tfjson "github.com/hashicorp/terraform-json"
)

const (
	gkeDevOpsKnownHosts = "[source.developers.google.com]:2022 ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBB5Iy4/cq/gt/fPqe3uyMy4jwv1Alc94yVPxmnwNhBzJqEV5gRPiRk5u4/JJMbbu9QUVAguBABxL7sBZa5PH/xY="
)

// createKubeConfigGKE constructs kubeconfig for a GKE cluster from the
// terraform state output at the given kubeconfig path.
func createKubeConfigGKE(ctx context.Context, state map[string]*tfjson.StateOutput, kcPath string) error {
	kubeconfigYaml, ok := state["kubeconfig"].Value.(string)
	if !ok || kubeconfigYaml == "" {
		return fmt.Errorf("failed to obtain kubeconfig from tf output")
	}
	return tftestenv.CreateKubeconfigGKE(ctx, kubeconfigYaml, kcPath)
}

// registryLoginGCR logs into the container/artifact registries using the
// provider's CLI tools and returns a list of test repositories.
func registryLoginGCR(ctx context.Context, output map[string]*tfjson.StateOutput) (string, error) {
	// NOTE: ACR registry accept dynamic repository creation by just pushing a
	// new image with a new repository name.
	project := output["gcp_project"].Value.(string)
	region := output["gcp_region"].Value.(string)
	repositoryID := output["artifact_registry_id"].Value.(string)
	artifactRegistryURL, artifactRepoURL := tftestenv.GetGoogleArtifactRegistryAndRepository(project, region, repositoryID)
	if err := tftestenv.RegistryLoginGCR(ctx, artifactRegistryURL); err != nil {
		return "", err
	}

	return artifactRepoURL, nil
}

func getTestConfigGKE(ctx context.Context, outputs map[string]*tfjson.StateOutput) (*testConfig, error) {
	sharedSopsId := outputs["sops_id"].Value.(string)

	privateKeyFile, ok := os.LookupEnv("GCP_SOURCEREPO_SSH")
	if !ok {
		return nil, fmt.Errorf("GCP_SOURCEREPO_SSH env variable isn't set")
	}
	privateKeyData, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("error getting gcp source repositories private key, '%s': %w", privateKeyFile, err)
	}

	pubKeyFile, ok := os.LookupEnv("GCP_SOURCEREPO_SSH_PUB")
	if !ok {
		return nil, fmt.Errorf("GCP_SOURCEREPO_SSH_PUB env variable isn't set")
	}
	pubKeyData, err := os.ReadFile(pubKeyFile)
	if err != nil {
		return nil, fmt.Errorf("error getting ssh pubkey '%s', %w", pubKeyFile, err)
	}

	config := &testConfig{
		defaultGitTransport: git.SSH,
		gitUsername:         "git",
		gitPrivateKey:       string(privateKeyData),
		gitPublicKey:        string(pubKeyData),
		knownHosts:          gkeDevOpsKnownHosts,
		fleetInfraRepository: repoConfig{
			ssh: outputs["fleet_infra_url"].Value.(string),
		},
		applicationRepository: repoConfig{
			ssh: outputs["application_url"].Value.(string),
		},
		sopsArgs: fmt.Sprintf("--gcp-kms %s", sharedSopsId),
	}

	opts, err := authOpts(config.fleetInfraRepository.ssh, map[string][]byte{
		"identity":    []byte(config.gitPrivateKey),
		"known_hosts": []byte(config.knownHosts),
	})
	if err != nil {
		return nil, err
	}

	config.defaultAuthOpts = opts

	// In Azure, the repository is initialized with a default branch through
	// terraform. We have to do it manually here for GCP to prevent errors
	// when trying to clone later. We only need to do it for the application repository
	// since flux bootstrap pushes to the main branch.
	files := make(map[string]io.Reader)
	files["README.md"] = strings.NewReader("# Flux test repo")
	tmpDir, err := os.MkdirTemp("", "*-flux-test")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	client, err := getRepository(context.Background(), tmpDir, config.applicationRepository.ssh, defaultBranch, config.defaultAuthOpts)
	if err != nil {
		return nil, err
	}
	err = commitAndPushAll(context.Background(), client, files, defaultBranch)
	if err != nil {
		return nil, err
	}

	return config, nil
}
