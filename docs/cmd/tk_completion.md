## tk completion

Generates bash completion scripts

### Synopsis

Generates bash completion scripts

```
tk completion [flags]
```

### Examples

```
To load completion run

. <(tk completion)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(tk completion)

```

### Options

```
  -h, --help   help for completion
```

### Options inherited from parent commands

```
      --components strings   list of components, accepts comma-separated values (default [source-controller,kustomize-controller])
      --kubeconfig string    path to the kubeconfig file (default "~/.kube/config")
      --namespace string     the namespace scope for this operation (default "gitops-system")
      --timeout duration     timeout for this operation (default 5m0s)
      --verbose              print generated objects
```

### SEE ALSO

* [tk](tk.md)	 - Command line utility for assembling Kubernetes CD pipelines

