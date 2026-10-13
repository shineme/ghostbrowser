param([string]$PhpPath = '', [int]$Port = 8787)
$ErrorActionPreference = 'Stop'
if (-not $PhpPath) {
    $command = Get-Command php.exe -ErrorAction SilentlyContinue
    if ($command) { $PhpPath = $command.Source }
    else { $PhpPath = Join-Path $env:USERPROFILE 'Tools\php-8.5.11\php.exe' }
}
if (-not (Test-Path -LiteralPath $PhpPath)) { throw 'Pass -PhpPath with the path to php.exe.' }
if (-not $env:GHOST_SYNC_TOKEN -and -not (Test-Path -LiteralPath (Join-Path $PSScriptRoot 'config.php'))) {
    throw 'Set GHOST_SYNC_TOKEN or create config.php from config.example.php first.'
}
Set-Location -LiteralPath $PSScriptRoot
$extensionDir = Join-Path (Split-Path -Parent $PhpPath) 'ext'
& $PhpPath -d "extension_dir=$extensionDir" -d extension=pdo_sqlite -S "127.0.0.1:$Port" -t public router.php
