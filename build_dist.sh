#!/bin/bash
set -ex
cd control-panel
pnpm i
pnpm build
cd ..
rm -rf cmd/main/build
cp -r control-panel/build cmd/main

appname=pear-desktop-twitch-song-requests
archfname=(amd64 arm64)
osfname=(windows linux darwin)

for os in "${osfname[@]}"; do
    for arch in "${archfname[@]}"; do
        execname=${appname}_${os}_${arch}
        if [ "$os" = 'windows' ]; then execname=$execname.exe ; fi
        GOOS=$os GOARCH=$arch go build -ldflags="-s -w" -trimpath -o "$execname" cmd/main/main.go &
    done
done

wait
