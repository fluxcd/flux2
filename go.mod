module github.com/fluxcd/flux2

go 1.15

require (
	github.com/blang/semver/v4 v4.0.0
	github.com/cyphar/filepath-securejoin v0.2.2
	github.com/fluxcd/helm-controller/api v0.5.2
	github.com/fluxcd/image-automation-controller/api v0.3.1
	github.com/fluxcd/image-reflector-controller/api v0.3.0
	github.com/fluxcd/kustomize-controller/api v0.6.2
	github.com/fluxcd/notification-controller/api v0.6.2
	github.com/fluxcd/pkg/apis/meta v0.6.0
	github.com/fluxcd/pkg/git v0.2.2
	github.com/fluxcd/pkg/runtime v0.6.2
	github.com/fluxcd/pkg/ssh v0.0.5
	github.com/fluxcd/pkg/untar v0.0.5
	github.com/fluxcd/source-controller/api v0.6.3
	github.com/google/go-containerregistry v0.2.0
	github.com/manifoldco/promptui v0.7.0
	github.com/olekukonko/tablewriter v0.0.4
	github.com/spf13/cobra v1.1.1
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.0
	sigs.k8s.io/kustomize/api v0.7.0
	sigs.k8s.io/yaml v1.2.0
)
