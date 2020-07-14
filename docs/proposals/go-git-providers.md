# go-git-providers

## Abstract

This proposal aims to create a library with the import path `github.com/fluxcd/go-git-providers`'
(import name: `gitprovider`), which provides an abstraction layer for talking to Git providers
like GitHub, GitLab and Bitbucket.

This would become a new repository, specifically targeted at being a general-purpose Git provider
client for multiple providers and domains.

## Goals

- Support multiple Git provider backends (e.g. GitHub, GitLab, Bitbucket, etc.) using the same interface
- Support talking to multiple domains at once, including custom domains (e.g. talking to "gitlab.com" and "version.aalto.fi" from the same client)
- Support both no authentication (for public repos), basic auth, and OAuth2 for authentication
- Manipulating the following resources:
  - **Organizations**: `GET`, `LIST` (both all accessible top-level orgs and sub-orgs)
  - For a given **Organization**:
    - **Teams**: `GET` and `LIST`
  - **Repositories**: `GET`, `LIST` and `POST`
    - **Teams**: `ADD` and `REMOVE`
    - **Credentials**: `GET`, `LIST` and `POST`
- Support sub-organizations (or "sub-groups" in GitLab) if possible
- Pagination is automatically handled for `LIST` requests
- Transparently can manage teams (collections of users, sub-groups in Gitlab) with varying access to repos
- Follow library best practices in order to be easy to vendor (e.g. use major `vX` versioning & go.mod)

## Non-goals

- Support for features not mentioned above

## Design decisions

- A `context.Context` should be passed to every request as the first argument
- There should be two interfaces per resource, if applicable:
  - one collection-specific interface, with a plural name (e.g. `OrganizationsClient`), that has methods like `Get()` and `List()`
  - one instance-specific interface, with a singular name (e.g. `OrganizationClient`), that operates on that instance, e.g. allowing access to child resources, e.g. `Teams()`
- Every `Create()` signature shall have a `{Resource}CreateOptions` struct as the last argument.
  - `Delete()` and similar methods may use the same pattern if needed
- All `*Options` structs shall be passed by value (i.e. non-nillable) and contain only nillable, optional fields
- All optional fields in the type structs shall be nillable
- It should be possible to create a fake API client for testing, implementing the same interfaces
- All type structs shall have a `Validate()` method, and optionally a `Default()` one
- All type structs shall expose their internal representation (from the underlying library) through the `InternalGetter` interface with a method `GetInternal() interface{}`
- By default, mutating methods (e.g. `Create()`) shall be idempotent, unless opted out in `*CreateOptions` field.
- Typed errors shall be returned, wrapped using Go 1.14's new features
- Go-style enums are used when there are only a few supported values for a field
- Every field is documented using Godoc comment, including `+required` or `+optional` to clearly signify its importance
- Support serializing the types to JSON (if needed for e.g. debugging) by adding tags

## Implementation

### Provider package

The provider package, e.g. at `github.com/fluxcd/go-git-providers/github`, will have constructor methods so a client can be created, e.g. as follows:

```go
// Create a client for github.com without any authentication
c := github.NewClient()

// Create a client for an enterprise GitHub account, without any authentication
c = github.NewClient(github.WithBaseURL("enterprise.github.com"))

// Create a client for github.com using a personal oauth2 token
c = github.NewClient(github.WithOAuth2("<token-here>"))
```

### Client

The definition of a `Client` is as follows:

```go
// Client is an interface that allows talking to a Git provider
type Client interface {
	// The Client allows accessing all known resources
	ResourceClient

	// SupportedDomain returns the supported domain
	// This field is set at client creation time, and can't be changed
	SupportedDomain() string

	// ProviderID returns the provider ID (e.g. "github", "gitlab") for this client
	// This field is set at client creation time, and can't be changed
	ProviderID() ProviderID

	// Raw returns the Go client used under the hood for accessing the Git provider
	Raw() interface{}
}
```

As one can see, the `Client` is scoped for a single backing domain. `ProviderID` is a typed string, and every
implementation package defines their own constant, e.g. `const ProviderName = gitprovider.ProviderID("github")`.

