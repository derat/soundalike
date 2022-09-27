#!/bin/sh -e

set -x

if [ $# -ne 3 ]; then
  echo "Usage: $0 <os> <arch> <version>" >&2
  exit 2
fi

export GOOS=$1
export GOARCH=$2
export CGO_ENABLED=1

# Strip off the leading 'v' from tag names like 'v0.1'.
version=${3#v}

# See https://github.com/mattn/go-sqlite3/issues/303.
if [ "$GOOS" = windows ]; then
  export CC=x86_64-w64-mingw32-gcc
  deps="mingw-w64 zip"
fi

# Install dependencies here instead of in release.yaml since changes
# outside of the workspace don't persist across build steps.
if [ -n "$deps" ] && [ "$(id -u)" -eq 0 ]; then
  apt-get update && apt-get install -y $deps
fi

go build -ldflags "-X main.buildVersion=${version}"

archive=soundalike-${version}-${GOOS}-${GOARCH}
files="README.md LICENSE"
if [ "$GOOS" = windows ]; then
  zip "${archive}.zip" soundalike.exe $files
  rm soundalike.exe
else
  tar -czvf "${archive}.tar.gz" soundalike $files
  rm soundalike
fi
