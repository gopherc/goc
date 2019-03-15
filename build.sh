#!/bin/sh

if [ -z "$GOPHERC_VERSION" ]; then
    echo WARNING: No GOPHERC_VERSION variable!
    export GOPHERC_VERSION=0.0.1
fi
echo GopherC version: $GOPHERC_VERSION

export BUILD_DIR=$PWD

echo "[Building Go]"

cd go
rm -rf bin
rm -rf pkg

if [ -z "$GOROOT" ]; then
    echo GOROOT was not set!
    export GOROOT=/usr/local/go
fi

export GOROOT_BOOTSTRAP=$GOROOT
export CGO_ENABLED=0

cd src
./make.bash
cd ../..

echo "[Building GopherC]"

cd cmd/goc
rm -f goc

echo "package version" > version/version.go
echo "var Version = \"$GOPHERC_VERSION\"" >> version/version.go

export GOROOT=$BUILD_DIR/go
$GOROOT/bin/go build goc.go
cd ../..

echo "[Building WABT]"

cd wabt
rm -rf build
rm -rf bin

mkdir build
cd build

cmake ..
make

cd ..
rm -rf build
cd ..

echo "[Done]"