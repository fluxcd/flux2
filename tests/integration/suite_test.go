/*
Copyright 2021 The Flux authors

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

package test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"go.uber.org/multierr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/fluxcd/pkg/git/libgit2/transport"
	"github.com/fluxcd/test-infra/tftestenv"
)

const (
	// aksTerraformPath is the path to the folder containg the
	// terraform files for azure infra
	aksTerraformPath = "./terraform/azure"
)

var (
	cfg            *testConfig
	kubeconfigPath string
	infraOpts      tftestenv.Options
	testRepos      map[string]string
	testEnv        *tftestenv.Environment
)

// testConfig hold different variable that will be needed by the different test functions.
type testConfig struct {
	client client.Client

	// secret data for git repositories
	pat      string
	idRsa    string
	idRsaPub string

	// Generate known host? Use flux cli?
	knownHosts            string
	fleetInfraRepository  repoConfig
	applicationRepository repoConfig

	dockerCred  dockerCred
	eventHubSas string

	// cloud provider dependent argument to pass to the sops cli
	sopsArgs string
	// sops secret data
	sopsSecretData map[string]string
}

// repoConfig contains the http/ssh urls for the created git repositories
// on the various cloud providers.
type repoConfig struct {
	http string
	ssh  string
}

// dockerCred contains credentials for the container repository
type dockerCred struct {
	url      string
	username string
	password string
}

// getTestConfig gets the test configuration that contains different variables for running the tests
type getTestConfig func(ctx context.Context, output map[string]*tfjson.StateOutput) (*testConfig, error)

// getFluxModification returns a map objects(like secrets) that should be created/updated for Flux to run well and a kustomization.yaml
// that can be used to customize Flux
type getFluxKustomization func(ctx context.Context, output map[string]*tfjson.StateOutput) (map[client.Object]controllerutil.MutateFn, string, error)

// registryLoginFunc is used to perform registry login against a provider based
// on the terraform state output values. It returns a map of registry common
// name and test repositories to test against, read from the terraform state
// output.
type registryLoginFunc func(ctx context.Context, output map[string]*tfjson.StateOutput) (map[string]string, error)

// ProviderConfig contains the test configurations for the different cloud providers
type ProviderConfig struct {
	terraformPath        string
	createKubeconfig     tftestenv.CreateKubeconfig
	getTestConfig        getTestConfig
	getFluxKustomization getFluxKustomization
	// registryLogin is used to perform registry login.
	registryLogin registryLoginFunc
}

func TestMain(m *testing.M) {
	infraOpts.Bindflags(flag.CommandLine)
	flag.Parse()

	err := infraOpts.Validate()
	if err != nil {
		log.Fatal(err)
	}

	// TODO(somtochiama): remove when tests have been updated to support GCP and AWS
	if infraOpts.Provider != "azure" {
		log.Fatal("only azure e2e tests are currently supported.")
	}

	exitVal, err := setup(m)
	if err != nil {
		log.Printf("Received an error while running setup: %v", err)
		os.Exit(1)
	}
	os.Exit(exitVal)
}

func setup(m *testing.M) (exitVal int, err error) {
	ctx := context.TODO()

	// get provider specific configuration
	providerCfg, err := getProviderConfig(infraOpts.Provider)

	// Setup Terraform binary and init state
	log.Printf("Setting up %s e2e test infrastructure", infraOpts.Provider)
	envOpts := []tftestenv.EnvironmentOption{
		tftestenv.WithExisting(infraOpts.Existing),
		tftestenv.WithRetain(infraOpts.Retain),
		tftestenv.WithVerbose(infraOpts.Verbose),
		tftestenv.WithCreateKubeconfig(providerCfg.createKubeconfig),
	}

	tmpDir, err := os.MkdirTemp("", "*-e2e")
	kubeconfigPath = fmt.Sprintf("%s/kubeconfig", tmpDir)
	defer func() {
		if ferr := os.RemoveAll(filepath.Dir(kubeconfigPath)); ferr != nil {
			err = multierr.Append(fmt.Errorf("could not clean up kubeconfig file: %v", ferr), err)
		}
	}()

	if err != nil {
		return 0, err
	}

	// Create terraform infrastructure
	testEnv, err = tftestenv.New(context.Background(), scheme.Scheme, providerCfg.terraformPath, kubeconfigPath, envOpts...)
	if err != nil {
		return 0, err
	}

	// get terrraform infrastructure
	outputs, err := testEnv.StateOutput(context.Background())
	if err != nil {
		return 0, err
	}

	// get provider specific test configuration
	cfg, err = providerCfg.getTestConfig(context.Background(), outputs)
	if err != nil {
		return 0, err
	}
	cfg.client = testEnv.Client

	testRepos, err = providerCfg.registryLogin(ctx, outputs)
	if err != nil {
		return 0, err
	}

	objectMap, kustomizeYaml, err := providerCfg.getFluxKustomization(context.Background(), outputs)
	if err != nil {
		return 0, err
	}

	err = setupScheme()
	if err != nil {
		return 0, err
	}

	// setup managed transport so the `pkg/libgit2` package can be used
	// for interacting repositories
	err = transport.InitManagedTransport()
	if err != nil {
		return 0, fmt.Errorf("could not init managed transport")
	}

	// Setup Kubernetes clients for test cluster
	err = installFlux(ctx, cfg.client, fluxConfig{
		kubeconfigPath: kubeconfigPath,
		repoURL:        cfg.fleetInfraRepository.http,
		password:       cfg.pat,
		objects:        objectMap,
		kustomizeYaml:  kustomizeYaml,
	})
	if err != nil {
		return 1, fmt.Errorf("error installing Flux: %v", err)
	}

	// Run tests
	log.Println("Running e2e tests")
	result := m.Run()

	if err := testEnv.Stop(ctx); err != nil {
		log.Printf("Failed to stop environment: %v", err)
	}

	return result, nil
}

func getProviderConfig(provider string) (*ProviderConfig, error) {
	switch provider {
	case "azure":
		return &ProviderConfig{
			terraformPath:        aksTerraformPath,
			createKubeconfig:     createKubeConfigAKS,
			getTestConfig:        getTestConfigAKS,
			getFluxKustomization: createAzureSPSecret,
			registryLogin:        registryLoginACR,
		}, nil
	default:
		return nil, fmt.Errorf("provider '%s' is not supported", provider)
	}
}
