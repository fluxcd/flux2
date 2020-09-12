module github.com/fluxcd/toolkit

go 1.14

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/fluxcd/helm-controller/api v0.0.8
	github.com/fluxcd/kustomize-controller/api v0.0.11
	github.com/fluxcd/pkg/git v0.0.7
	github.com/fluxcd/pkg/runtime v0.0.1
	github.com/fluxcd/pkg/ssh v0.0.5
	github.com/fluxcd/pkg/untar v0.0.5
	github.com/fluxcd/source-controller/api v0.0.16
	github.com/manifoldco/promptui v0.7.0
	github.com/spf13/cobra v1.0.0
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9 // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/protobuf v1.24.0 // indirect
	k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v0.18.8
	sigs.k8s.io/controller-runtime v0.6.2
	sigs.k8s.io/kustomize/api v0.5.1
	sigs.k8s.io/yaml v1.2.0
)
