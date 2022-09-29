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
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	tfjson "github.com/hashicorp/terraform-json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	helmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	automationv1beta1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
	reflectorv1beta2 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notiv1beta2 "github.com/fluxcd/notification-controller/api/v1beta2"
	"github.com/fluxcd/pkg/git"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/fluxcd/test-infra/tftestenv"
)

const (
	// azureTerraformPath is the path to the folder containing the
	// terraform files for azure infra
	azureTerraformPath = "./terraform/azure"

	// kubeconfigPath is the path of the file containing the kubeconfig
	kubeconfigPath = "./build/kubeconfig"

	// default branch to be used when cloning git repositories
	defaultBranch = "main"
)

var (
	// supportedProviders are the providers supported by the test.
	supportedProviders = []string{"azure"}

	// cfg is a struct containing different variables needed for the test.
	cfg *testConfig

	// infraOpts are the options for running the terraform environment
	infraOpts tftestenv.Options

	// versions to tag and push for the podinfo image
	oldPodinfoVersion = "6.0.0"
	newPodinfoVersion = "6.0.1"
	podinfoTags       = []string{oldPodinfoVersion, newPodinfoVersion}

	// testEnv is the test environment. It contains test infrastructure and
	// kubernetes client of the created cluster.
	testEnv *tftestenv.Environment

	// testTimeout is used as a timeout when testing a condition with gomega's eventually
	testTimeout = 60 * time.Second
	// testInterval is used as an interval when testing a condition with gomega's eventually
	testInterval = 5 * time.Second

	random *rand.Rand

	letterRunes = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

	localImg = "ghcr.io/stefanprodan/podinfo"
)

// testConfig hold different variable that will be needed by the different test functions.
type testConfig struct {
	// authentication info for git repositories
	gitPat                string
	gitUsername           string
	gitPrivateKey         string
	gitPublicKey          string
	defaultGitTransport   git.TransportType
	defaultAuthOpts       *git.AuthOptions
	knownHosts            string
	fleetInfraRepository  repoConfig
	applicationRepository repoConfig

	notificationURL string

	// cloud provider dependent argument to pass to the sops cli
	sopsArgs string
	// secret data for sops
	sopsSecretData map[string]string
	// envCredsData are data field for a secret containing environment variables that the Flux deployments
	// may need
	envCredsData map[string]string
	// kustomizationYaml is the  content of the kustomization.yaml for customizing the Flux manifests
	kustomizationYaml string

	// testRegistry is the registry of the cloud provider.
	testRegistry string
}

// repoConfig contains the http/ssh urls for the created git repositories
// on the various cloud providers.
type repoConfig struct {
	http string
	ssh  string
}

// getTestConfig gets the test configuration that contains different variables for running the tests
type getTestConfig func(ctx context.Context, output map[string]*tfjson.StateOutput) (*testConfig, error)

// registryLoginFunc is used to perform registry login against a provider based
// on the terraform state output values. It returns the test registry
// to test against, read from the terraform state output.
type registryLoginFunc func(ctx context.Context, output map[string]*tfjson.StateOutput) (string, error)

// providerConfig contains the test configurations for the different cloud providers
type providerConfig struct {
	terraformPath    string
	createKubeconfig tftestenv.CreateKubeconfig
	getTestConfig    getTestConfig
	// registryLogin is used to perform registry login.
	registryLogin registryLoginFunc
}

