## flux create image policy

Create or update an ImagePolicy object

### Synopsis

The create image policy command generates an ImagePolicy resource.
An ImagePolicy object calculates a "latest image" given an image
repository and a policy, e.g., semver.

The image that sorts highest according to the policy is recorded in
the status of the object.

```
flux create image policy [name] [flags]
```

### Examples

```
  # Create an ImagePolicy to select the latest stable release
  flux create image policy podinfo \
    --image-ref=podinfo \
    --select-semver=">=1.0.0"

  # Create an ImagePolicy to select the latest main branch build tagged as "${GIT_BRANCH}-${GIT_SHA:0:7}-$(date +%s)"
  flux create image policy podinfo \
    --image-ref=podinfo \
    --select-numeric=asc \
	--filter-regex='^main-[a-f0-9]+-(?P<ts>[0-9]+)' \
	--filter-extract='$ts'

```

### Options

```
      --filter-extract string   replacement pattern (using capture groups from --filter-regex) to use for sorting
      --filter-regex string     regular expression pattern used to filter the image tags
  -h, --help                    help for policy
      --image-ref string        the name of an image repository object
      --select-alpha string     use alphabetical sorting to select image; either "asc" meaning select the last, or "desc" meaning select the first
      --select-numeric string   use numeric sorting to select image; either "asc" meaning select the last, or "desc" meaning select the first
      --select-semver string    a semver range to apply to tags; e.g., '1.x'
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --export              export in YAML format to stdout
      --interval duration   source sync interval (default 1m0s)
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --label strings       set labels on the resource (can specify multiple labels with commas: label1=value1,label2=value2)
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux create image](flux_create_image.md)	 - Create or update resources dealing with image automation

