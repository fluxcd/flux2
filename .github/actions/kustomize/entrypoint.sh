#!/bin/sh -l

VERSION=3.5.4
curl -sL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv${VERSION}/kustomize_v${VERSION}_linux_amd64.tar.gz | tar xz

mkdir -p $GITHUB_WORKSPACE/bin
cp ./kustomize $GITHUB_WORKSPACE/bin
chmod +x $GITHUB_WORKSPACE/bin/kustomize
ls -lh $GITHUB_WORKSPACE/bin

echo "::add-path::$GITHUB_WORKSPACE/bin"
echo "::add-path::$RUNNER_WORKSPACE/$(basename $GITHUB_REPOSITORY)/bin"
