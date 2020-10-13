## gotk create alert-provider

Create or update a Provider resource

### Synopsis

The create alert-provider command generates a Provider resource.

```
gotk create alert-provider [name] [flags]
```

### Examples

```
  # Create a Provider for a Slack channel
  gotk create alert-provider slack \
  --type slack \
  --channel general \
  --address https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK \
  --secret-ref webhook-url

  # Create a Provider for a Github repository
  gotk create alert-provider github-podinfo \
  --type github \
  --address https://github.com/stefanprodan/podinfo \
  --secret-ref github-token

```

### Options

```
      --address string      path to either the git repository, chat provider or webhook
      --channel string      channel to send messages to in the case of a chat provider
  -h, --help                help for alert-provider
      --secret-ref string   name of secret containing authentication token
      --type string         type of provider
      --username string     bot username used by the provider
```

### Options inherited from parent commands

```
      --export              export in YAML format to stdout
      --interval duration   source sync interval (default 1m0s)
      --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")
      --label strings       set labels on the resource (can specify multiple labels with commas: label1=value1,label2=value2)
  -n, --namespace string    the namespace scope for this operation (default "gotk-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [gotk create](gotk_create.md)	 - Create or update sources and resources

