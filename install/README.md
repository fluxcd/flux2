# GOTK CLI Installation

Binaries for macOS and Linux AMD64 are available for download on the 
[release page](https://github.com/fluxcd/toolkit/releases).

To install the latest release run:

```bash
curl -s https://raw.githubusercontent.com/fluxcd/toolkit/master/install/gotk.sh | sudo bash
```

The install script does the following:
* attempts to detect your OS
* downloads and unpacks the release tar file in a temporary directory
* copies the gotk binary to `/usr/local/bin`
* removes the temporary directory

If you want to use gotk as a kubectl plugin, rename the binary to `kubectl-gotk`:

```sh
mv /usr/local/bin/gotk /usr/local/bin/kubectl-gotk
```

## Build from source

Clone the repository:

```bash
git clone https://github.com/fluxcd/toolkit
cd toolkit
```

Build the `gotk` binary (requires go >= 1.14):

```bash
make build
```

Run the binary:

```bash
./bin/gotk -h
```

## Installation via Homebrew

MacOS users (and Linux users who choose to adopt Homebrew) may install the
`gotk` CLI via Homebrew as well. Installation instructions and prerequisites for
Homebrew are documented on that tool's [home page](https://brew.sh/).

``` bash
brew tap fluxcd/toolkit
# Install latest tagged release
brew install fluxcd/toolkit/gotk
# Install from source
brew install --HEAD fluxcd/toolkit/gotk
```

Updated definitions for installing newer releases of `gotk` can be fetched once
this repository has updated the formula by executing `brew update`. The
availability of a possible upgrade can be checked using `brew outdated`. If a
newer version is available, it may be installed using `brew upgrade
fluxcd/toolkit/gotk` or prevented using `brew pin`. Please see the instructions
for more details.

### Uninstalling via Homebrew

``` bash
brew uninstall fluxcd/toolkit/gotk
brew untap fluxcd/toolkit
```
