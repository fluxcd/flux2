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

# This script installs the prerequisites for running Flux end-to-end tests with Docker and GitHub self-hosted runners.

set -eu

KIND_VERSION=0.22.0
KUBECTL_VERSION=1.29.0
KUSTOMIZE_VERSION=5.3.0
HELM_VERSION=3.14.1
GITHUB_RUNNER_VERSION=2.313.0
PACKAGES="apt-transport-https ca-certificates software-properties-common build-essential libssl-dev gnupg lsb-release jq pkg-config"

# install prerequisites
apt-get update \
  && apt-get install -y -q ${PACKAGES} \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*

# fix Kubernetes DNS resolution
rm /etc/resolv.conf
cat "/run/systemd/resolve/stub-resolv.conf" | sed '/search/d' > /etc/resolv.conf

# install docker
curl -fsSL https://get.docker.com -o get-docker.sh \
  && chmod +x get-docker.sh
./get-docker.sh
systemctl enable docker.service
systemctl enable containerd.service
usermod -aG docker ubuntu

# install kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v${KIND_VERSION}/kind-linux-arm64
install -o root -g root -m 0755 kind /usr/local/bin/kind

# install kubectl
curl -LO "https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/arm64/kubectl"
install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# install kustomize
curl -Lo ./kustomize.tar.gz https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv${KUSTOMIZE_VERSION}/kustomize_v${KUSTOMIZE_VERSION}_linux_arm64.tar.gz \
  && tar -zxvf kustomize.tar.gz \
  && rm kustomize.tar.gz
install -o root -g root -m 0755 kustomize /usr/local/bin/kustomize

# install helm
curl -Lo ./helm.tar.gz https://get.helm.sh/helm-v${HELM_VERSION}-linux-arm64.tar.gz \
  && tar -zxvf helm.tar.gz \
  && rm helm.tar.gz
install -o root -g root -m 0755 linux-arm64/helm /usr/local/bin/helm

# download runner
curl -o actions-runner-linux-arm64.tar.gz -L https://github.com/actions/runner/releases/download/v${GITHUB_RUNNER_VERSION}/actions-runner-linux-arm64-${GITHUB_RUNNER_VERSION}.tar.gz \
  && tar xzf actions-runner-linux-arm64.tar.gz \
  && rm actions-runner-linux-arm64.tar.gz

# install runner dependencies
./bin/installdependencies.sh
