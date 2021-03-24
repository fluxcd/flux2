---
title: "flux completion bash command"
---
## flux completion bash

Generates bash completion scripts

```
flux completion bash [flags]
```

### Examples

```
To load completion run

. <(flux completion bash)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
command -v flux >/dev/null && . <(flux completion bash)

```

### Options

```
  -h, --help   help for bash
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

