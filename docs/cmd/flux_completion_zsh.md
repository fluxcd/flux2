---
title: "flux completion zsh command"
---
## flux completion zsh

Generates zsh completion scripts

```
flux completion zsh [flags]
```

### Examples

```
To load completion run

. <(flux completion zsh) && compdef _flux flux

To configure your zsh shell to load completions for each session add to your zshrc

# ~/.zshrc or ~/.profile
command -v flux >/dev/null && . <(flux completion zsh) && compdef _flux flux

or write a cached file in one of the completion directories in your ${fpath}:

echo "${fpath// /\n}" | grep -i completion
flux completion zsh > _flux

mv _flux ~/.oh-my-zsh/completions  # oh-my-zsh
mv _flux ~/.zprezto/modules/completion/external/src/  # zprezto

```

### Options

```
  -h, --help   help for zsh
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

