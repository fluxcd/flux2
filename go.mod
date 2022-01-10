module github.com/fluxcd/flux2

go 1.16

require (
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/ProtonMail/go-crypto v0.0.0-20210428141323-04723f9f07d7
	github.com/cyphar/filepath-securejoin v0.2.2
	github.com/fluxcd/go-git-providers v0.5.0
	github.com/fluxcd/helm-controller/api v0.15.0
	github.com/fluxcd/image-automation-controller/api v0.19.0
	github.com/fluxcd/image-reflector-controller/api v0.15.0
	github.com/fluxcd/kustomize-controller/api v0.19.0
	github.com/fluxcd/notification-controller/api v0.19.0
	github.com/fluxcd/pkg/apis/meta v0.10.2
	github.com/fluxcd/pkg/runtime v0.12.3
	github.com/fluxcd/pkg/ssa v0.9.0
	github.com/fluxcd/pkg/ssh v0.3.0
	github.com/fluxcd/pkg/untar v0.0.5
	github.com/fluxcd/pkg/version v0.0.1
	github.com/fluxcd/source-controller/api v0.20.1
	github.com/go-errors/errors v1.4.0 // indirect
	github.com/go-git/go-git/v5 v5.4.2
	github.com/google/go-cmp v0.5.6
	github.com/google/go-containerregistry v0.2.0
	github.com/manifoldco/promptui v0.9.0
	github.com/mattn/go-shellwords v1.0.12
	github.com/olekukonko/tablewriter v0.0.4
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b
	k8s.io/api v0.23.1
	k8s.io/apiextensions-apiserver v0.23.1
	k8s.io/apimachinery v0.23.1
	k8s.io/cli-runtime v0.23.1
	k8s.io/client-go v0.23.1
	k8s.io/kubectl v0.23.1
	sigs.k8s.io/cli-utils v0.26.1
	sigs.k8s.io/controller-runtime v0.11.0
	sigs.k8s.io/kustomize/api v0.10.1
	sigs.k8s.io/yaml v1.3.0
)
