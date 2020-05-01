# TK CLI Installation

Binaries for macOS and Linux AMD64 are available for download on the 
[release page](https://github.com/fluxcd/toolkit/releases).

To install the latest release run:

```bash
curl -s https://raw.githubusercontent.com/fluxcd/toolkit/master/install/tk.sh | sudo bash
```

The install script does the following:
* attempts to detect your OS
* downloads and unpacks the release tar file in a temporary directory
* copies the tk binary to `/usr/local/bin`
* removes the temporary directory

## Build from source

Clone the repository:

```bash
git clone https://github.com/fluxcd/toolkit
cd toolkit
```

Build the tk binary (requires go >= 1.14):

```bash
make build
```

Run the binary:

```bash
./bin/tk -h
```
