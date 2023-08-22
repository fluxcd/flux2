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

package install

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-cleanhttp"

	"github.com/fluxcd/pkg/kustomize/filesys"
	"github.com/fluxcd/pkg/tar"

	"github.com/fluxcd/flux2/v2/pkg/manifestgen/kustomization"
)

func fetch(ctx context.Context, url, version, dir string) error {
	ghURL := fmt.Sprintf("%s/latest/download/manifests.tar.gz", url)
	if strings.HasPrefix(version, "v") {
		ghURL = fmt.Sprintf("%s/download/%s/manifests.tar.gz", url, version)
	}

	req, err := http.NewRequest("GET", ghURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for %s, error: %w", ghURL, err)
	}

	// download
	resp, err := cleanhttp.DefaultClient().Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to download manifests.tar.gz from %s, error: %w", ghURL, err)
	}
	defer resp.Body.Close()

	// check response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download manifests.tar.gz from %s, status: %s", ghURL, resp.Status)
	}

	// extract
	if err = tar.Untar(resp.Body, dir, tar.WithMaxUntarSize(-1)); err != nil {
		return fmt.Errorf("failed to untar manifests.tar.gz from %s, error: %w", ghURL, err)
	}

	return nil
}

func generate(base string, options Options) error {
	if containsItemString(options.Components, options.NotificationController) {
		// We need to use full domain name here, as some users may deploy flux
		// in environments that use http proxy.
		//
		// In such environments they normally add `.cluster.local` and `.local`
		// suffixes to `no_proxy` variable in order to prevent cluster-local
		// traffic from going through http proxy. Without fully specified
		// domain they need to mention `notifications-controller` explicitly in
		// `no_proxy` variable after debugging http proxy logs.
		options.EventsAddr = fmt.Sprintf("http://%s.%s.svc.%s./", options.NotificationController, options.Namespace, options.ClusterDomain)
	}

	if err := execTemplate(options, namespaceTmpl, path.Join(base, "namespace.yaml")); err != nil {
		return fmt.Errorf("generate namespace failed: %w", err)
	}

	if err := execTemplate(options, labelsTmpl, path.Join(base, "labels.yaml")); err != nil {
		return fmt.Errorf("generate labels failed: %w", err)
	}

	if err := execTemplate(options, nodeSelectorTmpl, path.Join(base, "node-selector.yaml")); err != nil {
		return fmt.Errorf("generate node selector failed: %w", err)
	}

	if err := execTemplate(options, kustomizationTmpl, path.Join(base, "kustomization.yaml")); err != nil {
		return fmt.Errorf("generate kustomization failed: %w", err)
	}

	if err := os.MkdirAll(path.Join(base, "roles"), os.ModePerm); err != nil {
		return fmt.Errorf("generate roles failed: %w", err)
	}

	if err := execTemplate(options, kustomizationRolesTmpl, path.Join(base, "roles/kustomization.yaml")); err != nil {
		return fmt.Errorf("generate roles kustomization failed: %w", err)
	}

	rbacFile := filepath.Join(base, "roles/rbac.yaml")
	if err := copyFile(filepath.Join(base, "rbac.yaml"), rbacFile); err != nil {
		return fmt.Errorf("generate rbac failed: %w", err)
	}

	// workaround for kustomize not being able to patch the SA in ClusterRoleBindings
	defaultNS := MakeDefaultOptions().Namespace
	if defaultNS != options.Namespace {
		rbac, err := os.ReadFile(rbacFile)
		if err != nil {
			return fmt.Errorf("reading rbac file failed: %w", err)
		}
		rbac = bytes.ReplaceAll(rbac, []byte(defaultNS), []byte(options.Namespace))
		if err := os.WriteFile(rbacFile, rbac, os.ModePerm); err != nil {
			return fmt.Errorf("replacing service account namespace in rbac failed: %w", err)
		}
	}
	return nil
}

func build(base, output string) error {
	resources, err := kustomization.Build(base)
	if err != nil {
		return err
	}

	outputBase := filepath.Dir(strings.TrimSuffix(output, string(filepath.Separator)))
	fs, err := filesys.MakeFsOnDiskSecure(outputBase)
	if err != nil {
		return err
	}
	if err = fs.WriteFile(output, resources); err != nil {
		return err
	}

	return nil
}
