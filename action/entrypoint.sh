#!/bin/bash

#  Copyright 2020 The Flux authors
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

set -e

VERSION=$1

if [ -z $VERSION ]; then
  # Find latest release if no version is specified
  VERSION=$(curl https://api.github.com/repos/fluxcd/flux2/releases/latest -sL | grep tag_name | sed -E 's/.*"([^"]+)".*/\1/' | cut -c 2-)
fi

# Download linux binary
BIN_URL="https://github.com/fluxcd/flux2/releases/download/v${VERSION}/flux_${VERSION}_linux_amd64.tar.gz"
curl -sL $BIN_URL | tar xz

# Copy binary to GitHub runner
mkdir -p $GITHUB_WORKSPACE/bin
cp ./flux $GITHUB_WORKSPACE/bin
chmod +x $GITHUB_WORKSPACE/bin/flux

# Print version
$GITHUB_WORKSPACE/bin/flux -v

# Add binary to GitHub runner path
echo "$GITHUB_WORKSPACE/bin" >> $GITHUB_PATH
echo "$RUNNER_WORKSPACE/$(basename $GITHUB_REPOSITORY)/bin" >> $GITHUB_PATH
