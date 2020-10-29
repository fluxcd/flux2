## gotk resume alert

Resume a suspended Alert

### Synopsis

The resume command marks a previously suspended Alert resource for reconciliation and waits for it to
finish the apply.

```
gotk resume alert [name] [flags]
```

### Examples

```
  # Resume reconciliation for an existing Alert
  gotk resume alert main

```

### Options

```
  -h, --help   help for alert
```

### Options inherited from parent commands

```
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk resume](gotk_resume.md)	 - Resume suspended resources

