#!/usr/bin/env bash

# Copyright 2020, 2021 The Flux authors. All rights reserved.
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

set -e

IN_PATH=${1:-"$(realpath $(dirname "${BASH_SOURCE[0]}")/../..)/manifests"}
OUT_PATH=${2:-"$(realpath $(dirname "${BASH_SOURCE[0]}")/../..)/cmd/flux/manifests"}
TAR=${3}

info() {
    echo '[INFO] ' "$@"
}

fatal() {
    echo '[ERROR] ' "$@" >&2
    exit 1
}

build() {
  info "building $(basename $2)"
  kustomize build "$1" > "$2"
}

if ! [ -x "$(command -v kustomize)" ]; then
  fatal 'kustomize is not installed'
fi

rm -rf $OUT_PATH
mkdir -p $OUT_PATH
files=""

info using "$(kustomize version --short)"

# build controllers
for controller in ${IN_PATH}/bases/*/; do
    output_path="${OUT_PATH}/$(basename $controller).yaml"
    build $controller $output_path
    files+=" $(basename $output_path)"
done

# build rbac
rbac_path="${IN_PATH}/rbac"
rbac_output_path="${OUT_PATH}/rbac.yaml"
build $rbac_path $rbac_output_path
files+=" $(basename $rbac_output_path)"

# build policies
policies_path="${IN_PATH}/policies"
policies_output_path="${OUT_PATH}/policies.yaml"
build $policies_path $policies_output_path
files+=" $(basename $policies_output_path)"

# create tarball
if [[ -n $TAR ]];then
  info "archiving $TAR"
  cd ${OUT_PATH} && tar -czf $TAR $files
fi
