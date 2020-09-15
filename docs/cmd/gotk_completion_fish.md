## gotk completion fish

Generates fish completion scripts

### Synopsis

Generates fish completion scripts

```
gotk completion fish [flags]
```

### Examples

```
To load completion run

. <(gotk completion fish)

To configure your fish shell to load completions for each session write this script to your completions dir:

gotk completion fish > ~/.config/fish/completions/gotk

See http://fishshell.com/docs/current/index.html#completion-own for more details

```

### Options

```
  -h, --help   help for fish
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

