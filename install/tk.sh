#!/usr/bin/env bash

set -e

DEFAULT_BIN_DIR="/usr/local/bin"
BIN_DIR=${1:-"$DEFAULT_BIN_DIR"}

opsys=""
if [[ "$OSTYPE" == linux* ]]; then
  opsys=linux
elif [[ "$OSTYPE" == darwin* ]]; then
  opsys=darwin
fi

if [[ "$opsys" == "" ]]; then
  echo "OS $OSTYPE not supported"
  exit 1
fi

if [[ ! -x "$(command -v curl)" ]]; then
    echo "curl not found"
    exit 1
fi

tmpDir=`mktemp -d`
if [[ ! "$tmpDir" || ! -d "$tmpDir" ]]; then
  echo "could not create temp dir"
  exit 1
fi

function cleanup {
  rm -rf "$tmpDir"
}

trap cleanup EXIT

pushd $tmpDir >& /dev/null

curl -s https://api.github.com/repos/fluxcd/toolkit/releases/latest |\
  grep browser_download |\
  grep $opsys |\
  cut -d '"' -f 4 |\
  xargs curl -sL -o tk.tar.gz

tar xzf ./tk.tar.gz

mv ./tk $BIN_DIR

popd >& /dev/null

echo "$(tk --version) installed"
