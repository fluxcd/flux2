---
title: "flux create source bucket command"
---
## flux create source bucket

Create or update a Bucket source

### Synopsis


The create source bucket command generates a Bucket resource and waits for it to be downloaded.
For Buckets with static authentication, the credentials are stored in a Kubernetes secret.

```
flux create source bucket [name] [flags]
```

### Examples

```
  # Create a source from a Buckets using static authentication
  flux create source bucket podinfo \
	--bucket-name=podinfo \
    --endpoint=minio.minio.svc.cluster.local:9000 \
	--insecure=true \
	--access-key=myaccesskey \
	--secret-key=mysecretkey \
    --interval=10m

  # Create a source from an Amazon S3 Bucket using IAM authentication
  flux create source bucket podinfo \
	--bucket-name=podinfo \
	--provider=aws \
    --endpoint=s3.amazonaws.com \
	--region=us-east-1 \
    --interval=10m

```

### Options

```
      --access-key string               the bucket access key
      --bucket-name string              the bucket name
      --endpoint string                 the bucket endpoint address
  -h, --help                            help for bucket
      --insecure                        for when connecting to a non-TLS S3 HTTP endpoint
      --provider sourceBucketProvider   the S3 compatible storage provider name, available options are: (generic, aws) (default generic)
      --region string                   the bucket region
      --secret-key string               the bucket secret key
      --secret-ref string               the name of an existing secret containing credentials
```

### Options inherited from parent commands

```
      --context string      kubernetes context to use
      --export              export in YAML format to stdout
      --interval duration   source sync interval (default 1m0s)
      --kubeconfig string   absolute path to the kubeconfig file
      --label strings       set labels on the resource (can specify multiple labels with commas: label1=value1,label2=value2)
  -n, --namespace string    the namespace scope for this operation (default "flux-system")
      --timeout duration    timeout for this operation (default 5m0s)
      --verbose             print generated objects
```

### SEE ALSO

* [flux create source](/cmd/flux_create_source/)	 - Create or update sources

