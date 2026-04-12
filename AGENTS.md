# AGENTS.md

Guidance for AI coding assistants working in `fluxcd/flux2`. Read this file before making changes.

## Contribution workflow for AI agents

These rules come from [`fluxcd/flux2/CONTRIBUTING.md`](https://github.com/fluxcd/flux2/blob/main/CONTRIBUTING.md) and apply to every Flux repository.

- **Do not add `Signed-off-by` or `Co-authored-by` trailers with your agent name.** Only a human can legally certify the DCO.
- **Disclose AI assistance** with an `Assisted-by` trailer naming your agent and model:
  ```sh
  git commit -s -m "Add support for X" --trailer "Assisted-by: <agent-name>/<model-id>"
  ```
  The `-s` flag adds the human's `Signed-off-by` from their git config — do not remove it.
- **Commit message format:** Subject in imperative mood ("Add feature X" instead of "Adding feature X"), capitalized, no trailing period, ≤50 characters. Body wrapped at 72 columns, explaining what and why. No `@mentions` or `#123` issue references in the commit — put those in the PR description.
- **Trim verbiage:** in PR descriptions, commit messages, and code comments. No marketing prose, no restating the diff, no emojis.
- **Rebase, don't merge:** Never merge `main` into the feature branch; rebase onto the latest `main` and push with `--force-with-lease`. Squash before merge when asked.
- **Pre-PR gate:** `make tidy fmt vet && make test` must pass and the working tree must be clean.
- **Flux is GA:** Backward compatibility is mandatory. Breaking changes to CLI flags, output format, or behavior will be rejected. Design additive changes.
- **Copyright:** All new `.go` files must begin with the header from `cmd/flux/main.go` (Apache 2.0). Update the year to the current year when copying.
- **Tests:** New features, improvements and fixes must have test coverage. Add unit tests in `cmd/flux/*_test.go` tagged `//go:build unit`. Follow the existing `cmdTestCase` + golden file patterns. Run tests locally before pushing.

## Code quality

Before submitting code, review your changes for the following:

- **No secrets in logs or output.** Never surface auth tokens, passwords, deploy keys, or credential URLs in error messages, log lines, or CLI output. Bootstrap and source-secret commands handle sensitive material — take extra care.
- **No unchecked I/O.** Close HTTP response bodies, file handles, and tar readers in `defer` statements. Check and propagate errors from I/O operations.
- **No path traversal.** Validate and sanitize file paths extracted from archives or user input. Never `filepath.Join` with untrusted components without validation.
- **No command injection.** Do not shell out via `os/exec` for git, helm, or kustomize operations. Use the Go libraries already in use (`fluxcd/pkg/git`, `fluxcd/pkg/kustomize`, `fluxcd/pkg/ssa`).
- **No hardcoded defaults for security settings.** TLS verification must remain enabled by default. Git auth settings come from user-provided secrets.
- **Error handling.** Wrap errors with `%w` for chain inspection. Do not swallow errors silently. CLI errors must be actionable — tell the user what went wrong and how to fix it without leaking internal state.
- **Resource cleanup.** Ensure temporary files and directories (manifest staging, downloaded tarballs) are cleaned up on all code paths (success and error). Use `defer` and `t.TempDir()` in tests.
- **No panics.** Never use `panic` in runtime code paths. Return errors and let the CLI handle them gracefully.
- **Output discipline.** Machine-readable data (tables, YAML, JSON) goes to stdout via `rootCmd.OutOrStdout()`. Human-readable status messages go to stderr via the `stderrLogger`.
- **Minimal surface.** Keep new exported APIs in `pkg/` to the minimum needed. Every export is a backward-compatibility commitment.

## Project overview

flux2 is the Flux CLI (`flux` command) and distribution repository. It is **not** a controller — it consumes CRD APIs from six independent controller repos (source-controller, kustomize-controller, helm-controller, notification-controller, image-reflector-controller, image-automation-controller). It serves two purposes:

1. **CLI tool** — a Cobra-based binary that installs Flux onto Kubernetes clusters, bootstraps GitOps pipelines, and manages all Flux CRD objects (create, get, export, reconcile, suspend, resume, delete, diff, build, etc.).
2. **Distribution hub** — it bundles the Kustomize manifests for all Flux controllers and releases them as `manifests.tar.gz` on GitHub. Those manifests are also compiled into the binary itself via `//go:embed`.

## Repository layout

- `cmd/flux/` — all CLI source. Single `main` package with one file per command or resource type. `main.go` defines the root cobra command with global flags. `manifests.embed.go` embeds the generated controller manifests via `//go:embed`.
- `internal/build/` — `flux build kustomization` logic (kustomize-based diff/build, SOPS secret masking).
- `internal/flags/` — custom `pflag.Value` types providing enum validation (e.g. `LogLevel`, `ECDSACurve`, `RSAKeyBits`, `PublicKeyAlgorithm`, `DecryptionProvider`).
- `internal/tree/` — tree-printing helper for `flux tree kustomization`.
- `internal/utils/` — shared helpers: `KubeClient`, `KubeConfig`, `NewScheme` (registers all controller API groups), `Apply` (SSA-based two-phase apply), `ExecKubectlCommand`, `ValidateComponents`.
- `pkg/bootstrap/` — bootstrap orchestration: `Run()`, `PlainGitBootstrapper`, `ProviderBootstrapper`. `provider/` has the git provider factory (GitHub, GitLab, Gitea, Bitbucket).
- `pkg/log/` — `Logger` interface (`Actionf`, `Generatef`, `Waitingf`, `Successf`, `Warningf`, `Failuref`).
- `pkg/manifestgen/` — manifest generation for install, sync, kustomization, and source secrets.
- `pkg/printers/` — specialized printers `TablePrinter` and `DyffPrinter`.
- `pkg/status/` — `StatusChecker` using `fluxcd/cli-utils` kstatus polling.
- `pkg/uninstall/` — `flux uninstall` logic.
- `manifests/` — Kustomize bases per controller, RBAC, network policies, CRD references, and `scripts/bundle.sh` which runs `kustomize build` to generate `cmd/flux/manifests/`.
- `tests/integration/` — cloud e2e tests (Azure/GCP) with their own `go.mod` and Terraform infrastructure.
- `rfcs/` — Request for Comments documents for major design proposals and changes.

## CLI architecture

Commands are Cobra-based, organized as parent + per-resource children:
- Group files (`create.go`, `get.go`, `reconcile.go`, etc.) register the parent subcommand.
- Per-resource files (`create_kustomization.go`, `get_helmrelease.go`, etc.) register children.

Core interfaces in `cmd/flux/` enable generic command implementations:
- `adapter` / `copyable` / `listAdapter` — wrap controller API types for generic CRUD.
- `reconcilable` — annotate-and-poll pattern for triggering reconciliation.
- `summarisable` — generic table output for `get` commands.

Each resource type (e.g. `kustomizationAdapter` in `kustomization.go`) wraps the controller API type and implements these interfaces. Follow this pattern when adding new resource support.

Commands interact with the Kubernetes API via `internal/utils.KubeClient()` → `client.WithWatch`. `internal/utils.NewScheme()` registers all six controller API groups plus core k8s types. `internal/utils.Apply()` implements SSA-based two-phase apply (CRDs/Namespaces first, then remaining objects).

## Manifest pipeline

1. `manifests/bases/<controller>/` contains a Kustomize base per controller referencing the controller's GitHub release for CRDs and deployment manifests.
2. `manifests/install/kustomization.yaml` assembles all bases plus RBAC and policies.
3. `manifests/scripts/bundle.sh` runs `kustomize build` on each base, writing output to `cmd/flux/manifests/` (not checked in — generated).
4. The Makefile `$(EMBEDDED_MANIFESTS_TARGET)` runs `bundle.sh` and creates a sentinel file `cmd/flux/.manifests.done`.
5. `cmd/flux/manifests.embed.go` uses `//go:embed manifests/*.yaml` to compile everything into the binary.

When modifying `manifests/`, always run `make build` and verify the generated output before committing. Never hand-edit files under `cmd/flux/manifests/`.

## Build, test, lint

All targets in the root `Makefile`. Go version tracks `go.mod`.

- `make tidy` — tidy the root module and `tests/integration/`.
- `make fmt` / `make vet` — run in the root module.
- `make build` — builds `bin/flux` (CGO disabled, version injected via ldflags). Depends on embedded manifests being generated.
- `make build-dev` — builds with `DEV_VERSION`.
- `make install` / `make install-dev` — `go install` or copy to `/usr/local/bin`.
- `make test` — unit tests with envtest: runs `tidy fmt vet install-envtest`, then `go test ./... -coverprofile cover.out --tags=unit $(TEST_ARGS)`.
- `make e2e` — e2e tests against a live cluster: `go test ./cmd/flux/... --tags=e2e -v -failfast`.
- `make test-with-kind` — sets up a kind cluster, runs e2e, tears it down.
- `make install-envtest` — downloads `setup-envtest` and fetches k8s binaries into `testbin/`.

Run a single test: `make test TEST_ARGS='-run TestCreate -v'`.

## Codegen and generated files

Check `go.mod` and the `Makefile` for current dependency and tool versions. The main codegen pipeline is the manifest bundle:

```sh
./manifests/scripts/bundle.sh
```

Generated files (never hand-edit):

- `cmd/flux/manifests/*.yaml` — generated by `bundle.sh` from `manifests/` sources.
- `cmd/flux/.manifests.done` — sentinel file tracking bundle state.

Bump `fluxcd/pkg/*` and controller `api` modules as a set. Run `make tidy` after any bump.

## Conventions

- Standard `gofmt`. All exported names need doc comments.
- **Command pattern:** follow the existing group-parent + per-resource-child cobra structure. New resources need an adapter type implementing `adapter`, `copyable`, and the relevant command interfaces (`reconcilable`, `summarisable`, etc.).
- **Output:** stderr for human status messages via `stderrLogger` (Unicode symbols: `►` action, `✔` success, `✗` failure, `◎` waiting, `⚠️` warning, `✚` generate). Stdout for machine-readable data (tables, YAML, JSON) via `rootCmd.OutOrStdout()`.
- **Global flags:** kubeconfig flags come from `k8s.io/cli-runtime/pkg/genericclioptions.ConfigFlags`. Client tuning comes from `fluxcd/pkg/runtime/client.Options`. `FLUX_SYSTEM_NAMESPACE` env var overrides the default namespace.
- **SSA apply:** always use `internal/utils.Apply()` (two-phase: CRDs/Namespaces first, then rest). Do not apply manifests directly via the k8s client.
- **Reconcile triggering:** patch `meta.ReconcileRequestAnnotation` with a timestamp, then poll with `kstatus.Compute()` until ready. See `reconcile.go`.
- **Error handling:** return errors from `RunE`. Use `*RequestError` with exit codes for actionable CLI errors. Exit code 1 = warning, anything else = failure.
- **Flags:** use `internal/flags/` custom `pflag.Value` types for enum flags (providers, algorithms, sources). Add new enum types there.

## Testing

Three test suites with build tags:

- **Unit** (`//go:build unit`): lives in `cmd/flux/*_test.go`. Uses `controller-runtime/envtest` for an in-process fake k8s API. CRDs are loaded from `cmd/flux/manifests/` (embedded manifests). Pattern: `cmdTestCase{args: "...", assert: assertGoldenFile("testdata/...")}`. The `executeCommand()` helper captures stdout.
- **E2e** (`//go:build e2e`): lives in `cmd/flux/*_test.go`. Requires a live cluster via `TEST_KUBECONFIG`. `TestMain` runs `flux install` for setup and teardown.
- **Integration** (`//go:build integration`): lives in `tests/integration/` with its own `go.mod`. Uses Terraform-provisioned cloud clusters.

Golden files live in `cmd/flux/testdata/`. Update them with `go test ./cmd/flux/... --tags=unit -update`.

Run a single unit test: `make test TEST_ARGS='-run TestInstall -v'`.

## Gotchas and non-obvious rules

- The `cmd/flux/manifests/` directory is **generated, not checked in**. It is created by `manifests/scripts/bundle.sh` and embedded into the binary. `make build` and `make test` both trigger the bundle if the sentinel file is stale.
- `kustomize` must be on `PATH` for `bundle.sh` to work. If you see "command not found" errors during build, install kustomize.
- `internal/utils.NewScheme()` registers all six controller API groups. Adding support for a new CRD type means updating the scheme registration there.
- The `VERSION` constant is injected via `-ldflags` at build time. In dev builds it defaults to `0.0.0-dev.0`. The embedded manifest version check (`isEmbeddedVersion`) determines whether `flux install` uses compiled-in manifests or downloads from GitHub.
- `resetCmdArgs()` in tests is critical — Cobra persists flag state between test runs. Every test case must reset to avoid pollution.
- `executeCommand()` captures stdout only. Stderr output (from `stderrLogger`) is not captured in test assertions. If your command's output goes to the wrong stream, tests will silently pass with empty golden files.
- The `adapter` / `listAdapter` interfaces use type assertions internally. If you add a new resource type and forget to implement an interface method, you'll get a runtime panic in the generic command handler, not a compile error. Add interface compliance checks (`var _ reconcilable = ...`).
- Bootstrap commands create real Git commits and push to real repos. E2e tests for bootstrap need careful cleanup. Do not add bootstrap e2e tests without a corresponding teardown.
- `pkg/` packages are importable by external consumers (e.g. Terraform provider, other tools). Treat their exported surface as public API.
