#!/bin/bash
set -ex
appname=pear-desktop-twitch-song-requests
archfname=(amd64 arm64)
osfname=(linux windows darwin)

for os in "${osfname[@]}"; do
    for arch in "${archfname[@]}"; do
        execname=${appname}_${os}_${arch}
        if [ "$os" = 'windows' ]; then execname=$execname.exe ; fi
        GOOS=$os GOARCH=$arch go build -ldflags="-s -w" -trimpath -o "$execname" cmd/main/main.go &
    done
done

wait
