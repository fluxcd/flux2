package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/google/go-cmp/cmp"
	"github.com/mattn/go-shellwords"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func readYamlObjects(objectFile string) ([]client.Object, error) {
	obj, err := os.ReadFile(objectFile)
	if err != nil {
		return nil, err
	}
	objects := []client.Object{}
	reader := k8syaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(obj)))
	for {
		doc, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
		}
		unstructuredObj := &unstructured.Unstructured{}
		decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewBuffer(doc), len(doc))
		err = decoder.Decode(unstructuredObj)
		if err != nil {
			return nil, err
		}
		objects = append(objects, unstructuredObj)
	}
	return objects, nil
}

// A KubeManager that can create objects that are subject to a test.
type testEnvKubeManager struct {
	client  client.WithWatch
	testEnv *envtest.Environment
}

func (m *testEnvKubeManager) NewClient(kubeconfig string, kubecontext string) (client.WithWatch, error) {
	return m.client, nil
}

func (m *testEnvKubeManager) CreateObjects(clientObjects []client.Object) error {
	for _, obj := range clientObjects {
		err := m.client.Create(context.Background(), obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *testEnvKubeManager) Stop() error {
	if m.testEnv == nil {
		return fmt.Errorf("do nothing because testEnv is nil")
	}
	return m.testEnv.Stop()
}

func NewTestEnvKubeManager(testClusterMode TestClusterMode) (*testEnvKubeManager, error) {
	switch testClusterMode {
	case FakeClusterMode:
		c := fakeclient.NewClientBuilder().WithScheme(utils.NewScheme()).Build()
		return &testEnvKubeManager{
			client: c,
		}, nil
	case TestEnvClusterMode:
		useExistingCluster := false
		testEnv := &envtest.Environment{
			UseExistingCluster: &useExistingCluster,
		}
		cfg, err := testEnv.Start()
		if err != nil {
			return nil, err
		}
		user, err := testEnv.ControlPlane.AddUser(envtest.User{
			Name:   "envtest-admin",
			Groups: []string{"system:masters"},
		}, nil)
		if err != nil {
			return nil, err
		}

		kubeConfig, err := user.KubeConfig()
		if err != nil {
			return nil, err
		}

		tmpFilename := filepath.Join("/tmp", "kubeconfig-"+time.Nanosecond.String())
		ioutil.WriteFile(tmpFilename, kubeConfig, 0644)
		rootArgs.kubeconfig = tmpFilename
		k8sClient, err := client.NewWithWatch(cfg, client.Options{})
		if err != nil {
			return nil, err
		}
		return &testEnvKubeManager{
			testEnv: testEnv,
			client:  k8sClient,
		}, nil
	case ExistingClusterMode:
		// TEST_KUBECONFIG is mandatory to prevent destroying a current cluster accidentally.
		testKubeConfig := os.Getenv("TEST_KUBECONFIG")
		if testKubeConfig == "" {
			return nil, fmt.Errorf("environment variable TEST_KUBECONFIG is required to run tests against an existing cluster")
		}
		rootArgs.kubeconfig = testKubeConfig

		useExistingCluster := true
		config, err := clientcmd.BuildConfigFromFlags("", testKubeConfig)
		testEnv := &envtest.Environment{
			UseExistingCluster: &useExistingCluster,
			Config:             config,
		}
		cfg, err := testEnv.Start()
		if err != nil {
			return nil, err
		}
		k8sClient, err := client.NewWithWatch(cfg, client.Options{})
		if err != nil {
			return nil, err
		}
		return &testEnvKubeManager{
			testEnv: testEnv,
			client:  k8sClient,
		}, nil
	}

	return nil, nil
}

type TestClusterMode int

const (
	FakeClusterMode = TestClusterMode(iota + 1)
	TestEnvClusterMode
	ExistingClusterMode
)

// Structure used for each test to load objects into kubernetes, run
// commands and assert on the expected output.
type cmdTestCase struct {
	// The command line arguments to test.
	args string
	// When true, the test expects the command to fail.
	wantError bool
	// String literal that contains the expected test output.
	goldenValue string
	// Filename that contains the expected test output.
	goldenFile string
	// Filename that contains yaml objects to load into Kubernetes
	objectFile string
	// TestClusterMode to bootstrap and testing, default to Fake
	testClusterMode TestClusterMode
}

func (cmd *cmdTestCase) runTestCmd(t *testing.T) {
	km, err := NewTestEnvKubeManager(cmd.testClusterMode)
	if err != nil {
		t.Fatalf("Error creating kube manager: '%v'", err)
	}

	if km != nil {
		rootCtx.kubeManager = km
		defer km.Stop()
	}

	if cmd.objectFile != "" {
		clientObjects, err := readYamlObjects(cmd.objectFile)
		if err != nil {
			t.Fatalf("Error loading yaml: '%v'", err)
		}
		err = km.CreateObjects(clientObjects)
		if err != nil {
			t.Fatalf("Error creating test objects: '%v'", err)
		}
	}

	actual, err := executeCommand(cmd.args)
	if (err != nil) != cmd.wantError {
		t.Fatalf("Expected error='%v', Got: %v", cmd.wantError, err)
	}
	if err != nil {
		actual = err.Error()
	}

	var expected string
	if cmd.goldenValue != "" {
		expected = cmd.goldenValue
	}
	if cmd.goldenFile != "" {
		expectedOutput, err := os.ReadFile(cmd.goldenFile)
		if err != nil {
			t.Fatalf("Error reading golden file: '%s'", err)
		}
		expected = string(expectedOutput)
	}

	diff := cmp.Diff(expected, actual)
	if diff != "" {
		t.Errorf("Mismatch from '%s' (-want +got):\n%s", cmd.goldenFile, diff)
	}
}

// Run the command and return the captured output.
func executeCommand(cmd string) (string, error) {
	args, err := shellwords.Parse(cmd)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)

	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	logger.stderr = rootCmd.ErrOrStderr()

	_, err = rootCmd.ExecuteC()
	result := buf.String()

	return result, err
}
