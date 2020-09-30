## gotk completion zsh

Generates zsh completion scripts

### Synopsis

Generates zsh completion scripts

```
gotk completion zsh [flags]
```

### Examples

```
To load completion run

. <(gotk completion zsh) && compdef _gotk gotk

To configure your zsh shell to load completions for each session add to your zshrc

# ~/.zshrc or ~/.profile
command -v gotk >/dev/null && . <(gotk completion zsh) && compdef _gotk gotk

or write a cached file in one of the completion directories in your ${fpath}:

echo "${fpath// /\n}" | grep -i completion
gotk completions zsh > _gotk

mv _gotk ~/.oh-my-zsh/completions  # oh-my-zsh
mv _gotk ~/.zprezto/modules/completion/external/src/  # zprezto

```

### Options

```
  -h, --help   help for zsh
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk completion](gotk_completion.md)	 - Generates completion scripts for various shells