The `ResourceClient` actually allows talking to resources of the API, both for single objects, and collections:

```go
// ResourceClient allows access to resource-specific clients
type ResourceClient interface {
	// Organization gets the OrganizationClient for the specific top-level organization
	// ErrNotTopLevelOrganization will be returned if the organization is not top-level when using
	Organization(o OrganizationRef) OrganizationClient

	// Organizations returns the OrganizationsClient handling sets of organizations
	Organizations() OrganizationsClient

	// Repository gets the RepositoryClient for the specified RepositoryRef
	Repository(r RepositoryRef) RepositoryClient

	// Repositories returns the RepositoriesClient handling sets of organizations
	Repositories() RepositoriesClient
}
```

In order to reference organizations and repositories, there are the `OrganizationRef` and `RepositoryRef`
interfaces:

```go
// OrganizationRef references an organization in a Git provider
type OrganizationRef interface {
	// String returns the HTTPS URL
	fmt.Stringer

	// GetDomain returns the URL-domain for the Git provider backend, e.g. gitlab.com or version.aalto.fi
	GetDomain() string
	// GetOrganization returns the top-level organization, i.e. "weaveworks" or "kubernetes-sigs"
	GetOrganization() string
	// GetSubOrganizations returns the names of sub-organizations (or sub-groups),
	// e.g. ["engineering", "frontend"] would be returned for gitlab.com/weaveworks/engineering/frontend
	GetSubOrganizations() []string
}

// RepositoryRef references a repository hosted by a Git provider
type RepositoryRef interface {
	// RepositoryRef requires an OrganizationRef to fully-qualify a repo reference
	OrganizationRef

	// GetRepository returns the name of the repository
	GetRepository() string
}
```

Along with these, there is `OrganizationInfo` and `RepositoryInfo` which implement the above mentioned interfaces in a straightforward way.

If you want to create an `OrganizationRef` or `RepositoryRef`, you can either use `NewOrganizationInfo()` or `NewRepositoryInfo()`, filling in all parts of the reference, or use the `ParseRepositoryURL(r string) (RepositoryRef, error)` or `ParseOrganizationURL(o string) (OrganizationRef, error)` methods.

As mentioned above, only one target domain is supported by the `Client`. This means e.g. that if the `Client` is configured for GitHub, and you feed it a GitLab URL to parse, `ErrDomainUnsupported` will be returned.

This brings us to a higher-level client abstraction, `MultiClient`.

### MultiClient

In order to automatically support multiple domains and providers using the same interface, `MultiClient` is introduced.

The user would use the `MultiClient` as follows:

```go
// Create a client to github.com without authentication
gh := github.NewClient()

// Create a client to gitlab.com, authenticating with basic auth
gl := gitlab.NewClient(gitlab.WithBasicAuth("<username>", "<password"))

// Create a client to the GitLab instance at version.aalto.fi, with a given OAuth2 token
aalto := gitlab.NewClient(gitlab.WithBaseURL("version.aalto.fi"), gitlab.WithOAuth2Token("<your-token>"))

// Create a MultiClient which supports talking to any of these backends
client := gitprovider.NewMultiClient(gh, gl, aalto)
```

The interface definition of `MultiClient` is similar to that one of `Client`, both embedding `ResourceClient`, but it also allows access to domain-specific underlying `Client`'s:

```go
// MultiClient allows talking to multiple Git providers at once
type MultiClient interface {
	// The MultiClient allows accessing all known resources, automatically choosing the right underlying
	// Client based on the resource's domain
	ResourceClient

	// SupportedDomains returns a list of known domains
	SupportedDomains() []string

	// ClientForDomain returns the Client used for a specific domain
	ClientForDomain(domain string) (Client, bool)
}
```

### OrganizationsClient

The `OrganizationsClient` provides access to a set of organizations, as follows:

