## flux create image update

Create or update an ImageUpdateAutomation object

### Synopsis

The create image update command generates an ImageUpdateAutomation resource.
An ImageUpdateAutomation object specifies an automated update to images
mentioned in YAMLs in a git repository.

```
flux create image update [name] [flags]
```

### Examples

```
  # Configure image updates for the main repository created by flux bootstrap
  flux create image update flux-system \
    --git-repo-ref=flux-system \
    --git-repo-path="./clusters/my-cluster" \
    --checkout-branch=main \
	--author-name=flux \
	--author-email=flux@example.com \
	--commit-template="{{range .Updated.Images}}{{println .}}{{end}}"

  # Configure image updates to push changes to a different branch, if the branch doesn't exists it will be created
  flux create image update flux-system \
    --git-repo-ref=flux-system \
    --git-repo-path="./clusters/my-cluster" \
    --checkout-branch=main \
    --push-branch=image-updates \
	--author-name=flux \
	--author-email=flux@example.com \
	--commit-template="{{range .Updated.Images}}{{println .}}{{end}}"

```

### Options

```
      --author-email string      the email to use for commit author
      --author-name string       the name to use for commit author
      --checkout-branch string   the branch to checkout
      --commit-template string   a template for commit messages
      --git-repo-path string     path to the directory containing the manifests to be updated, defaults to the repository root
      --git-repo-ref string      the name of a GitRepository resource with details of the upstream Git repository
  -h, --help                     help for update
      --push-branch string       the branch to push commits to, defaults to the checkout branch if not specified
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

