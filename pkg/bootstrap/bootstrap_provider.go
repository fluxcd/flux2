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

package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/go-git-providers/gitprovider"

	"github.com/fluxcd/flux2/v2/pkg/bootstrap/provider"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sync"
	"github.com/fluxcd/pkg/git/repository"
)

type GitProviderBootstrapper struct {
	*PlainGitBootstrapper

	owner          string
	repositoryName string
	repository     gitprovider.UserRepository

	personal bool

	description   string
	defaultBranch string
	visibility    string

	reconcile bool

	teams map[string]string

	readWriteKey bool

	bootstrapTransportType string
	syncTransportType      string

	sshHostname string

	useDeployTokenAuth bool

	provider gitprovider.Client
}

func NewGitProviderBootstrapper(git repository.Client, provider gitprovider.Client,
	kube client.Client, opts ...GitProviderOption) (*GitProviderBootstrapper, error) {
	b := &GitProviderBootstrapper{
		PlainGitBootstrapper: &PlainGitBootstrapper{
			gitClient: git,
			kube:      kube,
		},
		bootstrapTransportType: "https",
		syncTransportType:      "ssh",
		provider:               provider,
	}
	b.PlainGitBootstrapper.postGenerateSecret = append(b.PlainGitBootstrapper.postGenerateSecret, b.reconcileDeployKey)
	for _, opt := range opts {
		opt.applyGitProvider(b)
	}
	return b, nil
}

type GitProviderOption interface {
	applyGitProvider(b *GitProviderBootstrapper)
}

func WithProviderRepository(owner, repositoryName string, personal bool) GitProviderOption {
	return providerRepositoryOption{
		owner:          owner,
		repositoryName: repositoryName,
		personal:       personal,
	}
}

type providerRepositoryOption struct {
	owner          string
	repositoryName string
	personal       bool
}

func (o providerRepositoryOption) applyGitProvider(b *GitProviderBootstrapper) {
	b.owner = o.owner
	b.repositoryName = o.repositoryName
	b.personal = o.personal
}

func WithProviderRepositoryConfig(description, defaultBranch, visibility string) GitProviderOption {
	return providerRepositoryConfigOption{
		description:   description,
		defaultBranch: defaultBranch,
		visibility:    visibility,
	}
}

type providerRepositoryConfigOption struct {
	description   string
	defaultBranch string
	visibility    string
}

func (o providerRepositoryConfigOption) applyGitProvider(b *GitProviderBootstrapper) {
	b.description = o.description
	b.defaultBranch = o.defaultBranch
	b.visibility = o.visibility
}

func WithProviderTeamPermissions(teams map[string]string) GitProviderOption {
	return providerRepositoryTeamPermissionsOption(teams)
}

type providerRepositoryTeamPermissionsOption map[string]string

func (o providerRepositoryTeamPermissionsOption) applyGitProvider(b *GitProviderBootstrapper) {
	b.teams = o
}

func WithReadWriteKeyPermissions(b bool) GitProviderOption {
	return withReadWriteKeyPermissionsOption(b)
}

type withReadWriteKeyPermissionsOption bool

func (o withReadWriteKeyPermissionsOption) applyGitProvider(b *GitProviderBootstrapper) {
	b.readWriteKey = bool(o)
}

func WithBootstrapTransportType(protocol string) GitProviderOption {
	return bootstrapTransportTypeOption(protocol)
}

type bootstrapTransportTypeOption string

func (o bootstrapTransportTypeOption) applyGitProvider(b *GitProviderBootstrapper) {
	b.bootstrapTransportType = string(o)
}

func WithSyncTransportType(protocol string) GitProviderOption {
	return syncProtocolOption(protocol)
}

type syncProtocolOption string

func (o syncProtocolOption) applyGitProvider(b *GitProviderBootstrapper) {
	b.syncTransportType = string(o)
}

func WithSSHHostname(hostname string) GitProviderOption {
	return sshHostnameOption(hostname)
}

type sshHostnameOption string

func (o sshHostnameOption) applyGitProvider(b *GitProviderBootstrapper) {
	b.sshHostname = string(o)
}

func WithReconcile() GitProviderOption {
	return reconcileOption(true)
}

