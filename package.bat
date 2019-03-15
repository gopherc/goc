@echo off

if "%GOPHERC_VERSION%"=="" (
    echo WARNING: No GOPHERC_VERSION variable!
    set GOPHERC_VERSION=0.0.1
)
echo GopherC version: %GOPHERC_VERSION%

echo [Create Installer]

set INSTALLER=gopherc%GOPHERC_VERSION%.windows-amd64.exe
if exist %INSTALLER% del /Q /F %INSTALLER%
iscc /Qp /O".\" package\windows\setup.iss

echo [Done!]