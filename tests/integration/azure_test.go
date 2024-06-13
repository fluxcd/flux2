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
	"os"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/test-infra/tftestenv"
	tfjson "github.com/hashicorp/terraform-json"
)

const (
	azureDevOpsKnownHosts = "ssh.dev.azure.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7Hr1oTWqNqOlzGJOfGJ4NakVyIzf1rXYd4d7wo6jBlkLvCA4odBlL0mDUyZ0/QUfTTqeu+tm22gOsv+VrVTMk6vwRU75gY/y9ut5Mb3bR5BV58dKXyq9A9UeB5Cakehn5Zgm6x1mKoVyf+FFn26iYqXJRgzIZZcZ5V6hrE0Qg39kZm4az48o0AUbf6Sp4SLdvnuMa2sVNwHBboS7EJkm57XQPVU3/QpyNLHbWDdzwtrlS+ez30S3AdYhLKEOxAG8weOnyrtLJAUen9mTkol8oII1edf7mWWbWVf0nBmly21+nZcmCTISQBtdcyPaEno7fFQMDD26/s0lfKob4Kw8H"
)

// createKubeConfigAKS constructs kubeconfig for an AKS cluster from the
// terraform state output at the given kubeconfig path.
func createKubeConfigAKS(ctx context.Context, state map[string]*tfjson.StateOutput, kcPath string) error {
	kubeconfigYaml, ok := state["aks_kubeconfig"].Value.(string)
	if !ok || kubeconfigYaml == "" {
		return fmt.Errorf("failed to obtain kubeconfig from tf output")
	}
	return tftestenv.CreateKubeconfigAKS(ctx, kubeconfigYaml, kcPath)
}

func getTestConfigAKS(ctx context.Context, outputs map[string]*tfjson.StateOutput) (*testConfig, error) {
	fleetInfraRepository := outputs["fleet_infra_repository"].Value.(map[string]interface{})
	applicationRepository := outputs["application_repository"].Value.(map[string]interface{})

	eventHubSas := outputs["event_hub_sas"].Value.(string)
	sharedSopsId := outputs["sops_id"].Value.(string)

	kustomizeYaml := `
resources:
  - gotk-components.yaml
  - gotk-sync.yaml
patchesStrategicMerge:
  - |-
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: kustomize-controller
      namespace: flux-system
    spec:
      template:
        spec:
          containers:
          - name: manager
            env:
            - name: AZURE_AUTH_METHOD
              value: msi
`

	privateKeyFile, ok := os.LookupEnv(envVarGitRepoSSHPath)
	if !ok {
		return nil, fmt.Errorf("%s env variable isn't set", envVarGitRepoSSHPath)
	}
	privateKeyData, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("error getting azure devops private key, '%s': %w", privateKeyFile, err)
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
	closefn, err := setupEventHubHandler(ctx, c, eventHubSas)

	var notificationCfg = notificationConfig{
		notificationChan: c,
		providerType:     "azureeventhub",
		closeChan:        closefn,
		secret: map[string]string{
			"address": eventHubSas,
		},
	}

	config := &testConfig{
		defaultGitTransport: git.HTTP,
		gitUsername:         git.DefaultPublicKeyAuthUser,
		gitPat:              outputs["azure_devops_access_token"].Value.(string),
		gitPrivateKey:       string(privateKeyData),
		gitPublicKey:        string(pubKeyData),
		knownHosts:          azureDevOpsKnownHosts,
		fleetInfraRepository: gitUrl{
			http: fleetInfraRepository["http"].(string),
			ssh:  fleetInfraRepository["ssh"].(string),
		},
		applicationRepository: gitUrl{
			http: applicationRepository["http"].(string),
			ssh:  applicationRepository["ssh"].(string),
		},
		notificationCfg: notificationCfg,
		sopsArgs:        fmt.Sprintf("--azure-kv %s", sharedSopsId),
		sopsSecretData: map[string]string{
			"sops.azure-kv": fmt.Sprintf(`clientId: %s`, outputs["aks_client_id"].Value.(string)),
		},
		kustomizationYaml: kustomizeYaml,
	}

	opts, err := authOpts(config.fleetInfraRepository.http, map[string][]byte{
		"password": []byte(config.gitPat),
		"username": []byte("git"),
	})
	if err != nil {
		return nil, err
	}
	config.defaultAuthOpts = opts

	return config, nil
}

// registryLoginACR logs into the Azure Container Registries using the
// provider's CLI tools and returns the test repositories.
func registryLoginACR(ctx context.Context, output map[string]*tfjson.StateOutput) (string, error) {
	// NOTE: ACR registry accept dynamic repository creation by just pushing a
	// new image with a new repository name.
	registryURL := output["acr_url"].Value.(string)
	if err := tftestenv.RegistryLoginACR(ctx, registryURL); err != nil {
		return "", err
	}

	return registryURL, nil
}

func setupEventHubHandler(ctx context.Context, c chan []byte, eventHubSas string) (func(), error) {
	hub, err := eventhub.NewHubFromConnectionString(eventHubSas)
	if err != nil {
		return nil, err
	}

	handler := func(ctx context.Context, event *eventhub.Event) error {
		c <- event.Data
		return nil
	}
	runtimeInfo, err := hub.GetRuntimeInformation(ctx)
	if err != nil {
		return nil, err
	}
	listenerHandler, err := hub.Receive(ctx, runtimeInfo.PartitionIDs[0], handler, eventhub.ReceiveWithLatestOffset())
	if err != nil {
		return nil, err
	}

	closefn := func() {
		listenerHandler.Close(ctx)
		hub.Close(ctx)
	}

	return closefn, nil
}