type reconcileOption bool

func (o reconcileOption) applyGitProvider(b *GitProviderBootstrapper) {
	b.reconcile = true
}

func WithDeployTokenAuth() GitProviderOption {
	return deployTokenAuthOption(true)
}

type deployTokenAuthOption bool

func (o deployTokenAuthOption) applyGitProvider(b *GitProviderBootstrapper) {
	b.useDeployTokenAuth = true
}

func (b *GitProviderBootstrapper) ReconcileSyncConfig(ctx context.Context, options sync.Options) error {
	if b.repository == nil {
		return errors.New("repository is required")
	}

	if b.url == "" {
		bootstrapURL, err := b.getCloneURL(b.repository, gitprovider.TransportType(b.bootstrapTransportType))
		if err != nil {
			return err
		}
		WithRepositoryURL(bootstrapURL).applyGit(b.PlainGitBootstrapper)
	}
	if options.URL == "" {
		syncURL, err := b.getCloneURL(b.repository, gitprovider.TransportType(b.syncTransportType))
		if err != nil {
			return err
		}

		options.URL = syncURL
	}

	return b.PlainGitBootstrapper.ReconcileSyncConfig(ctx, options)
}

func (b *GitProviderBootstrapper) ReconcileSourceSecret(ctx context.Context, options sourcesecret.Options) error {
	if b.repository == nil {
		return errors.New("repository is required")
	}

	if b.useDeployTokenAuth {
		deployTokenInfo, err := b.reconcileDeployToken(ctx, options)
		if err != nil {
			return err
		}

		if deployTokenInfo != nil {
			options.Username = deployTokenInfo.Username
			options.Password = deployTokenInfo.Token
		}
	}

	return b.PlainGitBootstrapper.ReconcileSourceSecret(ctx, options)
}

// ReconcileRepository reconciles an organization or user repository with the
// GitProviderBootstrapper configuration. On success, the URL in the embedded
// PlainGitBootstrapper is set to clone URL for the configured protocol.
//
// When part of the reconciliation fails with a warning without aborting, an
// ErrReconciledWithWarning error is returned.
func (b *GitProviderBootstrapper) ReconcileRepository(ctx context.Context) error {
	var repo gitprovider.UserRepository
	var err error
	if b.personal {
		repo, err = b.reconcileUserRepository(ctx)
	} else {
		repo, err = b.reconcileOrgRepository(ctx)
	}
	if err != nil && !errors.Is(err, ErrReconciledWithWarning) {
		return err
	}

	cloneURL, err := b.getCloneURL(repo, gitprovider.TransportType(b.bootstrapTransportType))
	if err != nil {
		return err
	}

	b.repository = repo
	WithRepositoryURL(cloneURL).applyGit(b.PlainGitBootstrapper)

	return err
}

func (b *GitProviderBootstrapper) reconcileDeployKey(ctx context.Context, secret corev1.Secret, options sourcesecret.Options) error {
	if b.repository == nil {
		return errors.New("repository is required")
	}

	ppk, ok := secret.StringData[sourcesecret.PublicKeySecretKey]
	if !ok {
		return nil
	}
	b.logger.Successf("public key: %s", strings.TrimSpace(ppk))

	name := deployKeyName(options.Namespace, b.branch, options.Name, options.TargetPath)
	deployKeyInfo := newDeployKeyInfo(name, ppk, b.readWriteKey)

	_, changed, err := b.repository.DeployKeys().Reconcile(ctx, deployKeyInfo)
	if err != nil {
		return err
	}
	if changed {
		b.logger.Successf("configured deploy key %q for %q", deployKeyInfo.Name, b.repository.Repository().String())
	}
	return nil
}

func (b *GitProviderBootstrapper) reconcileDeployToken(ctx context.Context, options sourcesecret.Options) (*gitprovider.DeployTokenInfo, error) {
	dts, err := b.repository.DeployTokens()
	if err != nil {
		return nil, err
	}

	b.logger.Actionf("checking to reconcile deploy token for source secret")
	name := deployTokenName(options.Namespace, b.branch, options.Name, options.TargetPath)
	deployTokenInfo := gitprovider.DeployTokenInfo{Name: name}

	deployToken, changed, err := dts.Reconcile(ctx, deployTokenInfo)
	if err != nil {
		return nil, err
	}

	if changed {
		b.logger.Successf("configured deploy token %q for %q", deployTokenInfo.Name, b.repository.Repository().String())
		deployTokenInfo := deployToken.Get()
		return &deployTokenInfo, nil
	}

	b.logger.Successf("reconciled deploy token for source secret")

	return nil, nil
}

