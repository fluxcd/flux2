# RFC-0006 Flux OCI Artifact Support for Helm and Kubernetes Manifests

**Status:** provisional

**Creation date:** 2022-03-24

**Last update:** 2022-03-24

## Summary

We want to make it possible to reconcile workload from OCI Registries, the same we can reconcile today from git or Helm repositories. The targets are Helm charts and kubernetes manifests.

## Motivation

Registries are evolving into generic artifact stores. Many of our users would like to take advantage of this fact in order to store their kubernetes declaration and Helm charts in registries and apply them using Flux. Helm already supports oci registries to store Charts.

This [request](https://github.com/fluxcd/source-controller/issues/124) has been made to the Flux team to add support for Helm charts in OCI registries.

### Goals

- Introduce a new source that can reconcile 2 OCI artifact types from OCI compliant registries, Helm artifact and OCI image.
- This new source must respect the existing source contract, i.e. it must produce the same outputs as the existing sources. this will enable existing consumers to use the new source with minimal effort.
- Pave the way for eventually supporting other OCI artifact types.

### Non-Goals

- Introduce a new OCI artifact type that can be used to store kubernetes manifests in OCI registries.

## Proposal

Introduce 2 new Flux resources:
  - `OCIRegistry`, which holds the registry information, i.e. URL and credentials.
  - `OCIArtifact`, which holds the artifact information, mainly a reference to the desired tag, semantic version or digest.

An `OCIArtifact` can be used to reference a Helm Chart published with a Helm artifact media type or any tar package published with an OCI Image media type.
For any other artifact type, we will return an event stating that the OCI artifact type is not supported. 

The OCI artifact reconciler will retrieve an artifact based on the url and credentials provided by the reference `OCIRegistry`, and store the artifact locally for dependent controllers to consume.

The Helm Chart reconciler will retrieve an artifact based on the url and credentials provided by the reference `OCIRegistry`, build the chart, and store the artifact locally for dependent controllers to consume.


### User Stories

#### Story 1

As a developper I want to push my kubernetes manifests as a tar archive to an OCI Registry, and then use Flux to reconcile it in my cluster.

#### Story 2

As a developper I want to push my Helm charts using the Helm cli to an OCI Registry, and then use Flux to reconcile it in my cluster.


### Alternatives

We could keep the `OCIRepository` proposal and tie `OCIArtifact` and `OCIRegistry` specs. That is considered unpractical, as we would like to be able to reuse a registry spec for different artifact types with different reconcilers.

Instead of reusing the `oci image` mediaType to adress the need to store Kubernetes manifests declaration, we could register our own artifact type. This is also considered un practical, as declaring a type has to go through the IANA process and Flux is not the owner of those type as Helm is for Helm artifact for example. 

## Design Details

 The new APIs:

```go
type OCIRegistrySpec struct {
	// URL is a reference to a namespace in a remote registry
	// .e.g. oci://registry.example.com/namespace/
	// +required
	URL string `json:"url"`

	// The credentials to use to pull and monitor for changes, defaults
	// to anonymous access.
	// +optional
	Authentication *OCIRepositoryAuth `json:"auth,omitempty"`
}

type OCIRepositoryAuth struct {
	// SecretRef contains the secret name containing the registry login
	// credentials to resolve image metadata.
	// The secret must be of type kubernetes.io/dockerconfigjson.
	// +optional
	SecretRef *meta.LocalObjectReference `json:"secretRef,omitempty"`

	// ServiceAccountName is the name of the Kubernetes ServiceAccount used to authenticate
	// the image pull if the service account has attached pull secrets. For more information:
	// https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#add-imagepullsecrets-to-a-service-account
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

type OCIArtifactSpec struct {
	// Name is the name of the artifact to pull.
	// +required
	Name string `json:"name"`

  // Pull artifacts using this provider.
	// +required
	OCIRegistryRef meta.LocalObjectReference `json:"ociRegistryRef"`
    
	// The OCI reference to pull and monitor for changes, defaults to
	// latest tag.
	// +optional
	Reference *OCIRepositoryRef `json:"ref,omitempty"`

	// The interval at which to check for image updates.
	// +required
	Interval metav1.Duration `json:"interval"`

	// The timeout for remote OCI Repository operations like pulling, defaults to 20s.
	// +kubebuilder:default="20s"
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Ignore overrides the set of excluded patterns in the .sourceignore format
	// (which is the same as .gitignore). If not provided, a default will be used,
	// consult the documentation for your version to find out what those are.
	// +optional
	Ignore *string `json:"ignore,omitempty"`

	// This flag tells the controller to suspend the reconciliation of this source.
	// +optional
	Suspend bool `json:"suspend,omitempty"`
}

type OCIRepositoryRef struct {
	// Digest is the image digest to pull, takes precedence over SemVer.
	// Value should be in the form sha256:cbbf2f9a99b47fc460d422812b6a5adff7dfee951d8fa2e4a98caa0382cfbdbf
	// +optional
	Digest string `json:"digest,omitempty"`

	// SemVer is the range of tags to pull selecting the latest within
	// the range, takes precedence over Tag.
	// +optional
	SemVer string `json:"semver,omitempty"`

	// Tag is the image tag to pull, defaults to latest.
	// +kubebuilder:default:=latest
	// +optional
	Tag string `json:"tag,omitempty"`
}
```

An `OCIArtifact` declaration would be:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: OCIRegistry
metadata:
  name: ghcr-podinfo
  namespace: flux-system
spec:
  url: ghcr.io/stefanprodan
  # one of secretRef or serviceAccountName
  auth:
    secretRef:
      name: regcred
    serviceAccountName: reg

---
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: OCIArtifact
metadata:
  name: podinfo-deploy
  namespace: flux-system
spec:
	name: podinfo-deploy
  ociRegistryRef:
    name: ghcr-podinfo
  interval: 10m
  timeout: 1m
  url: ghcr.io/stefanprodan/podinfo-deploy
  ref:
    # one of
    tag: "latest"
    digest: "sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2"
    semver: "1.x"
  serviceAccountName: reg
  ignore: ".git, .out"
  suspend: true
```

for HelmChart, we would have something like:
```yaml
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: podinfo-deploy
spec:
  interval: 5m
  chart:
    spec:
      chart: podinfo-deploy
      version: '4.0.x'
      sourceRef:
        kind: OCIRegistry
        name: ghcr-podinfo
      interval: 1m
```

We will introduce our own client based on the `oras` library. This will allow us to decide the supported artifact types and the way to retrieve them.

```go
	memoryStore := content.NewMemory()

  // define the allowed media types
  // for our proposal they will be:
  // "application/vnd.oci.image.config.v1+json"
  // "application/vnd.oci.image.manifest.v1+json"
  // "application/vnd.oci.image.layer.v1.tar+gzip"
  // "application/vnd.cncf.helm.config.v1+json"
  // "application/vnd.cncf.helm.chart.content.v1.tar+gzip"
  // "application/vnd.cncf.helm.chart.provenance.v1.prov"
  // "application/vnd.docker.distribution.manifest.v2+json"
  // "application/vnd.docker.image.rootfs.diff.tar.gzip"
  // "application/vnd.docker.container.image.v1+json"
	allowedMediaTypes := []string{
		ocispec.MediaTypeImageConfig,
		ocispec.MediaTypeImageLayerGzip,
		helmTypes.ConfigMediaType,
		helmTypes.ChartLayerMediaType,
		helmTypes.ProvLayerMediaType,
		distributionSchema.MediaTypeManifest,
		distributionSchema.MediaTypeLayer,
		distributionSchema.MediaTypeImageConfig,
	}

	var layers []ocispec.Descriptor
	registryStore := content.Registry{Resolver: c.resolver}

	manifest, err := oras.Copy(ctx(c.out, c.debug), registryStore, parsedRef.String(), memoryStore, "",
		oras.WithPullEmptyNameAllowed(),
		oras.WithAllowedMediaTypes(allowedMediaTypes),
		oras.WithAdditionalCachedMediaTypes(distributionSchema.MediaTypeManifest),
		oras.WithLayerDescriptors(func(l []ocispec.Descriptor) {
			layers = l
		}))
	if err != nil {
		return nil, fmt.Errorf("pulling %s failed: %s", parsedRef.String(), err)
	}
```

### Multi-tenancy considerations

`OCIArtifact` and `Helmchart` reference `OCIRegistry` as `local` reference, i.e. in the same namespace. 

But the Kustomize and Helm controllers are able to reference cross-namespaced resources.

To support multi-tenancy while allowing cross-namespace reference from Kustomize and Helm controllers, we will implement `rfc-0002` and add an `acl` field to `OCIRegistry` and `OCIArtifact`.

```go
type OCIRegistrySpec struct {
	// URL is a reference to a namespace in a remote registry
	// +required
	URL string `json:"url"`

	// The credentials to use to pull and monitor for changes, defaults
	// to anonymous access.
	// +optional
	Authentication *OCIRepositoryAuth `json:"auth,omitempty"`

  // +optional
	AccessFrom *acl.AccessFrom `json:"accessFrom,omitempty"`
}

type OCIArtifactSpec struct {
	// Name is the name of the artifact to pull.
	// +required
	Name string `json:"name"`

  // Pull artifacts using this provider.
	// +required
	OCIRegistryRef meta.LocalObjectReference `json:"ociRegistryRef"`
    
	// The OCI reference to pull and monitor for changes, defaults to
	// latest tag.
	// +optional
	Reference *OCIRepositoryRef `json:"ref,omitempty"`

	// The interval at which to check for image updates.
	// +required
	Interval metav1.Duration `json:"interval"`

	// The timeout for remote OCI Repository operations like pulling, defaults to 20s.
	// +kubebuilder:default="20s"
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Ignore overrides the set of excluded patterns in the .sourceignore format
	// (which is the same as .gitignore). If not provided, a default will be used,
	// consult the documentation for your version to find out what those are.
	// +optional
	Ignore *string `json:"ignore,omitempty"`

	// This flag tells the controller to suspend the reconciliation of this source.
	// +optional
	Suspend bool `json:"suspend,omitempty"`

  // +optional
	AccessFrom *acl.AccessFrom `json:"accessFrom,omitempty"`
}
```

### Dealing with dependencies

`The OCI Registry` client pulls all artifacts layers for supported types.

The `OCIArtifact` and `HelmChart` reconciler will expect `data` to be in the first layer of the artifact, and will discard all the other layers. The controllers will then resolve the artifact dependencies, as it is a domain specific problem.

For example, if we have a Helm artifact, the dependent Helm Chart controller will pull the artifact, extract the Chart and then resolve the chart dependencies.

### Enabling the feature

The feature is enabled by default. You can disable it by setting the `enabe-oci-registry` flag to false. Which will prevent registering the oci reconcilers in the controller manager. For Helm, the flag is checked in the `HelmChart` reconciler.

The Helm chart reconciler will have a new source for downloading the charts, but we will keep using the `remoteChartBuilder` to build the charts. The `remoteChartBuilder` will be modified to accept a `Remote` interface, in order to support the OCI Registry.

## Follow-up work

There will be a specific Flux command (built on top of ORAS) to help users push to their preferred OCI Registry

```shell
⋊> ~ flux push localhost:5000/podinfo:v1 -path ./podinfo/kustomize --username souleb
```

The artifact will be of type `application/vnd.oci.image.config.v1+json`. The directory pointed to by `path` will be archived and compressed to `tar+gzip`. The layer media type will be `application/vnd.oci.image.layer.v1.tar+gzip`.

This will ease adoption of this new feature.

## Implementation History

<!--
Major milestones in the lifecycle of the RFC such as:
- The first Flux release where an initial version of the RFC was available.
- The version of Flux where the RFC graduated to general availability.
- The version of Flux where the RFC was retired or superseded.
-->
