$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath $PSScriptRoot
& (Join-Path $PSScriptRoot 'bin\ghost-launcher.exe') -config (Join-Path $PSScriptRoot 'examples\native-profile.json')
