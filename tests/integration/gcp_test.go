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
	"log"
	"os"
	"strings"

	"cloud.google.com/go/pubsub"
	tfjson "github.com/hashicorp/terraform-json"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/test-infra/tftestenv"
)

const (
	gcpSourceRepoKnownHosts = "[source.developers.google.com]:2022 ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBB5Iy4/cq/gt/fPqe3uyMy4jwv1Alc94yVPxmnwNhBzJqEV5gRPiRk5u4/JJMbbu9QUVAguBABxL7sBZa5PH/xY="
)

// createKubeConfigGKE constructs kubeconfig for a GKE cluster from the
// terraform state output at the given kubeconfig path.
func createKubeConfigGKE(ctx context.Context, state map[string]*tfjson.StateOutput, kcPath string) error {
	kubeconfigYaml, ok := state["gke_kubeconfig"].Value.(string)
	if !ok || kubeconfigYaml == "" {
		return fmt.Errorf("failed to obtain kubeconfig from tf output")
	}
	return tftestenv.CreateKubeconfigGKE(ctx, kubeconfigYaml, kcPath)
}

// registryLoginGCR logs into the Artifact registries using the gcloud
// and returns a list of test repositories.
func registryLoginGCR(ctx context.Context, output map[string]*tfjson.StateOutput) (string, error) {
	project := output["gcp_project_id"].Value.(string)
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

	privateKeyFile, ok := os.LookupEnv(envVarGitRepoSSHPath)
	if !ok {
		return nil, fmt.Errorf("%s env variable isn't set", envVarGitRepoSSHPath)
	}
	privateKeyData, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("error getting gcp source repositories private key, '%s': %w", privateKeyFile, err)
	}

	pubKeyFile, ok := os.LookupEnv(envVarGitRepoSSHPubPath)
	if !ok {
		return nil, fmt.Errorf("%s env variable isn't set", envVarGitRepoSSHPubPath)
	}
	pubKeyData, err := os.ReadFile(pubKeyFile)
	if err != nil {
		return nil, fmt.Errorf("error getting ssh pubkey '%s', %w", pubKeyFile, err)
	}

	c := make(chan []byte, 10)
	projectID := outputs["gcp_project_id"].Value.(string)
	topicID := outputs["pubsub_topic"].Value.(string)

	fn, err := setupPubSubReceiver(ctx, c, projectID, topicID)
	if err != nil {
		return nil, err
	}

	var notificationCfg = notificationConfig{
		providerType:     "googlepubsub",
		providerChannel:  topicID,
		notificationChan: c,
		closeChan:        fn,
		secret: map[string]string{
			"address": projectID,
		},
	}

	config := &testConfig{
		defaultGitTransport: git.SSH,
		gitUsername:         "git",
		gitPrivateKey:       string(privateKeyData),
		gitPublicKey:        string(pubKeyData),
		knownHosts:          gcpSourceRepoKnownHosts,
		fleetInfraRepository: gitUrl{
			ssh: outputs["fleet_infra_repository"].Value.(string),
		},
		applicationRepository: gitUrl{
			ssh: outputs["application_repository"].Value.(string),
		},
		notificationCfg: notificationCfg,
		sopsArgs:        fmt.Sprintf("--gcp-kms %s", sharedSopsId),
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

func setupPubSubReceiver(ctx context.Context, c chan []byte, projectID string, topicID string) (func(), error) {
	newCtx, cancel := context.WithCancel(ctx)
	pubsubClient, err := pubsub.NewClient(newCtx, projectID)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("error creating pubsub client: %s", err)
	}

	sub := pubsubClient.Subscription(topicID)
	go func() {
		err = sub.Receive(ctx, func(ctx context.Context, message *pubsub.Message) {
			c <- message.Data
			message.Ack()
		})
		if err != nil && status.Code(err) != codes.Canceled {
			log.Printf("error receiving message in subscription: %s\n", err)
			return
		}
	}()

	return func() {
		cancel()
		pubsubClient.Close()
	}, nil
}
