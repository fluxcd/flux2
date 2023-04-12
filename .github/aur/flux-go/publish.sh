#!/usr/bin/env bash

set -e

WD=$(cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd)
PKGNAME=$(basename $WD)
ROOT=${WD%/.github/aur/$PKGNAME}

LOCKFILE=/tmp/aur-$PKGNAME.lock
exec 100>$LOCKFILE || exit 0
flock -n 100 || exit 0
trap "rm -f $LOCKFILE" EXIT

export VERSION=$1
echo "Publishing to AUR as version ${VERSION}"

cd $WD

export GIT_SSH_COMMAND="ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no"

eval $(ssh-agent -s)
ssh-add <(echo "$AUR_BOT_SSH_PRIVATE_KEY")

GITDIR=$(mktemp -d /tmp/aur-$PKGNAME-XXX)
trap "rm -rf $GITDIR" EXIT
git clone aur@aur.archlinux.org:$PKGNAME $GITDIR 2>&1

CURRENT_PKGVER=$(cat $GITDIR/.SRCINFO | grep pkgver | awk '{ print $3 }')
CURRENT_PKGREL=$(cat $GITDIR/.SRCINFO | grep pkgrel | awk '{ print $3 }')

# Transform pre-release to AUR compatible version format
export PKGVER=${VERSION/-/}

if [[ "${CURRENT_PKGVER}" == "${PKGVER}" ]]; then
    export PKGREL=$((CURRENT_PKGREL+1))
else
    export PKGREL=1
fi

export SHA256SUM=$(curl -sL https://github.com/fluxcd/flux2/archive/v${VERSION}.tar.gz | sha256sum | awk '{ print $1 }')

envsubst '$VERSION $PKGVER $PKGREL $SHA256SUM' < .SRCINFO.template > $GITDIR/.SRCINFO
envsubst '$VERSION $PKGVER $PKGREL $SHA256SUM' < PKGBUILD.template > $GITDIR/PKGBUILD

cd $GITDIR
git config user.name "fluxcdbot"
git config user.email "fluxcdbot@users.noreply.github.com"
git add -A
if [ -z "$(git status --porcelain)" ]; then
  echo "No changes."
else
  git commit -m "Updated to version v${PKGVER} release ${PKGREL}"
  git push origin master
fi
