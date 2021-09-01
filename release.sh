#!/bin/bash
set -eu -o pipefail

cd "$(dirname "$0")"

version="$1"

if [[ "$(git rev-list -n 1 $version)" != "$(git rev-parse HEAD)" ]]; then
  echo "Not currently on version $version" 2>&1
  exit 1
fi

rm -rf release
mkdir release

for goos in linux darwin windows; do
  for goarch in amd64 arm64; do
    dir="release/cljfmt_${goos}_${goarch}"
    mkdir "$dir"
    cp LICENSE.txt "${dir}/LICENSE.txt"
    GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 go build -o "${dir}/cljfmt" github.com/cespare/goclj/cljfmt
    tar -c -f - -C release "$(basename "$dir")" | gzip -9 >"${dir}.tar.gz"
    rm -rf "${dir}"
    sha256sum "${dir}.tar.gz" >"${dir}.tar.gz.sha256"
  done
done

exec gh release create "$version" \
  --title "Cljfmt ${version#v}" \
  ./release/*.tar.gz \
  ./release/*.tar.gz.sha256
