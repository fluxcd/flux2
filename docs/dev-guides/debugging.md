# Advanced debugging

This guide covers more advanced debugging topics such as collecting
runtime profiling data from GitOps Toolkit components.

As a user, this page normally should be a last resort, but you may
be asked by a maintainer to share a [collected profile](#collecting-a-profile)
to debug e.g. performance issues.

## Pprof

The [GitOps Toolkit components](../components/index.md) serve [`pprof`](https://golang.org/pkg/net/http/pprof/)
runtime profiling data on their metrics HTTP server (default `:8080`).

### Endpoints

| Endpoint    | Path                   |
|-------------|------------------------|
| Index       | `/debug/pprof/`        |
| CPU profile | `/debug/pprof/profile` |
| Symbol      | `/debug/pprof/symbol`  |
| Trace       | `/debug/pprof/trace`   |

### Collecting a profile

To collect a profile, port-forward to the component's metrics endpoint
and collect the data from the [endpoint](#endpoints) of choice:

```console
$ kubectl port-forward -n <namespace> deploy/<component> 8080
$ curl -Sk -v http://localhost:8080/debug/pprof/heap > heap.out
```

The collected profile [can be analyzed using `go`](https://blog.golang.org/pprof),
or shared with one of the maintainers.

## Resource usage

As `kubectl top` gives a limited (and at times inaccurate) overview of
resource usage, it is often better to make use of the Grafana metrics
to gather insights. See [monitoring](../guides/monitoring.md) for a
guide on how to visualize this data with a Grafana dashboard.
