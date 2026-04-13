# RFC-XXXX Flux CLI Plugin System

**Status:** provisional

**Creation date:** 2026-03-30

**Last update:** 2026-04-01

## Summary

This RFC proposes a plugin system for the Flux CLI that allows external CLI tools to be
discoverable and invocable as `flux <name>` subcommands. Plugins are installed from a
centralized catalog hosted on GitHub, with SHA-256 checksum verification and automatic
version updates. The design follows the established kubectl plugin pattern used across
the Kubernetes ecosystem.

## Motivation

The Flux CLI currently has no mechanism for extending its functionality with external tools.
Projects like [flux-operator](https://github.com/controlplaneio-fluxcd/flux-operator) and
[flux-local](https://github.com/allenporter/flux-local) provide complementary CLI tools
that users install and invoke separately. This creates a fragmented user experience where
Flux-related workflows require switching between multiple binaries with different flag
conventions and discovery mechanisms.

The Kubernetes ecosystem has a proven model for CLI extensibility: kubectl plugins are
executables prefixed with `kubectl-` that can be discovered, installed via
[krew](https://krew.sigs.k8s.io/), and invoked as `kubectl <name>`. This model has
been widely adopted and is well understood by Kubernetes users.

### Goals

- Allow external CLI tools to be invoked as `flux <name>` subcommands without modifying
  the external binary.
- Provide a `flux plugin install` command to download plugins from a centralized catalog
  with checksum verification.
- Support shell completion for plugin subcommands by delegating to the plugin's own
  Cobra `__complete` command.
- Support plugins written as scripts (Python, Bash, etc.) via symlinks into the
  plugin directory.
- Ensure built-in commands always take priority over plugins.
- Keep the plugin system lightweight with zero impact on non-plugin Flux commands.

### Non-Goals

- Plugin dependency management (plugins are standalone binaries).
- Cosign/SLSA signature verification (SHA-256 only in v1beta1; signatures can be added later).
- Automatic update checks on startup (users run `flux plugin update` explicitly).
- Private catalog authentication (users can use `$FLUXCD_PLUGIN_CATALOG` with TLS).
- Flag sharing between Flux and plugins (`--namespace`, `--context`, etc. are not
  forwarded; plugins manage their own flags).

## Proposal

### Plugin Discovery

Plugins are executables prefixed with `flux-` placed in a single plugin directory.
The `flux-<name>` binary maps to the `flux <name>` command. For example,
`flux-operator` becomes `flux operator`.

The default plugin directory is `~/.fluxcd/plugins/`. Users can override it with the
`FLUXCD_PLUGINS` environment variable. Only this single directory is scanned.

When a plugin is discovered, it appears under a "Plugin Commands:" group in `flux --help`:

```
Plugin Commands:
  operator    Runs the operator plugin

Additional Commands:
  bootstrap   Deploy Flux on a cluster the GitOps way.
  ...
```

### Plugin Execution

On macOS and Linux, `flux operator export report` replaces the current process with
`flux-operator export report` via `syscall.Exec`, matching kubectl's behavior.
On Windows, the plugin runs as a child process with full I/O passthrough.
All arguments after the plugin name are passed through verbatim with
`DisableFlagParsing: true`.

### Shell Completion

Shell completion is delegated to the plugin binary via Cobra's `__complete` protocol.
When the user types `flux operator get <TAB>`, Flux runs
`flux-operator __complete get ""` and returns the results. This works automatically
for all Cobra-based plugins (like flux-operator). Non-Cobra plugins gracefully degrade
to no completions.

### Plugin Catalog

A dedicated GitHub repository ([fluxcd/plugins](https://github.com/fluxcd/plugins))
serves as the plugin catalog. Each plugin has a YAML manifest:

```yaml
apiVersion: cli.fluxcd.io/v1beta1
kind: Plugin
name: operator
description: Flux Operator CLI
homepage: https://fluxoperator.dev/
source: https://github.com/controlplaneio-fluxcd/flux-operator
bin: flux-operator
versions:
  - version: 0.45.0
    platforms:
      - os: darwin
        arch: arm64
        url: https://github.com/.../flux-operator_0.45.0_darwin_arm64.tar.gz
        checksum: sha256:cd85d5d84d264...
      - os: linux
        arch: amd64
        url: https://github.com/.../flux-operator_0.45.0_linux_amd64.tar.gz
        checksum: sha256:96198da969096...
      - os: windows
        arch: amd64
        url: https://github.com/.../flux-operator_0.45.0_windows_amd64.zip
        checksum: sha256:9712026094a5...
```

The plugin manifest includes metadata (name, description, homepage, source repo), the binary name
(`bin`), and a list of versions with platform-specific download URLs and checksums.

The download URLs can point to one of the following formats:

- An archive containing the binary (`tar`, `tar.gz` or `zip`), with the binary at the root of the archive. The binary name inside the archive must match the `bin` field in the manifest.
- An optional `extractPath` field can be specified in a `platforms` entry to override this default, either because the binary has a different name on this platform, or because it is nested in a subfolder rather than at the root of the archive. It accepts an absolute path to a file within the archive (e.g., `bin/flux-operator`).
- A direct binary URL. The binary is downloaded and saved without extraction. The `bin` field is used for naming the installed plugin, not for discovery in this case.
- Note that when the OS is Windows, the binary name is composed by appending `.exe` to the `bin` field (e.g., `flux-operator.exe`).

The Flux Operator CLI detects if the URL points to an archive by checking the file extension.
If no extension is present, it checks the content type by probing for archive magic bytes after downloading the file.
If the content is not an archive, it treats it as a direct binary URL.

A generated `catalog.yaml` (`PluginCatalog` kind) contains static metadata for all
plugins, enabling `flux plugin search` with a single HTTP fetch.

### CLI Commands

| Command | Description |
|---------|-------------|
| `flux plugin list` (alias: `ls`) | List installed plugins with versions and paths |
| `flux plugin install <name>[@<version>]` | Install a plugin from the catalog |
| `flux plugin uninstall <name>` | Remove a plugin binary and receipt |
| `flux plugin update [name]` | Update one or all installed plugins |
| `flux plugin search [query]` | Search the plugin catalog |

### Install Flow

1. Fetch `plugins/<name>.yaml` from the catalog URL
2. Validate `apiVersion: cli.fluxcd.io/v1beta1` and `kind: Plugin`
3. Resolve version (latest if unspecified, or match `@version`)
4. Find platform entry matching `runtime.GOOS` / `runtime.GOARCH`
5. Download archive to temp file with SHA-256 checksum verification
6. Extract only the declared binary from the archive (tar.gz or zip), streaming
   directly to disk without buffering in memory
7. Write binary to plugin directory as `flux-<name>` (mode `0755`)
8. Write install receipt (`flux-<name>.yaml`) recording version, platform, download URL, checksum and timestamp

Install is idempotent -- reinstalling overwrites the binary and receipt.

### Install Receipts

When a plugin is installed via `flux plugin install`, a receipt file is written
next to the binary:

```yaml
name: operator
version: "0.45.0"
installedAt: "2026-03-30T10:00:00Z"
platform:
  os: darwin
  arch: arm64
  url: https://github.com/.../flux-operator_0.45.0_darwin_arm64.tar.gz
  checksum: sha256:cd85d5d84d264...
```

Receipts enable `flux plugin list` to show versions, `flux plugin update` to compare
installed vs. latest, and provenance tracking. Manually installed plugins (no receipt)
show `manual` in listings and are skipped by `flux plugin update`.

### User Stories

#### Flux User Installs a Plugin

As a Flux user, I want to install the Flux Operator CLI as a plugin so that I can
manage Flux instances using `flux operator` instead of a separate `flux-operator` binary.

```bash
flux plugin install operator
flux operator get instance -n flux-system
```

#### Flux User Updates Plugins

As a Flux user, I want to update all my installed plugins to the latest versions
with a single command.

```bash
flux plugin update
```

#### Flux User Symlinks a Python Plugin

As a Flux user, I want to use [flux-local](https://github.com/allenporter/flux-local)
(a Python tool) as a Flux CLI plugin by symlinking it into the plugin directory.
Since flux-local is not a Go binary distributed via the catalog, I install it with
pip and register it manually.

```bash
uv venv
source .venv/bin/activate
uv pip install flux-local
ln -s "$(pwd)/.venv/bin/flux-local" ~/.fluxcd/plugins/flux-local
flux local test
```

Manually symlinked plugins show `manual` in `flux plugin list` and are skipped by
`flux plugin update`.

#### Flux User Discovers Available Plugins

As a Flux user, I want to search for available plugins so that I can extend my
Flux CLI with community tools.

```bash
flux plugin search
```

#### Plugin Author Publishes a Plugin

As a plugin author, I want to submit my tool to the Flux plugin catalog so that
Flux users can install it with `flux plugin install <name>`.

1. Release binary with GoReleaser (produces tarballs/zips + checksums)
2. Submit a PR to `fluxcd/plugins` with `plugins/<name>.yaml`
3. Subsequent releases are picked up by automated polling workflows

Plugin authors are responsible for maintaining their plugin definitions in the catalog,
by responding to issues and approving PRs for updates.

### Alternatives

#### PATH-based Discovery (kubectl model)

kubectl discovers plugins by scanning `$PATH` for `kubectl-*` executables. This is
simple but has drawbacks:

- Scanning the entire PATH is slow on some systems
- No control over what's discoverable (any `flux-*` binary on PATH becomes a plugin)
- No install/update mechanism built in (requires a separate tool like krew)

The single-directory approach is faster, more predictable, and integrates install/update
directly into the CLI.

## Design Details

### Package Structure

```
internal/plugin/
  discovery.go        # Plugin dir scanning, DI-based Handler
  completion.go       # Shell completion via Cobra __complete protocol
  exec_unix.go        # syscall.Exec (//go:build !windows)
  exec_windows.go     # os/exec fallback (//go:build windows)
  catalog.go          # Catalog fetching, manifest parsing, version/platform resolution
  install.go          # Download, verify, extract, receipts
  update.go           # Compare receipts vs catalog, update check

cmd/flux/
  plugin.go           # Cobra command registration, all plugin subcommands
```

The `internal/plugin` package uses dependency injection (injectable `ReadDir`, `Stat`,
`GetEnv`, `HomeDir` on a `Handler` struct) for testability. Tests mock these functions
directly without filesystem fixtures.

### Plugin Directory

- **Default**: `~/.fluxcd/plugins/` -- auto-created by install/update commands
  (best-effort, no error if filesystem is read-only).
- **Override**: `FLUXCD_PLUGINS` env var replaces the default directory path.
  When set, the CLI does not auto-create the directory.

### Startup Behavior

`registerPlugins()` is called in `main()` before `rootCmd.Execute()`. It scans the
plugin directory and registers discovered plugins as Cobra subcommands. The scan is
lightweight (a single `ReadDir` call) and only occurs if the plugin directory exists.
Built-in commands always take priority.

### Manifest Validation

Both plugin manifests and the catalog are validated after fetching:

- `apiVersion` must be `cli.fluxcd.io/v1beta1`
- `kind` must be `Plugin` or `PluginCatalog` respectively
- Checksum format is `<algorithm>:<hex>` (currently `sha256:...`), allowing future
  algorithm migration without schema changes

### Security Considerations

- **Checksum verification**: All downloaded archives are verified against SHA-256
  checksums declared in the catalog manifest before extraction.
- **Path traversal protection**: Archive extraction guards against tar traversal.
- **Response size limits**: HTTP responses from the catalog are capped at 10 MiB to
  prevent unbounded memory allocation from malicious servers.
- **No code execution during discovery**: Plugin directory scanning only reads directory
  entries and file metadata. No plugin binary is executed during startup.
- **Retryable fetching**: All HTTP/S operations use automatic retries for transient network failures.

### Catalog Repository CI

The `fluxcd/plugins` repository includes CI workflows that:

1. Validate plugin manifests on every PR (schema, name consistency, URL reachability,
   checksum verification, binary presence in archives, no builtin collisions)
2. Regenerate `catalog.yaml` when plugins are added or removed
3. Automatically poll upstream repositories for new releases and create update PRs
4. Plugin authors have to agree to maintain their plugin's definition by responding to issues and approving PRs in the catalog repo.

### Known Limitations (v1beta1)

1. **No cosign/SLSA verification** -- SHA-256 only. Signature verification can be added later.
2. **No plugin dependencies** -- plugins are standalone binaries.
3. **No automatic update checks** -- users run `flux plugin update` explicitly.
4. **No private catalog auth** -- `$FLUXCD_PLUGIN_CATALOG` works for private URLs but no token injection.
5. **No version constraints** -- no `>=0.44.0` ranges. Exact version or latest only.
6. **Flag names differ between Flux and plugins** -- e.g., `--context` (flux) vs
   `--kube-context` (flux-operator). This is a plugin concern, not a system concern.

## Implementation History

- **2026-03-30** PoC plugin catalog repository with example manifests and CI validation workflows available at [fluxcd/plugins](https://github.com/fluxcd/plugins).
