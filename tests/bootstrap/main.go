package main

import (
	"context"
	"log"
	"os"

	"github.com/fluxcd/go-git-providers/github"
	"github.com/fluxcd/go-git-providers/gitprovider"
	"k8s.io/client-go/util/retry"
)

func main() {
	ks := "test-cluster/flux-system/kustomization.yaml"
	patchName := "test-cluster/flux-system/gotk-patches.yaml"
	ksContent := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- gotk-components.yaml
- gotk-sync.yaml
patches:
  - path: gotk-patches.yaml
    target:
      kind: Deployment`
	patchContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: all-flux-components
spec:
  template:
    metadata:
      annotations:
        # Required by Kubernetes node autoscaler
        cluster-autoscaler.kubernetes.io/safe-to-evict: "true"
    spec:
      securityContext:
        runAsUser: 10000
        fsGroup: 1337
      containers:
        - name: manager
          securityContext:
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
            runAsNonRoot: true
            capabilities:
              drop:
                - ALL
`
	commitFiles := []gitprovider.CommitFile{
		{
			Path:    &ks,
			Content: &ksContent,
		},
		{
			Path:    &patchName,
			Content: &patchContent,
		},
	}

	orgName := os.Getenv("GITHUB_ORG_NAME")
	repoName := os.Getenv("GITHUB_REPO_NAME")
	githubToken := os.Getenv(github.TokenVariable)
	client, err := github.NewClient(gitprovider.WithOAuth2Token(githubToken))
	if err != nil {
		log.Fatalf("error initializing github client: %s", err)
	}

	repoRef := gitprovider.OrgRepositoryRef{
		OrganizationRef: gitprovider.OrganizationRef{
			Organization: orgName,
			Domain:       github.DefaultDomain,
		},
		RepositoryName: repoName,
	}

	if ok, err := client.HasTokenPermission(context.Background(), gitprovider.TokenPermissionRWRepository); err != nil {
		log.Fatalf("error getting token permission: %s", err)
	} else {
		if !ok {
			log.Fatal("token has no write permissions")
		}
	}

	var repo gitprovider.OrgRepository
	err = retry.OnError(retry.DefaultRetry, func(err error) bool {
		return err != nil
	}, func() error {
		repo, err = client.OrgRepositories().Get(context.Background(), repoRef)
		return err
	})
	if err != nil {
		log.Fatalf("error getting %s repository in org %s: %s", repoRef.RepositoryName, repoRef.Organization, err)
	}

	_, err = repo.Commits().Create(context.Background(), "main", "add patch manifest 3", commitFiles)
	if err != nil {
		log.Fatalf("error making commit: %s", err)
	}
}
