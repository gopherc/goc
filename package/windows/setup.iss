#define GopherCAppName "GopherC"
#define GopherCAppVersion GetEnv('GOPHERC_VERSION')
#define GopherCAppPublisher "Andreas T Jonsson"
#define GopherCAppURL "https://github.com/gopherc/goc"
#define GopherCAppRoot "..\..\"

#include "path.iss"

[Setup]
AppId={{FB66B5A7-9126-41E3-AC58-1607C3FBAF87}
AppName={#GopherCAppName}
AppVersion={#GopherCAppVersion}
AppVerName={cm:NameAndVersion,{#GopherCAppName},{#GopherCAppVersion}}
AppPublisher={#GopherCAppPublisher}
AppPublisherURL={#GopherCAppURL}
AppSupportURL={#GopherCAppURL}
AppUpdatesURL={#GopherCAppURL}
DefaultDirName=C:\GopherC
CreateAppDir=yes
DisableProgramGroupPage=yes
OutputBaseFilename=gopherc{#GopherCAppVersion}.windows-amd64
SetupIconFile={#GopherCAppRoot}\package\windows\icon.ico
Compression=lzma
SolidCompression=yes
ArchitecturesInstallIn64BitMode=x64
ChangesEnvironment=yes

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Code]
procedure CurStepChanged(CurStep: TSetupStep);
begin
    if CurStep = ssPostInstall 
        then EnvAddPath(ExpandConstant('{app}') +'\cmd\goc');
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
    if CurUninstallStep = usPostUninstall
        then EnvRemovePath(ExpandConstant('{app}') +'\cmd\goc');
end;

[Registry]
Root: HKLM; Subkey: "SYSTEM\CurrentControlSet\Control\Session Manager\Environment"; ValueType:string; ValueName: "GOCROOT"; ValueData: "{app}"; Flags: preservestringtype uninsdeletevalue

[Files]
Source: "{#GopherCAppRoot}\cmd\*"; DestDir: "{app}\cmd"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#GopherCAppRoot}\doc\*"; DestDir: "{app}\doc"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#GopherCAppRoot}\go\*"; DestDir: "{app}\go"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#GopherCAppRoot}\runtime\*"; DestDir: "{app}\runtime"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#GopherCAppRoot}\wabt\*"; DestDir: "{app}\doc"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#GopherCAppRoot}\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#GopherCAppRoot}\README.md"; DestDir: "{app}"; Flags: ignoreversion
