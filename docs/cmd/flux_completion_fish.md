---
title: "flux completion fish command"
---
## flux completion fish

Generates fish completion scripts

```
flux completion fish [flags]
```

### Examples

```
To configure your fish shell to load completions for each session write this script to your completions dir:

flux completion fish > ~/.config/fish/completions/flux.fish

See http://fishshell.com/docs/current/index.html#completion-own for more details
```

### Options

```
  -h, --help   help for fish
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --kubeconfig string   absolute path to the kubeconfig file
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux completion](/cmd/flux_completion/)	 - Generates completion scripts for various shells

