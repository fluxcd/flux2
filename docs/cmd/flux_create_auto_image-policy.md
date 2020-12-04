## flux create auto image-policy

Create or update an ImagePolicy object

### Synopsis

The create auto image-policy command generates an ImagePolicy resource.
An ImagePolicy object calculates a "latest image" given an image
repository and a policy, e.g., semver.

The image that sorts highest according to the policy is recorded in
the status of the object.

```
flux create auto image-policy <name> [flags]
```

### Options

```
  -h, --help               help for image-policy
      --image-ref string   the name of an image repository object
      --semver string      a semver range to apply to tags; e.g., '1.x'
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

* [flux create auto](flux_create_auto.md)	 - Create or update resources dealing with automation

