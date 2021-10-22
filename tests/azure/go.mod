module github.com/fluxcd/flux2/tests/azure

go 1.16

require (
	github.com/Azure/azure-event-hubs-go/v3 v3.3.13
	github.com/fluxcd/helm-controller/api v0.12.1
	github.com/fluxcd/image-automation-controller/api v0.15.0
	github.com/fluxcd/image-reflector-controller/api v0.13.0
	github.com/fluxcd/kustomize-controller/api v0.16.0
	github.com/fluxcd/notification-controller/api v0.18.1
	github.com/fluxcd/pkg/apis/meta v0.10.1
	github.com/fluxcd/pkg/runtime v0.12.1
	github.com/fluxcd/source-controller/api v0.16.1
	github.com/hashicorp/terraform-exec v0.14.0
	github.com/libgit2/git2go/v31 v31.6.1
	github.com/microsoft/azure-devops-go-api/azuredevops v1.0.0-b5
	github.com/stretchr/testify v1.7.0
	github.com/whilp/git-urls v1.0.0
	go.uber.org/multierr v1.6.0
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	sigs.k8s.io/controller-runtime v0.10.1
)
