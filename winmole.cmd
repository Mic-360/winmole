@echo off
set BIN=%~dp0winmole.exe
if exist "%BIN%" (
  "%BIN%" %*
  exit /b %ERRORLEVEL%
)
powershell.exe -ExecutionPolicy Bypass -File "%~dp0wimo.ps1" %*
