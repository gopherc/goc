#!/bin/sh

if [ -z "$GOPHERC_VERSION" ]; then
    echo WARNING: No GOPHERC_VERSION variable!
    export GOPHERC_VERSION=0.0.1
fi
echo GopherC version: $GOPHERC_VERSION

export BUILD_DIR=$PWD

echo "[Package]"

export PACKAGE=gopherc${GOPHERC_VERSION}.linux-amd64.zip
rm -f gopherc*.linux-amd64.zip

cd ..
zip -r /tmp/$PACKAGE goc -x \*.git\*
cd $BUILD_DIR
mv /tmp/$PACKAGE .

echo "[Done]"