```go
// OrganizationsClient operates on organizations the user has access to
type OrganizationsClient interface {
	// Get a specific organization the user has access to
	// This might also refer to a sub-organization
	Get(ctx context.Context, o OrganizationRef) (*Organization, error)

	// List all top-level organizations the specific user has access to
	// List should return all available organizations, using multiple paginated requests if needed
	List(ctx context.Context) ([]Organization, error)

	// Children returns the immediate child-organizations for the specific OrganizationRef o.
	// The OrganizationRef may point to any sub-organization that exists
	// This is not supported in GitHub
	// Children should return all available organizations, using multiple paginated requests if needed
	Children(ctx context.Context, o OrganizationRef) ([]Organization, error)

	// Possibly add Create/Update/Delete methods later
}
```

The `Organization` struct is fairly straightforward for now:

```go
// Organization represents an (top-level- or sub-) organization
type Organization struct {
	// OrganizationInfo provides the required fields
	// (Domain, Organization and SubOrganizations) required for being an OrganizationRef
	OrganizationInfo `json:",inline"`
	// InternalHolder implements the InternalGetter interface
	InternalHolder `json:",inline"`

	// Name is the human-friendly name of this organization, e.g. "Weaveworks" or "Kubernetes SIGs"
	// +required
	Name string `json:"name"`

	// Description returns a description for the organization
	// No default value at POST-time
	// +optional
	Description *string `json:"description"`
}
```

The `OrganizationInfo` struct is a straightforward struct just implementing the `OrganizationRef` interface
with basic fields & getters. `InternalHolder` is implementing the `InternalGetter` interface as follows, and is
embedded into all main structs:

```go
// InternalGetter allows access to the underlying object
type InternalGetter interface {
	// GetInternal returns the underlying struct that's used
	GetInternal() interface{}
}

// InternalHolder can be embedded into other structs to implement the InternalGetter interface
type InternalHolder struct {
	// Internal contains the underlying object.
	// +optional
	Internal interface{} `json:"-"`
}
```

### OrganizationClient

`OrganizationClient` allows access to a specific organization's underlying resources as follows:

```go
// OrganizationClient operates on a given/specific organization
type OrganizationClient interface {
	// Teams gives access to the TeamsClient for this specific organization
	Teams() OrganizationTeamsClient
}
```

#### Organization Teams

Teams belonging to a certain organization can at this moment be fetched on an individual basis, or listed.

```go
// OrganizationTeamsClient handles teams organization-wide
type OrganizationTeamsClient interface {
	// Get a team within the specific organization
	// teamName may include slashes, to point to e.g. "sub-teams" i.e. subgroups in Gitlab
	// teamName must not be an empty string
	Get(ctx context.Context, teamName string) (*Team, error)

	// List all teams (recursively, in terms of subgroups) within the specific organization
	// List should return all available organizations, using multiple paginated requests if needed
	List(ctx context.Context) ([]Team, error)

	// Possibly add Create/Update/Delete methods later
}
```

The `Team` struct is defined as follows:

```go
// Team is a representation for a team of users inside of an organization
type Team struct {
	// Team embeds OrganizationInfo which makes it automatically
	OrganizationRef `json:",inline"`
	// Team embeds InternalHolder for accessing the underlying object
	InternalHolder `json:",inline"`

	// Name describes the name of the team. The team name may contain slashes
	// +required
	Name string `json:"name"`

	// Members points to a set of user names (logins) of the members of this team
	// +required
	Members []string `json:"members"`
}
```

In GitLab, teams could be modelled as users in a sub-group. Those users can later be added as a single unit
to access a given repository.

### RepositoriesClient

`RepositoriesClient` provides access to a set of repositories for the user.

```go
// RepositoriesClient operates on repositories the user has access to
type RepositoriesClient interface {
	// Get returns the repository at the given path
	Get(ctx context.Context, r RepositoryRef) (*Repository, error)

	// List all repositories in the given organization
	// List should return all available organizations, using multiple paginated requests if needed
	List(ctx context.Context, o OrganizationRef) ([]Repository, error)

	// Create creates a repository at the given organization path, with the given URL-encoded name and options
	Create(ctx context.Context, r *Repository, opts RepositoryCreateOptions) (*Repository, error)
}
```

