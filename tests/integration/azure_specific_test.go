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

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
	. "github.com/onsi/gomega"
	giturls "github.com/whilp/git-urls"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	notiv1beta1 "github.com/fluxcd/notification-controller/api/v1beta1"
	"github.com/fluxcd/pkg/runtime/events"
)

func TestEventHubNotification(t *testing.T) {
	g := NewWithT(t)
	// Currently, only azuredevops is supported
	if infraOpts.Provider != "azure" {
		fmt.Printf("Skipping event notification for %s as it is not supported.\n", infraOpts.Provider)
		return
	}

	ctx := context.TODO()
	name := "event-hub"

	// Start listening to eventhub with latest offset
	hub, err := eventhub.NewHubFromConnectionString(cfg.eventHubSas)
	g.Expect(err).ToNot(HaveOccurred())
	c := make(chan string, 10)
	handler := func(ctx context.Context, event *eventhub.Event) error {
		c <- string(event.Data)
		return nil
	}
	runtimeInfo, err := hub.GetRuntimeInformation(ctx)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(len(runtimeInfo.PartitionIDs)).To(Equal(1))
	listenerHandler, err := hub.Receive(ctx, runtimeInfo.PartitionIDs[0], handler)
	g.Expect(err).ToNot(HaveOccurred())

	// Setup Flux resources
	repoUrl := cfg.applicationRepository.http
	manifest, err := executeTemplate("testdata/cm.yaml", map[string]string{
		"ns": name,
	})

	repo, err := cloneRepository(repoUrl, name, cfg.pat)
	repoDir := repo.Path()
	g.Expect(err).ToNot(HaveOccurred())
	err = addFile(repoDir, "configmap.yaml", manifest)
	g.Expect(err).ToNot(HaveOccurred())
	err = commitAndPushAll(ctx, repo, name, cfg.pat)
	g.Expect(err).ToNot(HaveOccurred())
	err = setupNamespace(ctx, name, nsConfig{})
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, cfg.client, &secret, func() error {
		secret.StringData = map[string]string{
			"address": cfg.eventHubSas,
		}
		return nil
	})
	provider := notiv1beta1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, cfg.client, &provider, func() error {
		provider.Spec = notiv1beta1.ProviderSpec{
			Type:    "azureeventhub",
			Address: repoUrl,
			SecretRef: &meta.LocalObjectReference{
				Name: name,
			},
		}
		return nil
	})
	g.Expect(err).ToNot(HaveOccurred())
	alert := notiv1beta1.Alert{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, cfg.client, &alert, func() error {
		alert.Spec = notiv1beta1.AlertSpec{
			ProviderRef: meta.LocalObjectReference{
				Name: provider.Name,
			},
			EventSources: []notiv1beta1.CrossNamespaceObjectReference{
				{
					Kind:      "Kustomization",
					Name:      name,
					Namespace: name,
				},
			},
		}
		return nil
	})
	g.Expect(err).ToNot(HaveOccurred())
	modifyKsSpec := func(spec *kustomizev1.KustomizationSpec) {
		spec.HealthChecks = []meta.NamespacedObjectKindReference{
			{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       "foobar",
				Namespace:  name,
			},
		}
	}
	err = setupNamespace(ctx, name, nsConfig{
		repoURL:      repoUrl,
		path:         "./",
		modifyKsSpec: modifyKsSpec,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Eventually(func() bool {
		err := verifyGitAndKustomization(ctx, cfg.client, name, name)
		if err != nil {
			return false
		}
		return true
	}, 60*time.Second, 5*time.Second)

	// Wait to read even from event hub
	g.Eventually(func() bool {
		select {
		case eventJson := <-c:
			event := &events.Event{}
			err := json.Unmarshal([]byte(eventJson), event)
			if err != nil {
				t.Logf("the received event type does not match Flux format, error: %v", err)
				return false
			}

			if event.InvolvedObject.Kind == kustomizev1.KustomizationKind &&
				strings.Contains(event.Message, "Health check passed") {
				return true
			}

			t.Logf("event received from '%s/%s': %s",
				event.InvolvedObject.Kind, event.InvolvedObject.Name, event.Message)
			return false
		default:
			return false
		}
	}, 60*time.Second, 1*time.Second)
	err = listenerHandler.Close(ctx)
	g.Expect(err).ToNot(HaveOccurred())
	err = hub.Close(ctx)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestAzureDevOpsCommitStatus(t *testing.T) {
	g := NewWithT(t)

	// Currently, only azuredevops is supported
	if infraOpts.Provider != "azure" {
		fmt.Printf("Skipping commit status test for %s as it is not supported.\n", infraOpts.Provider)
		return
	}

	ctx := context.TODO()
	name := "commit-status"
	repoUrl := cfg.applicationRepository.http
	manifest, err := executeTemplate("testdata/cm.yaml", map[string]string{
		"ns": name,
	})
	g.Expect(err).ToNot(HaveOccurred())

	c, err := cloneRepository(repoUrl, name, cfg.pat)
	g.Expect(err).ToNot(HaveOccurred())
	repoDir := c.Path()
	err = addFile(repoDir, "configmap.yaml", manifest)
	g.Expect(err).ToNot(HaveOccurred())
	err = commitAndPushAll(ctx, c, name, cfg.pat)
	g.Expect(err).ToNot(HaveOccurred())

	modifyKsSpec := func(spec *kustomizev1.KustomizationSpec) {
		spec.HealthChecks = []meta.NamespacedObjectKindReference{
			{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       "foobar",
				Namespace:  name,
			},
		}
	}
	err = setupNamespace(ctx, name, nsConfig{
		repoURL:      repoUrl,
		path:         "./",
		modifyKsSpec: modifyKsSpec,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Eventually(func() bool {
		err := verifyGitAndKustomization(ctx, cfg.client, name, name)
		if err != nil {
			return false
		}
		return true
	}, 10*time.Second, 1*time.Second)

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "azuredevops-token",
			Namespace: name,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, cfg.client, &secret, func() error {
		secret.StringData = map[string]string{
			"token": cfg.pat,
		}
		return nil
	})

	provider := notiv1beta1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "azuredevops",
			Namespace: name,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, cfg.client, &provider, func() error {
		provider.Spec = notiv1beta1.ProviderSpec{
			Type:    "azuredevops",
			Address: repoUrl,
			SecretRef: &meta.LocalObjectReference{
				Name: "azuredevops-token",
			},
		}
		return nil
	})
	g.Expect(err).ToNot(HaveOccurred())

	alert := notiv1beta1.Alert{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "azuredevops",
			Namespace: name,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, cfg.client, &alert, func() error {
		alert.Spec = notiv1beta1.AlertSpec{
			ProviderRef: meta.LocalObjectReference{
				Name: provider.Name,
			},
			EventSources: []notiv1beta1.CrossNamespaceObjectReference{
				{
					Kind:      "Kustomization",
					Name:      name,
					Namespace: name,
				},
			},
		}
		return nil
	})
	g.Expect(err).ToNot(HaveOccurred())

	url, err := ParseGitAddress(repoUrl)
	g.Expect(err).ToNot(HaveOccurred())

	rev, err := c.Head()
	connection := azuredevops.NewPatConnection(url.OrgURL, cfg.pat)
	client, err := git.NewClient(ctx, connection)
	g.Expect(err).ToNot(HaveOccurred())
	getArgs := git.GetStatusesArgs{
		Project:      &url.Project,
		RepositoryId: &url.Repo,
		CommitId:     &rev,
	}
	g.Eventually(func() bool {
		statuses, err := client.GetStatuses(ctx, getArgs)
		if err != nil {
			return false
		}
		if len(*statuses) != 1 {
			return false
		}
		return true
	}, 500*time.Second, 5*time.Second)
}

type AzureDevOpsURL struct {
	OrgURL  string
	Project string
	Repo    string
}

// TODO(somtochiama): move this into fluxcd/pkg and reuse in NC
func ParseGitAddress(s string) (AzureDevOpsURL, error) {
	var args AzureDevOpsURL
	u, err := giturls.Parse(s)
	if err != nil {
		return args, nil
	}

	scheme := u.Scheme
	if u.Scheme == "ssh" {
		scheme = "https"
	}

	id := strings.TrimLeft(u.Path, "/")
	id = strings.TrimSuffix(id, ".git")

	comp := strings.Split(id, "/")
	if len(comp) != 4 {
		return args, fmt.Errorf("invalid repository id %q", id)
	}

	args = AzureDevOpsURL{
		OrgURL:  fmt.Sprintf("%s://%s/%s", scheme, u.Host, comp[0]),
		Project: comp[1],
		Repo:    comp[3],
	}

	return args, nil

}
