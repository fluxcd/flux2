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

package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"text/template"
	"time"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/google/go-cmp/cmp"
	"github.com/mattn/go-shellwords"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var nextNamespaceId int64

// Return a unique namespace with the specified prefix, for tests to create
// objects that won't collide with each other.
func allocateNamespace(prefix string) string {
	id := atomic.AddInt64(&nextNamespaceId, 1)
	return fmt.Sprintf("%s-%d", prefix, id)
}

func readYamlObjects(rdr io.Reader) ([]unstructured.Unstructured, error) {
	objects := []unstructured.Unstructured{}
	reader := k8syaml.NewYAMLReader(bufio.NewReader(rdr))
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
		objects = append(objects, *unstructuredObj)
	}
	return objects, nil
}

// A KubeManager that can create objects that are subject to a test.
type testEnvKubeManager struct {
	client         client.WithWatch
	testEnv        *envtest.Environment
	kubeConfigPath string
}

func (m *testEnvKubeManager) CreateObjectFile(objectFile string, templateValues map[string]string, t *testing.T) {
	buf, err := os.ReadFile(objectFile)
	if err != nil {
		t.Fatalf("Error reading file '%s': %v", objectFile, err)
	}
	content, err := executeTemplate(string(buf), templateValues)
	if err != nil {
		t.Fatalf("Error evaluating template file '%s': '%v'", objectFile, err)
	}
	clientObjects, err := readYamlObjects(strings.NewReader(content))
	if err != nil {
		t.Fatalf("Error decoding yaml file '%s': %v", objectFile, err)
	}
	err = m.CreateObjects(clientObjects, t)
	if err != nil {
		t.Logf("Error creating test objects: '%v'", err)
	}
}

func (m *testEnvKubeManager) CreateObjects(clientObjects []unstructured.Unstructured, t *testing.T) error {
	for _, obj := range clientObjects {
		// First create the object then set its status if present in the
		// yaml file. Make a copy first since creating an object may overwrite
		// the status.
		createObj := obj.DeepCopy()
		err := m.client.Create(context.Background(), createObj)
		if err != nil {
			return err
		}
		obj.SetResourceVersion(createObj.GetResourceVersion())
		err = m.client.Status().Update(context.Background(), &obj)
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
	case TestEnvClusterMode:
		useExistingCluster := false
		testEnv := &envtest.Environment{
			UseExistingCluster: &useExistingCluster,
			CRDDirectoryPaths:  []string{"manifests"},
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
		os.WriteFile(tmpFilename, kubeConfig, 0644)
		k8sClient, err := client.NewWithWatch(cfg, client.Options{
			Scheme: utils.NewScheme(),
		})
		if err != nil {
			return nil, err
		}
		return &testEnvKubeManager{
			testEnv:        testEnv,
			client:         k8sClient,
			kubeConfigPath: tmpFilename,
		}, nil
	case ExistingClusterMode:
		// TEST_KUBECONFIG is mandatory to prevent destroying a current cluster accidentally.
		testKubeConfig := os.Getenv("TEST_KUBECONFIG")
		if testKubeConfig == "" {
			return nil, fmt.Errorf("environment variable TEST_KUBECONFIG is required to run tests against an existing cluster")
		}

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
		k8sClient, err := client.NewWithWatch(cfg, client.Options{
			Scheme: utils.NewScheme(),
		})
		if err != nil {
			return nil, err
		}
		return &testEnvKubeManager{
			testEnv:        testEnv,
			client:         k8sClient,
			kubeConfigPath: testKubeConfig,
		}, nil
	}

	return nil, nil
}

// Function that sets an expectation on the output of a command. Tests can
// either implement this directly or use a helper below.
type assertFunc func(output string, err error) error

// Assemble multiple assertFuncs into a single assertFunc
func assert(fns ...assertFunc) assertFunc {
	return func(output string, err error) error {
		for _, fn := range fns {
			if assertErr := fn(output, err); assertErr != nil {
				return assertErr
			}
		}
		return nil
	}
}

// Expect the command to run without error
func assertSuccess() assertFunc {
	return func(output string, err error) error {
		if err != nil {
			return fmt.Errorf("Expected success but was error: %v", err)
		}
		return nil
	}
}

// Expect the command to fail with the specified error
func assertError(expected string) assertFunc {
	return func(output string, err error) error {
		if err == nil {
			return fmt.Errorf("Expected error but was success")
		}
		if expected != err.Error() {
			return fmt.Errorf("Expected error '%v' but got '%v'", expected, err.Error())
		}
		return nil
	}
}

// Expect the command to succeed with the expected test output.
func assertGoldenValue(expected string) assertFunc {
	return assert(
		assertSuccess(),
		func(output string, err error) error {
			diff := cmp.Diff(expected, output)
			if diff != "" {
				return fmt.Errorf("Mismatch from expected value (-want +got):\n%s", diff)
			}
			return nil
		})
}

// Filename that contains the expected test output.
func assertGoldenFile(goldenFile string) assertFunc {
	return assertGoldenTemplateFile(goldenFile, map[string]string{})
}

// Filename that contains the expected test output. The golden file is a template that
// is pre-processed with the specified templateValues.
func assertGoldenTemplateFile(goldenFile string, templateValues map[string]string) assertFunc {
	goldenFileContents, fileErr := os.ReadFile(goldenFile)
	return func(output string, err error) error {
		if fileErr != nil {
			return fmt.Errorf("Error reading golden file '%s': %s", goldenFile, fileErr)
		}
		var expectedOutput string
		if len(templateValues) > 0 {
			expectedOutput, err = executeTemplate(string(goldenFileContents), templateValues)
			if err != nil {
				return fmt.Errorf("Error executing golden template file '%s': %s", goldenFile, err)
			}
		} else {
			expectedOutput = string(goldenFileContents)
		}
		if assertErr := assertGoldenValue(expectedOutput)(output, err); assertErr != nil {
			return fmt.Errorf("Mismatch from golden file '%s': %v", goldenFile, assertErr)
		}
		return nil
	}
}

type TestClusterMode int

const (
	TestEnvClusterMode = TestClusterMode(iota + 1)
	ExistingClusterMode
)

// Structure used for each test to load objects into kubernetes, run
// commands and assert on the expected output.
type cmdTestCase struct {
	// The command line arguments to test.
	args string
	// Tests use assertFunc to assert on an output, success or failure. This
	// can be a function defined by the test or existing function above.
	assert assertFunc
	// Filename that contains yaml objects to load into Kubernetes
	objectFile string
}

func (cmd *cmdTestCase) runTestCmd(t *testing.T) {
	actual, testErr := executeCommand(cmd.args)
	if assertErr := cmd.assert(actual, testErr); assertErr != nil {
		t.Error(assertErr)
	}
}

func executeTemplate(content string, templateValues map[string]string) (string, error) {
	tmpl := template.Must(template.New("golden").Parse(content))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, templateValues); err != nil {
		return "", err
	}
	return out.String(), nil
}

// Run the command and return the captured output.
func executeCommand(cmd string) (string, error) {
	defer resetCmdArgs()
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

func resetCmdArgs() {
	createArgs = createFlags{}
	getArgs = GetFlags{}
	secretGitArgs = NewSecretGitFlags()
}
