## gotk export receiver

Export Receiver resources in YAML format

### Synopsis

The export receiver command exports one or all Receiver resources in YAML format.

```
gotk export receiver [name] [flags]
```

### Examples

```
  # Export all Receiver resources
  gotk export rcv --all > receivers.yaml

  # Export a Receiver
  gotk export rcv main > main.yaml

```

### Options

```
  -h, --help   help for receiver
```

### Options inherited from parent commands

```
      --all                 select all resources
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk export](gotk_export.md)	 - Export resources in YAML format

