# Stops a running PostmanJanai.exe then runs wails build for Windows.
# Use when you see: unlinkat ... PostmanJanai.exe: Access is denied

$ErrorActionPreference = "Continue"
Get-Process -Name "PostmanJanai" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 300

Set-Location $PSScriptRoot\..
wails build -clean -platform windows/amd64 -o PostmanJanai.exe
