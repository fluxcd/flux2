# Watching for source changes

In this guide you'll be developing a Kubernetes controller with
[Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
that subscribes to [GitRepository](../components/source/gitrepositories.md)
events and reacts to revision changes by downloading the artifact produced by 
[source-controller](../components/source/controller.md).

## Prerequisites

On your dev machine install the following tools:

* go >= 1.13
* kubebuilder >= 2.3
* kind >= 0.8
* kubectl >= 1.18
* kustomize >= 3.5
* docker >= 19.03

## Install the GitOps Toolkit

Create a cluster for testing:

```sh
kind create cluster --name dev
```

Install the toolkit CLI:

```sh
curl -s https://toolkit.fluxcd.io/install.sh | sudo bash
```

Verify that your dev machine satisfies the prerequisites with:

```sh
tk check --pre
```

Install the toolkit controllers on the dev cluster:

```sh
tk install
```

## Clone the sample controller

You'll be using [stefanprodan/source-watcher](https://github.com/stefanprodan/source-watcher) as
a template for developing your own controller. The source-watcher was scaffolded with `kubebuilder init`.

Clone the source-watcher repo:

```sh
git clone https://github.com/stefanprodan/source-watcher
cd source-watcher
```

Build the controller:

```sh
make
```

## Run the controller

Port forward to source-controller artifacts server:

```sh
kubectl -n gitops-system port-forward svc/source-controller 8181:80
```

Export the local address as `SOURCE_HOST`:

```sh
export SOURCE_HOST=localhost:8181
```

Run source-watcher locally:

```sh
make run
```

Create a Git source:

```sh
tk create source git test \
--url=https://github.com/stefanprodan/podinfo \
--tag=4.0.0
```

The source-watcher should log the revision:

```console
New revision detected   {"gitrepository": "gitops-system/test", "revision": "4.0.0/ab953493ee14c3c9800bda0251e0c507f9741408"}
Extracted tarball into /var/folders/77/3y6x_p2j2g9fspdkzjbm5_s40000gn/T/test292235827: 123 files, 29 dirs (32.603415ms)
Processing files...
```

Change the Git tag:

```sh
tk create source git test \
--url=https://github.com/stefanprodan/podinfo \
--tag=4.0.1
```

The source-watcher should log the new revision:

```console
New revision detected   {"gitrepository": "gitops-system/test", "revision": "4.0.1/113360052b3153e439a0cf8de76b8e3d2a7bdf27"}
```

The source-controller reports the revision under `GitRepository.Status.Artifact.Revision` in the format: `<branch|tag>/<commit>`.

## How it works

The [GitRepositoryWatcher](https://github.com/stefanprodan/source-watcher/blob/master/controllers/gitrepository_watcher.go)
controller does the following:

* subscribes to `GitRepository` events
* detects when the Git revision changes
* downloads and extracts the source artifact
* write to stdout the extracted file names

```go
// GitRepositoryWatcher watches GitRepository objects for revision changes
type GitRepositoryWatcher struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=source.fluxcd.io,resources=gitrepositories,verbs=get;list;watch
// +kubebuilder:rbac:groups=source.fluxcd.io,resources=gitrepositories/status,verbs=get

func (r *GitRepositoryWatcher) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// set timeout for the reconciliation
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// get source object
	var repository sourcev1.GitRepository
	if err := r.Get(ctx, req.NamespacedName, &repository); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log := r.Log.WithValues(strings.ToLower(repository.Kind), req.NamespacedName)
	log.Info("New revision detected", "revision", repository.Status.Artifact.Revision)

	// create tmp dir
	tmpDir, err := ioutil.TempDir("", repository.Name)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to create temp dir, error: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// download and extract artifact
	summary, err := r.fetchArtifact(ctx, repository, tmpDir)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to fetch artifact, error: %w", err)
	}
	log.Info(summary)

	// list artifact content
	files, err := ioutil.ReadDir(tmpDir)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to list files, error: %w", err)
	}

	// do something with the artifact content
	for _, f := range files {
		log.Info("Processing " + f.Name())
	}

	return ctrl.Result{}, nil
}

func (r *GitRepositoryWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sourcev1.GitRepository{}).
		WithEventFilter(GitRepositoryRevisionChangePredicate{}).
		Complete(r)
}
```

To add the watcher to an existing project, copy the controller and the revision change predicate to your `controllers` dir:

* [gitrepository_watcher.go](https://github.com/stefanprodan/source-watcher/blob/master/controllers/gitrepository_watcher.go)
* [gitrepository_predicate.go](https://github.com/stefanprodan/source-watcher/blob/master/controllers/gitrepository_predicate.go)

In your `main.go` init function, register the Source API schema:

```go
import sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = sourcev1.AddToScheme(scheme)

	// +kubebuilder:scaffold:scheme
}
```

Start the controller in the main function:

```go
func main()  {

	if err = (&controllers.GitRepositoryWatcher{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("GitRepositoryWatcher"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GitRepositoryWatcher")
		os.Exit(1)
	}

}
```

Note that the watcher controller depends on Kubernetes client-go >= 1.18.
Your `go.mod` should require controller-runtime v0.6 or newer:

```go
require (
	k8s.io/apimachinery v0.18.4
	k8s.io/client-go v0.18.4
	sigs.k8s.io/controller-runtime v0.6.0
)
```

That's it! Happy hacking!
