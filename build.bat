@echo off
set BUILD_DIR=%~dp0

echo [Building Go]

cd go
if exist bin\ rd /S /Q bin
if exist pkg\ rd /S /Q pkg

set GOROOT=C:\Go
set GOROOT_BOOTSTRAP=%GOROOT%
set CGO_ENABLED=0

cd src
call make.bat
cd ..\..

echo [Building GopherC]

cd cmd/goc
if exist goc.exe del /Q /F goc.exe

set GOROOT=%BUILD_DIR%\go
%GOROOT%\bin\go build goc.go
cd ..\..

echo [Building WABT]

cd wabt
if exist build\ rd /S /Q build
if exist bin\ rd /S /Q bin

mkdir build
cd build

cmake ../ -DCMAKE_BUILD_TYPE=release -DCMAKE_INSTALL_PREFIX=../ -G "MinGW Makefiles" -DBUILD_TESTS=OFF
cmake --build . --config release

cd ..
rd /S /Q build
cd ..

echo [Done!]