// reconcileOrgRepository reconciles a gitprovider.OrgRepository
// with the GitProviderBootstrapper values, including any
// gitprovider.TeamAccessInfo configurations.
//
// If one of the gitprovider.TeamAccessInfo does not reconcile
// successfully, the gitprovider.UserRepository and an
// ErrReconciledWithWarning error are returned.
func (b *GitProviderBootstrapper) reconcileOrgRepository(ctx context.Context) (gitprovider.UserRepository, error) {
	b.logger.Actionf("connecting to %s", b.provider.SupportedDomain())

	// Construct the repository and other configuration objects
	// go-git-provider likes to work with
	subOrgs, repoName := splitSubOrganizationsFromRepositoryName(b.repositoryName)
	orgRef, err := b.getOrganization(ctx, subOrgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create new Git repository %q: %w", b.repositoryName, err)
	}
	repoRef := newOrgRepositoryRef(*orgRef, repoName)
	repoInfo := newRepositoryInfo(b.description, b.defaultBranch, b.visibility)

	// Reconcile the organization repository
	repo, err := b.provider.OrgRepositories().Get(ctx, repoRef)
	if err != nil {
		if !errors.Is(err, gitprovider.ErrNotFound) {
			return nil, fmt.Errorf("failed to get Git repository %q: provider error: %w", repoRef.String(), err)
		}
		// go-git-providers has at present some issues with the idempotency
		// of the available Reconcile methods, and setting e.g. the default
		// branch correctly. Resort to Create until this has been resolved.
		repo, err = b.provider.OrgRepositories().Create(ctx, repoRef, repoInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to create new Git repository %q: %w", repoRef.String(), err)
		}
		b.logger.Successf("repository %q created", repoRef.String())
	}

	var changed bool
	if b.reconcile {
		if err = retry(1, 2*time.Second, func() (err error) {
			changed, err = repo.Reconcile(ctx)
			return
		}); err != nil {
			return nil, fmt.Errorf("failed to reconcile Git repository %q: %w", repoRef.String(), err)
		}
		if changed {
			b.logger.Successf("repository %q reconciled", repoRef.String())
		}
	}

	// Build the team access config
	teamAccessInfo, err := buildTeamAccessInfo(b.teams, gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionMaintain))
	if err != nil {
		return nil, fmt.Errorf("failed to reconcile repository team access: %w", err)
	}

	// Reconcile the team access config on best effort (that being:
	// record the error as a warning, but continue with the
	// reconciliation of the others)
	var warning error
	if count := len(teamAccessInfo); count > 0 {
		b.logger.Actionf("reconciling repository permissions")
		for _, i := range teamAccessInfo {
			var err error
			// Don't reconcile team if team already exists and b.reconcile is false
			if team, err := repo.TeamAccess().Get(ctx, i.Name); err == nil && !b.reconcile && team != nil {
				continue
			}
			_, changed, err = repo.TeamAccess().Reconcile(ctx, i)
			if err != nil {
				warning = fmt.Errorf("failed to grant permissions to team: %w", ErrReconciledWithWarning)
				b.logger.Failuref("failed to grant %q permissions to %q: %s", *i.Permission, i.Name, err.Error())
			} else if changed {
				b.logger.Successf("granted %q permissions to %q", *i.Permission, i.Name)
			}
		}
		b.logger.Successf("reconciled repository permissions")
	}
	return repo, warning
}

