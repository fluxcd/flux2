# flux CLI Installation

Binaries for macOS and Linux AMD64 are available for download on the 
[release page](https://github.com/fluxcd/flux2/releases).

To install the latest release run:

```bash
curl -s https://raw.githubusercontent.com/fluxcd/flux2/main/install/flux.sh | sudo bash
```

**Note**: You may want to export the `GITHUB_TOKEN` environment variable using a [personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
to avoid GitHub API rate-limiting errors if executing the install script repeatedly during a short time frame.

The install script does the following:
* attempts to detect your OS
* downloads and unpacks the release tar file in a temporary directory
* copies the flux binary to `/usr/local/bin`
* removes the temporary directory

If you want to use flux as a kubectl plugin, rename the binary to `kubectl-flux`:

```sh
mv /usr/local/bin/flux /usr/local/bin/kubectl-flux
```

## Build from source

Clone the repository:

```bash
git clone https://github.com/fluxcd/flux2
cd flux2
```

Build the `flux` binary (requires go >= 1.15):

```bash
make build
```

Run the binary:

```bash
./bin/flux -h
```
