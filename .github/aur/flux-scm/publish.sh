#!/usr/bin/env bash

set -e

WD=$(cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd)
PKGNAME=$(basename $WD)
ROOT=${WD%/.github/aur/$PKGNAME}

export VERSION=$1
echo "Publishing to AUR as version ${VERSION}"

cd $WD

export GIT_SSH_COMMAND="ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no"

eval $(ssh-agent -s)
ssh-add <(echo "$AUR_BOT_SSH_PRIVATE_KEY")

rm -rf .pkg
git clone aur@aur.archlinux.org:$PKGNAME .pkg 2>&1

CURRENT_PKGVER=$(cat .pkg/.SRCINFO | grep pkgver | awk '{ print $3 }')
CURRENT_PKGREL=$(cat .pkg/.SRCINFO | grep pkgrel | awk '{ print $3 }')

export PKGVER=${VERSION/-/}

if [[ "${CURRENT_PKGVER}" == "${PKGVER}" ]]; then
    export PKGREL=$((CURRENT_PKGREL+1))
else
    export PKGREL=1
fi

envsubst '$PKGVER $PKGREL' < .SRCINFO.template > .pkg/.SRCINFO
envsubst '$PKGVER $PKGREL' < PKGBUILD.template > .pkg/PKGBUILD

cd .pkg
git config user.name "fluxcdbot"
git config user.email "fluxcdbot@users.noreply.github.com"
git add -A
if [ -z "$(git status --porcelain)" ]; then
  echo "No changes."
else
  git commit -m "Updated to version v${PKGVER} release ${PKGREL}"
  git push origin master
fi