// reconcileUserRepository reconciles a gitprovider.UserRepository
// with the GitProviderBootstrapper values. It returns the reconciled
// gitprovider.UserRepository, or an error.
func (b *GitProviderBootstrapper) reconcileUserRepository(ctx context.Context) (gitprovider.UserRepository, error) {
	b.logger.Actionf("connecting to %s", b.provider.SupportedDomain())

	// Construct the repository and other metadata objects
	// go-git-provider likes to work with.
	_, repoName := splitSubOrganizationsFromRepositoryName(b.repositoryName)
	userRef := newUserRef(b.provider.SupportedDomain(), b.owner)
	repoRef := newUserRepositoryRef(userRef, repoName)
	repoInfo := newRepositoryInfo(b.description, b.defaultBranch, b.visibility)

	// Reconcile the user repository
	repo, err := b.provider.UserRepositories().Get(ctx, repoRef)
	if err != nil {
		if !errors.Is(err, gitprovider.ErrNotFound) {
			return nil, fmt.Errorf("failed to get Git repository %q: provider error: %w", repoRef.String(), err)
		}
		// go-git-providers has at present some issues with the idempotency
		// of the available Reconcile methods, and setting e.g. the default
		// branch correctly. Resort to Create until this has been resolved.
		repo, err = b.provider.UserRepositories().Create(ctx, repoRef, repoInfo)
		if err != nil {
			var userErr *gitprovider.ErrIncorrectUser
			if errors.As(err, &userErr) {
				// return a better error message when the wrong owner is set
				err = fmt.Errorf("the specified owner '%s' doesn't match the identity associated with the given token", b.owner)
			}
			return nil, fmt.Errorf("failed to create new Git repository %q: %w", repoRef.String(), err)
		}
		b.logger.Successf("repository %q created", repoRef.String())
	}

	if b.reconcile {
		var changed bool
		if err = retry(1, 2*time.Second, func() (err error) {
			changed, err = repo.Reconcile(ctx)
			return
		}); err != nil {
			return nil, fmt.Errorf("failed to reconcile Git repository %q: %w", repoRef.String(), err)
		}
		if changed {
			b.logger.Successf("repository %q reconciled", repoRef.String())
		}
	}

	return repo, nil
}

// getOrganization retrieves and returns the gitprovider.Organization
// using the GitProviderBootstrapper values.
func (b *GitProviderBootstrapper) getOrganization(ctx context.Context, subOrgs []string) (*gitprovider.OrganizationRef, error) {
	orgRef := newOrganizationRef(b.provider.SupportedDomain(), b.owner, subOrgs)
	// With Stash get the organization to be sure to get the correct key
	if string(b.provider.ProviderID()) == string(provider.GitProviderStash) {
		org, err := b.provider.Organizations().Get(ctx, orgRef)
		if err != nil {
			return nil, fmt.Errorf("failed to get Git organization: %w", err)
		}

		orgRef = org.Organization()

		return &orgRef, nil
	}
	return &orgRef, nil
}

// getCloneURL returns the Git clone URL for the given
// gitprovider.UserRepository. If the given transport type is
// gitprovider.TransportTypeSSH and a custom SSH hostname is configured,
// the hostname of the URL will be modified to this hostname.
func (b *GitProviderBootstrapper) getCloneURL(repository gitprovider.UserRepository, transport gitprovider.TransportType) (string, error) {
	var url string
	if cloner, ok := repository.(gitprovider.CloneableURL); ok {
		url = cloner.GetCloneURL("", transport)
	} else {
		url = repository.Repository().GetCloneURL(transport)
	}

	var err error
	if transport == gitprovider.TransportTypeSSH && b.sshHostname != "" {
		if url, err = setHostname(url, b.sshHostname); err != nil {
			err = fmt.Errorf("failed to set SSH hostname for URL %q: %w", url, err)
		}
	}
	return url, err
}

// splitSubOrganizationsFromRepositoryName removes any prefixed sub
// organizations from the given repository name by splitting the
// string into a slice by '/'.
// The last (or only) item of the slice result is assumed to be the
// repository name, other items (nested) sub organizations.
func splitSubOrganizationsFromRepositoryName(name string) ([]string, string) {
	elements := strings.Split(name, "/")
	i := len(elements)
	switch i {
	case 1:
		return nil, name
	default:
		return elements[:i-1], elements[i-1]
	}
}

