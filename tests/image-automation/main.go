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
	ksPath := "test-cluster/podinfo-auto/kustomization.yaml"
	autoPath := "test-cluster/podinfo-auto/auto.yaml"

	ksContent, err := os.ReadFile("kustomization.yaml")
	if err != nil {
		log.Fatal(err)
	}
	ks := string(ksContent)

	autoContent, err := os.ReadFile("auto.yaml")
	if err != nil {
		log.Fatal(err)
	}
	auto := string(autoContent)

	commitFiles := []gitprovider.CommitFile{
		{
			Path:    &ksPath,
			Content: &ks,
		},
		{
			Path:    &autoPath,
			Content: &auto,
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

	_, err = repo.Commits().Create(context.Background(), "main", "automation test", commitFiles)
	if err != nil {
		log.Fatalf("error making commit: %s", err)
	}
}
