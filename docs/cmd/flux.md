# Flux CLI

## Overview
The Flux CLI is a tool for interacting with Flux resources on a Kubernetes cluster.

## Global Flags
- `--kubeconfig=<path>`: Path to a kubeconfig file. If unset, uses `KUBECONFIG` environment variable or `~/.kube/config`.
- `--context=<context>`: Kubernetes context to use.
- `--namespace=<namespace>`, `-n=<namespace>`: Namespace scope for the operation.
- `--timeout=<duration>`: Timeout for operations (default: 5m0s).
- `--insecure-skip-tls-verify`: Skip verification of the server's certificate chain and hostname.

## KUBECONFIG Environment Variable
Flux respects the `KUBECONFIG` environment variable, which can specify multiple kubeconfig files separated by `:` (Unix) or `;` (Windows). Files are merged following Kubernetes conventions, with later files overriding earlier ones for duplicate entries.

Example:
```bash
export KUBECONFIG=/path/to/config1:/path/to/config2
flux check --pre
