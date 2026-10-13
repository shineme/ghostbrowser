param([string]$PhpPath = '', [int]$Port = 8787)
# Local test server for the single-file sync backend.
# Then use http://127.0.0.1:8787/ghostsync.php as the sync server in GhostBrowser.
$ErrorActionPreference = 'Stop'
if (-not $PhpPath) {
    $command = Get-Command php.exe -ErrorAction SilentlyContinue
    if ($command) { $PhpPath = $command.Source } else { throw 'Pass -PhpPath with the path to php.exe.' }
}
Set-Location -LiteralPath $PSScriptRoot
& $PhpPath -S "127.0.0.1:$Port" -t .
