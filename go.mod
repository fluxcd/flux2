module github.com/fluxcd/toolkit

go 1.14

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/fluxcd/kustomize-controller v0.0.1-beta.1
	github.com/fluxcd/source-controller v0.0.1-beta.1
	github.com/manifoldco/promptui v0.7.0
	github.com/spf13/cobra v1.0.0
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
	sigs.k8s.io/controller-runtime v0.6.0
	sigs.k8s.io/kustomize/api v0.4.1
	sigs.k8s.io/yaml v1.2.0
)

// fix AKS auth
replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.0.1+incompatible
