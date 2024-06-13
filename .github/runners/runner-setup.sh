#!/usr/bin/env bash

# Copyright 2021 The Flux authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script installs a GitHub self-hosted ARM64 runner for running Flux end-to-end tests.

set -eu

RUNNER_NAME=$1
REPOSITORY_TOKEN=$2
REPOSITORY_URL=${3:-https://github.com/fluxcd/flux2}

GITHUB_RUNNER_VERSION=2.313.0

# download runner
curl -o actions-runner-linux-arm64.tar.gz -L https://github.com/actions/runner/releases/download/v${GITHUB_RUNNER_VERSION}/actions-runner-linux-arm64-${GITHUB_RUNNER_VERSION}.tar.gz \
  && tar xzf actions-runner-linux-arm64.tar.gz \
  && rm actions-runner-linux-arm64.tar.gz

# register runner with GitHub
./config.sh --unattended --url ${REPOSITORY_URL} --token ${REPOSITORY_TOKEN} --name ${RUNNER_NAME}

# start runner
sudo ./svc.sh install
sudo ./svc.sh start
