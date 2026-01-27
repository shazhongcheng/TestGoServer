@echo off
setlocal

set PROTOC=protoc.exe
set PROTO_DIR=..\internal\protocol\proto
set OUT_DIR=..\internal\protocol\internalpb

echo ===== Generating Go Protos =====

for %%f in (%PROTO_DIR%\*.proto) do (
    echo Generating %%~nxf
    %PROTOC% ^
        --proto_path=%PROTO_DIR% ^
        --go_out=%OUT_DIR% ^
        --go_opt=paths=source_relative ^
        %%f
)

echo ===== Go Proto Generate Finished =====
pause
