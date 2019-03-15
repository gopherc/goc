@echo off
set BUILD_DIR=%~dp

if "%GOPHERC_VERSION%"=="" (
    echo WARNING: No GOPHERC_VERSION variable!
    set GOPHERC_VERSION=0.0.1
)
echo GopherC version: %GOPHERC_VERSION%

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

echo package version > version\version.go
echo var Version = %GOPHERC_VERSION% >> version\version.go

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