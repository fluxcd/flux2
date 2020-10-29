## flux completion fish

Generates fish completion scripts

### Synopsis

Generates fish completion scripts

```
flux completion fish [flags]
```

### Examples

```
To load completion run

. <(flux completion fish)

To configure your fish shell to load completions for each session write this script to your completions dir:

flux completion fish > ~/.config/fish/completions/flux

See http://fishshell.com/docs/current/index.html#completion-own for more details

```

### Options

```
  -h, --help   help for fish
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux completion](flux_completion.md)	 - Generates completion scripts for various shells

