#!/bin/bash

set -e

# build.sh

OSNAME="$1"

if [ "$OSNAME" == "help" ]; then
  echo "Build NRT"
  echo " - Requirements:"
  echo "    . Access to nsip github"
  echo "    . go - version 1.14 or above"
  echo "    . Utilities - git, zip, unzip, curl"
  echo " - NRT version is detected using tools/release and tags, taken from nsip github"
  echo " - OS is auto detect, or can be provided:"
  echo "    . Linux64"
  echo "    . Mac"
  echo "    . Windows64"
  echo ""
  exit 1
fi

DEPENDS=("git" "zip" "unzip" "curl")
for b in ${DEPENDS[@]}; do
  if ! [ -x "$(command -v $b)" ]; then
        echo "Missing utilitiy: $b"
        exit 1
  fi
done

go get gopkg.in/cheggaaa/pb.v1
cd tools; go build release.go; cd ..

VERSION="$(./tools/release dev-nrt version)"

if [ -z "$VERSION" ]; then
  echo "Cannot determine version from repo."
  exit 1
fi
echo "NRT-BUILD: VERSION=$VERSION"

# Specific branches
# NRT_BRANCH="master" # always for formal release
NRT_BRANCH="add-splitter" # for testing/alphas etc.


ORIGINALPATH=`pwd`

GOARCH=amd64
LDFLAGS="-s -w"
export GOARCH

# Simplified - assumes all 64 bit for now
if [ "$OSNAME" == "" ]; then
    case "$OSTYPE" in
      linux*)   OSNAME="Linux64" ;;
      Linux*)   OSNAME="Linux64" ;;
      darwin*)  OSNAME="Mac" ;;
      Darwin*)  OSNAME="Mac" ;;
      win*)     OSNAME="Windows64" ;;
      Win*)     OSNAME="Windows64" ;;
    esac
fi

if [ "$OSNAME" == "" ]; then
  echo "Unknown operating system from $OSTYPE"
  exit 1
fi

GOOS=""
case "$OSNAME" in
  Linux64) GOOS="linux" ;;
  Mac) GOOS="darwin" ;;
  Windows64) GOOS="windows" ;;
esac
export GOOS

EXT=""
case "$OSNAME" in
  Windows64) EXT=".exe" ;;
esac


# creates a all build of the NRT components and combined into
# ZIP files for release.

echo "NRT-BUILD: Start - Building NRT-$OSNAME-$VERSION"

echo "NRT-BUILD NOTE: removing existing builds"
rm -rf build

mkdir -p build

# echo "NRTBUILD: Static Code"
cd "$ORIGINALPATH"

echo "NRT-BUILD: Creating NRT @ $NRT_BRANCH"
cd cmd/nrt
echo "NRT-BUILD: building NRT application"
go build -ldflags="$LDFLAGS"

# copy supporting files
rsync -av * "$ORIGINALPATH"/build/
rsync -av config "$ORIGINALPATH"/build/
rsync -av config_split "$ORIGINALPATH"/build/
# remove golang files copied
rm "$ORIGINALPATH"/build/main.go || true

# include test data samples in distribution
cd "$ORIGINALPATH"
rsync -av testdata "$ORIGINALPATH"/build/

# include this project readme as minimal documentation
echo "NRT-BUILD: Documentation"
cp README.md build/

echo "NRT-BUILD: Generating ZIP"
cd build
rm ../NRT-$OSNAME-$VERSION.zip || true
zip -qr ../NRT-$OSNAME-$VERSION.zip *

echo "NRT-BUILD: Complete - NRT-$OSNAME-$VERSION.zip"

echo "If happy with zip, run the following release command"
echo " ./tools/release dev-nrt NRT-$OSNAME.zip NRT-$OSNAME-$VERSION.zip"

