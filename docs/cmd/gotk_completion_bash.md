## gotk completion bash

Generates bash completion scripts

### Synopsis

Generates bash completion scripts

```
gotk completion bash [flags]
```

### Examples

```
To load completion run

. <(gotk completion bash)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
command -v gotk >/dev/null && . <(gotk completion bash)

```

### Options

```
  -h, --help   help for bash
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