`RepositoryCreateOptions` has options like `AutoInit *bool`, `LicenseTemplate *string` and so forth to allow an
one-time initialization step.

The `Repository` struct is defined as follows:

```go
// Repository represents a Git repository provided by a Git provider
type Repository struct {
	// RepositoryInfo provides the required fields
	// (Domain, Organization, SubOrganizations and (Repository)Name)
	// required for being an RepositoryRef
	RepositoryInfo `json:",inline"`
	// InternalHolder implements the InternalGetter interface
	InternalHolder `json:",inline"`

	// Description returns a description for the repository
	// No default value at POST-time
	// +optional
	Description *string `json:"description"`

	// Visibility returns the desired visibility for the repository
	// Default value at POST-time: RepoVisibilityPrivate
	// +optional
	Visibility *RepoVisibility
}

// GetCloneURL gets the clone URL for the specified transport type
func (r *Repository) GetCloneURL(transport TransportType) string {
	return GetCloneURL(r, transport)
}
```

As can be seen, there is also a `GetCloneURL` function for the repository which allows
resolving the URL from which to clone the repo, for a given transport method (`ssh` and `https`
are supported `TransportType`s)

### RepositoryClient

`RepositoryClient` allows access to a given repository's underlying resources, like follows:

```go
// RepositoryClient operates on a given/specific repository
type RepositoryClient interface {
	// TeamAccess gives access to what teams have access to this specific repository
	TeamAccess() RepositoryTeamAccessClient

	// Credentials gives access to manipulating credentials for accessing this specific repository
	Credentials() RepositoryCredentialsClient
}
```

#### Repository Teams

`RepositoryTeamAccessClient` allows adding & removing teams from the list of authorized persons to access a repository.

```go
// RepositoryTeamAccessClient operates on the teams list for a specific repository
type RepositoryTeamAccessClient interface {
	// Add adds a given team in the repo's (top-level) organization to the repository
	Add(ctx context.Context, teamName string, opts RepositoryAddTeamOptions) error

	// Remove removes the given team in the repo's (top-level) organization from the repository
	Remove(ctx context.Context, teamName string) error
}
```

`RepositoryAddTeamOptions` has a `Permission *TeamRepositoryPermission` which allows specifying what access level the team should have.

#### Repository Credentials

`RepositoryCredentialsClient` allows adding & removing credentials (e.g. deploy keys) from accessing a specific repository.

```go
// RepositoryCredentialsClient operates on the access credential list for a specific repository
type RepositoryCredentialsClient interface {
	// Create a credential with the given human-readable name, the given bytes and optional options
	Create(ctx context.Context, c RepositoryCredential, opts CredentialCreateOptions) error

	// Lists all credentials
	List(ctx context.Context) ([]RepositoryCredential, error)

	// Deletes a credential from the repo. keyName is the human-friendly title of the credential
	Delete(ctx context.Context, keyName string) error
}
```

In order to support multiple different types of credentials, `RepositoryCredential` is an interface:

```go
// RepositoryCredential is a credential that allows
type RepositoryCredential interface {
	// GetType returns the type of the credential
	GetType() RepositoryCredentialType

	// Title returns a description of the credential
	GetTitle() string

	// GetData returns the key that will be authorized to access the repo, this can e.g. be a SSH public key
	GetData() []byte

	// IsReadOnly returns whether this credential is authorized to write to the repository or not
	IsReadOnly() bool
}
```

The default implementation of `RepositoryCredential` is `DeployKey`:

```go
// DeployKey represents a short-lived credential (e.g. an SSH public key) used for accessing a repository
type DeployKey struct {
	// DeployKey embeds InternalHolder for accessing the underlying object
	InternalHolder `json:",inline"`

	// Title is the human-friendly interpretation of what the key is for (and does)
	// +required
	Title string `json:"title"`

	// Key specifies the public part of the deploy (e.g. SSH) key
	// +required
	Key []byte `json:"key"`

	// ReadOnly specifies whether this DeployKey can write to the repository or not
	// Default value at POST-time: true
	// +optional
	ReadOnly *bool `json:"readOnly"`
}
```