func init() {
	utilruntime.Must(sourcev1.AddToScheme(scheme.Scheme))
	utilruntime.Must(sourcev1beta2.AddToScheme(scheme.Scheme))
	utilruntime.Must(kustomizev1.AddToScheme(scheme.Scheme))
	utilruntime.Must(helmv2beta1.AddToScheme(scheme.Scheme))
	utilruntime.Must(reflectorv1beta2.AddToScheme(scheme.Scheme))
	utilruntime.Must(automationv1beta1.AddToScheme(scheme.Scheme))
	utilruntime.Must(notiv1beta2.AddToScheme(scheme.Scheme))

	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func TestMain(m *testing.M) {
	ctx := context.TODO()

	infraOpts.Bindflags(flag.CommandLine)
	flag.Parse()

	// Validate the provider.
	if infraOpts.Provider == "" {
		log.Fatalf("-provider flag must be set to one of %v", supportedProviders)
	}
	var supported bool
	for _, p := range supportedProviders {
		if p == infraOpts.Provider {
			supported = true
			break
		}
	}
	if !supported {
		log.Fatalf("Unsupported provider %q, must be one of %v", infraOpts.Provider, supportedProviders)
	}
	// get provider specific configuration
	providerCfg := getProviderConfig(infraOpts.Provider)
	if providerCfg == nil {
		log.Fatalf("Failed to get provider config for %q", infraOpts.Provider)
	}

	// Initialize with non-zero exit code to indicate failure by default unless
	// set by a successful test run.
	exitCode := 1

	// Setup Terraform binary and init state
	log.Printf("Setting up %s e2e test infrastructure", infraOpts.Provider)
	envOpts := []tftestenv.EnvironmentOption{
		tftestenv.WithExisting(infraOpts.Existing),
		tftestenv.WithRetain(infraOpts.Retain),
		tftestenv.WithVerbose(infraOpts.Verbose),
		tftestenv.WithCreateKubeconfig(providerCfg.createKubeconfig),
	}

	// Create terraform infrastructure
	var err error
	testEnv, err = tftestenv.New(ctx, scheme.Scheme, providerCfg.terraformPath, kubeconfigPath, envOpts...)
	if err != nil {
		log.Fatalf("Failed to provision the test infrastructure: %v", err)
	}

	defer func() {
		if err := testEnv.Stop(ctx); err != nil {
			log.Printf("Failed to stop environment: %v", err)
		}

		// Calling exit on panic prevents logging of panic error.
		// Exit only on normal return. Explicitly detect panic and log the error
		// on panic.
		if err := recover(); err == nil {
			os.Exit(exitCode)
		} else {
			log.Printf("panic: %v", err)
		}
	}()

	// get terrraform infrastructure
	outputs, err := testEnv.StateOutput(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to get the terraform state output: %v", err))
	}

	// get provider specific test configuration
	cfg, err = providerCfg.getTestConfig(ctx, outputs)
	if err != nil {
		panic(fmt.Sprintf("Failed to get provider config for %v", err))
	}

	regUrl, err := providerCfg.registryLogin(ctx, outputs)
	if err != nil {
		panic(fmt.Sprintf("Failed to log into registry: %v", err))
	}

	cfg.testRegistry = regUrl
	err = pushTestImages(ctx, cfg.testRegistry, podinfoTags)
	if err != nil {
		panic(fmt.Sprintf("Failed to push test images: %v", err))
	}

	tmpDir, err := os.MkdirTemp("", "*-flux-test")
	if err != nil {
		panic(fmt.Sprintf("Failed to create tmp dir: %v", err))
	}
	defer func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			log.Printf("error removing tmp dir: %s\n", err)
		}
	}()

	log.Println("Installing flux")
	err = installFlux(ctx, tmpDir, kubeconfigPath)
	defer func() {
		log.Println("Uninstalling Flux")
		if err := uninstallFlux(ctx); err != nil {
			log.Printf("Failed to uninstall: %v", err)
		}
	}()
	if err != nil {
		panic(fmt.Sprintf("error installing Flux: %v", err))
	}

	log.Println("Running e2e tests")
	exitCode = m.Run()
}

func getProviderConfig(provider string) *providerConfig {
	switch provider {
	case "azure":
		return &providerConfig{
			terraformPath:    azureTerraformPath,
			createKubeconfig: createKubeConfigAKS,
			getTestConfig:    getTestConfigAKS,
			registryLogin:    registryLoginACR,
		}
	}

	return nil
}

// pushTestImages pushes the local podinfo image to the remote repository specified
// by repoURL. The image should be existing on the machine.
func pushTestImages(ctx context.Context, repoURL string, tags []string) error {
	for _, tag := range tags {
		remoteImg := fmt.Sprintf("%s/podinfo:%s", repoURL, tag)
		err := tftestenv.RetagAndPush(ctx, fmt.Sprintf("%s:%s", localImg, tag), remoteImg)
		if err != nil {
			return err
		}

	}
	return nil
}

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[random.Intn(len(letterRunes))]
	}
	return string(b)
}