// buildTeamAccessInfo constructs a gitprovider.TeamAccessInfo slice
// from the given string map of team names to permissions.
//
// Providing a default gitprovider.RepositoryPermission is optional,
// and omitting it will make it default to the go-git-provider default.
//
// An error is returned if any of the given permissions is invalid.
func buildTeamAccessInfo(m map[string]string, defaultPermissions *gitprovider.RepositoryPermission) ([]gitprovider.TeamAccessInfo, error) {
	var infos []gitprovider.TeamAccessInfo
	if defaultPermissions != nil {
		if err := gitprovider.ValidateRepositoryPermission(*defaultPermissions); err != nil {
			return nil, fmt.Errorf("invalid default team permission %q", *defaultPermissions)
		}
	}
	for n, p := range m {
		permission := defaultPermissions
		if p != "" {
			p := gitprovider.RepositoryPermission(p)
			if err := gitprovider.ValidateRepositoryPermission(p); err != nil {
				return nil, fmt.Errorf("invalid permission %q for team %q", p, n)
			}
			permission = &p
		}
		i := gitprovider.TeamAccessInfo{
			Name:       n,
			Permission: permission,
		}
		infos = append(infos, i)
	}
	return infos, nil
}

// newOrganizationRef constructs a gitprovider.OrganizationRef with the
// given values and returns the result.
func newOrganizationRef(domain, organization string, subOrganizations []string) gitprovider.OrganizationRef {
	return gitprovider.OrganizationRef{
		Domain:           domain,
		Organization:     organization,
		SubOrganizations: subOrganizations,
	}
}

// newOrgRepositoryRef constructs a gitprovider.OrgRepositoryRef with
// the given values and returns the result.
func newOrgRepositoryRef(organizationRef gitprovider.OrganizationRef, name string) gitprovider.OrgRepositoryRef {
	return gitprovider.OrgRepositoryRef{
		OrganizationRef: organizationRef,
		RepositoryName:  name,
	}
}

// newUserRef constructs a gitprovider.UserRef with the given values
// and returns the result.
func newUserRef(domain, login string) gitprovider.UserRef {
	return gitprovider.UserRef{
		Domain:    domain,
		UserLogin: login,
	}
}

// newUserRepositoryRef constructs a gitprovider.UserRepositoryRef with
// the given values and returns the result.
func newUserRepositoryRef(userRef gitprovider.UserRef, name string) gitprovider.UserRepositoryRef {
	return gitprovider.UserRepositoryRef{
		UserRef:        userRef,
		RepositoryName: name,
	}
}

// newRepositoryInfo constructs a gitprovider.RepositoryInfo with the
// given values and returns the result.
func newRepositoryInfo(description, defaultBranch, visibility string) gitprovider.RepositoryInfo {
	var i gitprovider.RepositoryInfo
	if description != "" {
		i.Description = gitprovider.StringVar(description)
	}
	if defaultBranch != "" {
		i.DefaultBranch = gitprovider.StringVar(defaultBranch)
	}
	if visibility != "" {
		i.Visibility = gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibility(visibility))
	}
	return i
}

// newDeployKeyInfo constructs a gitprovider.DeployKeyInfo with the
// given values and returns the result.
func newDeployKeyInfo(name, publicKey string, readWrite bool) gitprovider.DeployKeyInfo {
	keyInfo := gitprovider.DeployKeyInfo{
		Name: name,
		Key:  []byte(publicKey),
	}
	if readWrite {
		keyInfo.ReadOnly = gitprovider.BoolVar(false)
	}
	return keyInfo
}

func deployKeyName(namespace, secretName, branch, path string) string {
	var name string
	for _, v := range []string{namespace, secretName, branch, path} {
		if v == "" {
			continue
		}
		if name == "" {
			name = v
		} else {
			name = name + "-" + v
		}
	}
	return name
}

func deployTokenName(namespace, secretName, branch, path string) string {
	var elems []string
	for _, v := range []string{namespace, secretName, branch, path} {
		if v == "" {
			continue
		}
		elems = append(elems, v)
	}
	return strings.Join(elems, "-")
}

// setHostname is a helper to replace the hostname of the given URL.
// TODO(hidde): support for this should be added in go-git-providers.
func setHostname(URL, hostname string) (string, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return URL, err
	}
	u.Host = hostname
	return u.String(), nil
}
