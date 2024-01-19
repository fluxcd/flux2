# Flux GitHub Workflows

## End-to-end Testing

The e2e workflows run a series of tests to ensure that the Flux CLI and
the GitOps Toolkit controllers work well all together.
The tests are written in Go, Bash, Make and Terraform.

| Workflow           | Jobs                 | Runner         | Role                                          |
|--------------------|----------------------|----------------|-----------------------------------------------|
| e2e.yaml           | e2e-amd64-kubernetes | GitHub Ubuntu  | integration testing with Kubernetes Kind<br/> |
| e2e-arm64.yaml     | e2e-arm64-kubernetes | Equinix Ubuntu | integration testing with Kubernetes Kind<br/> |
| e2e-bootstrap.yaml | e2e-boostrap-github  | GitHub Ubuntu  | integration testing with GitHub API<br/>      |
| e2e-azure.yaml     | e2e-amd64-aks        | GitHub Ubuntu  | integration testing with Azure API<br/>       |
| scan.yaml          | scan-fossa           | GitHub Ubuntu  | license scanning<br/>                         |
| scan.yaml          | scan-snyk            | GitHub Ubuntu  | vulnerability scanning<br/>                   |
| scan.yaml          | scan-codeql          | GitHub Ubuntu  | vulnerability scanning<br/>                   |

## Components Update

The components update workflow scans the GitOps Toolkit controller repositories for new releases,
amd when it finds a new controller version, the workflow performs the following steps:
- Updates the controller API package version in `go.mod`.
- Patches the controller CRDs version in the `manifests/crds` overlay.
- Patches the controller Deployment version in `manifests/bases` overlay.
- Opens a Pull Request against the `main` branch.
- Triggers the e2e test suite to run for the opened PR.


| Workflow    | Jobs              | Runner        | Role                                                |
|-------------|-------------------|---------------|-----------------------------------------------------|
| update.yaml | update-components | GitHub Ubuntu | update the GitOps Toolkit APIs and controllers<br/> |

## Release

The release workflow is triggered by a semver Git tag and performs the following steps:
- Generates the Flux install manifests (YAML).
- Generates the OpenAPI validation schemas for the GitOps Toolkit CRDs (JSON).
- Generates a Software Bill of Materials (SPDX JSON).
- Builds the Flux CLI binaries and the multi-arch container images.
- Pushes the container images to GitHub Container Registry and DockerHub.
- Signs the sbom, the binaries checksum and the container images with Cosign and GitHub OIDC.
- Uploads the sbom, binaries, checksums and install manifests to GitHub Releases.
- Pushes the install manifests as OCI artifacts to GitHub Container Registry and DockerHub.
- Signs the OCI artifacts with Cosign and GitHub OIDC.

| Workflow     | Jobs                   | Runner        | Role                                                 |
|--------------|------------------------|---------------|------------------------------------------------------|
| release.yaml | release-flux-cli       | GitHub Ubuntu | build, push and sign the CLI release artifacts<br/>  |
| release.yaml | release-flux-manifests | GitHub Ubuntu | build, push and sign the Flux install manifests<br/> |
