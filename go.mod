module github.com/fluxcd/flux2

go 1.16

require (
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/ProtonMail/go-crypto v0.0.0-20210428141323-04723f9f07d7
	github.com/cyphar/filepath-securejoin v0.2.2
	github.com/fluxcd/go-git-providers v0.3.1
	github.com/fluxcd/helm-controller/api v0.12.2
	github.com/fluxcd/image-automation-controller/api v0.17.1
	github.com/fluxcd/image-reflector-controller/api v0.13.1
	github.com/fluxcd/kustomize-controller/api v0.17.0
	github.com/fluxcd/notification-controller/api v0.18.1
	github.com/fluxcd/pkg/apis/meta v0.10.0
	github.com/fluxcd/pkg/runtime v0.12.0
	github.com/fluxcd/pkg/ssa v0.3.1
	github.com/fluxcd/pkg/ssh v0.0.5
	github.com/fluxcd/pkg/untar v0.0.5
	github.com/fluxcd/pkg/version v0.0.1
	github.com/fluxcd/source-controller/api v0.17.2
	github.com/go-errors/errors v1.4.0 // indirect
	github.com/go-git/go-git/v5 v5.4.2
	github.com/google/go-cmp v0.5.6
	github.com/google/go-containerregistry v0.2.0
	github.com/manifoldco/promptui v0.9.0
	github.com/mattn/go-shellwords v1.0.12
	github.com/olekukonko/tablewriter v0.0.4
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b
	golang.org/x/term v0.0.0-20210220032956-6a3ed077a48d
	k8s.io/api v0.22.2
	k8s.io/apiextensions-apiserver v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	k8s.io/kubectl v0.21.1
	sigs.k8s.io/cli-utils v0.26.0
	sigs.k8s.io/controller-runtime v0.10.2
	sigs.k8s.io/kustomize/api v0.8.10
	sigs.k8s.io/yaml v1.3.0
)
