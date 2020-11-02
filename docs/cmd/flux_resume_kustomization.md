## flux resume kustomization

Resume a suspended Kustomization

### Synopsis

The resume command marks a previously suspended Kustomization resource for reconciliation and waits for it to
finish the apply.

```
flux resume kustomization [name] [flags]
```

### Examples

```
  # Resume reconciliation for an existing Kustomization
  flux resume ks podinfo

```

### Options

```
  -h, --help   help for kustomization
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

* [flux resume](flux_resume.md)	 - Resume suspended resources

