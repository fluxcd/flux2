## flux completion powershell

Generates powershell completion scripts

### Synopsis

Generates powershell completion scripts

```
flux completion powershell [flags]
```

### Examples

```
To load completion run

. <(flux completion powershell)

To configure your powershell shell to load completions for each session add to your powershell profile

Windows:

cd "$env:USERPROFILE\Documents\WindowsPowerShell\Modules"
flux completion >> flux-completion.ps1

Linux:

cd "${XDG_CONFIG_HOME:-"$HOME/.config/"}/powershell/modules"
flux completion >> flux-completions.ps1

```

### Options

```
  -h, --help   help for powershell
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux completion](flux_completion.md)	 - Generates completion scripts for various shells

