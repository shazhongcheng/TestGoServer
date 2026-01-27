@echo off
setlocal

REM protoc 路径
set PROTOC=protoc.exe

REM proto 源目录
set PROTO_DIR=..\internal\protocol\proto

REM Python 输出目录
set OUT_DIR=..\..\Client\internal_pb

echo ===== Generating Python Protos =====

if not exist %OUT_DIR% (
    mkdir %OUT_DIR%
)

for %%f in (%PROTO_DIR%\*.proto) do (
    echo Generating %%~nxf
    %PROTOC% ^
        --proto_path=%PROTO_DIR% ^
        --python_out=%OUT_DIR% ^
        %%f
)

echo ===== Python Proto Generate Finished =====
pause
