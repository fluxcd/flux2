## tk resume kustomization

Resume a suspended Kustomization

### Synopsis

The resume command marks a previously suspended Kustomization resource for reconciliation and waits for it to
finish the apply.

```
tk resume kustomization [name] [flags]
```

### Options

```
  -h, --help   help for kustomization
```

### Options inherited from parent commands

```
      --components strings   list of components, accepts comma-separated values (default [source-controller,kustomize-controller])
      --kubeconfig string    path to the kubeconfig file (default "~/.kube/config")
      --namespace string     the namespace scope for this operation (default "gitops-system")
      --timeout duration     timeout for this operation (default 5m0s)
      --verbose              print generated objects
```

### SEE ALSO

* [tk resume](tk_resume.md)	 - Resume suspended resources

