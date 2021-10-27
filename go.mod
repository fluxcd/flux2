module github.com/fluxcd/flux2

go 1.17

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

require (
	cloud.google.com/go v0.81.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.18 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.13 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20170808103936-bb23615498cd // indirect
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/acomagu/bufpipe v1.0.3 // indirect
	github.com/chai2010/gettext-go v0.0.0-20160711120539-c6fed771bfd5 // indirect
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/fluxcd/pkg/apis/acl v0.0.3 // indirect
	github.com/fluxcd/pkg/apis/kustomize v0.3.1 // indirect
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible // indirect
	github.com/fvbommel/sortorder v1.0.1 // indirect
	github.com/go-errors/errors v1.4.0 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-git/go-billy/v5 v5.3.1 // indirect
	github.com/go-logr/logr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/go-github/v35 v35.3.0 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.8 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kevinburke/ssh_config v0.0.0-20201106050909-4977a11b4351 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/mattn/go-runewidth v0.0.7 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/term v0.0.0-20210610120745-9d4ed1856297 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/russross/blackfriday v1.5.2 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/stretchr/testify v1.7.0 // indirect
	github.com/xanzy/go-gitlab v0.51.1 // indirect
	github.com/xanzy/ssh-agent v0.3.0 // indirect
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca // indirect
	go.starlark.net v0.0.0-20200306205701-8dd3e2ee1dd5 // indirect
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f // indirect
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f // indirect
	golang.org/x/sys v0.0.0-20211029165221-6e7872819dc8 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/component-base v0.23.1 // indirect
	k8s.io/klog/v2 v2.30.0 // indirect
	k8s.io/kube-openapi v0.0.0-20211115234752-e816edb12b65 // indirect
	k8s.io/utils v0.0.0-20211208161948-7d6a63dca704 // indirect
	sigs.k8s.io/json v0.0.0-20211208200746-9f7c6b3444d2 // indirect
	sigs.k8s.io/kustomize/kyaml v0.13.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.0 // indirect
)
