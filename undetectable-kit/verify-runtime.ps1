$ErrorActionPreference = 'Stop'
$manifest = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'runtime-manifest.json') -Raw | ConvertFrom-Json
$checked = 0
foreach ($entry in $manifest.files) {
    $path = Join-Path $PSScriptRoot $entry.path
    $item = Get-Item -LiteralPath $path
    $hash = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($item.Length -ne $entry.size -or $hash -ne $entry.sha256) {
        throw "Runtime file differs: $($entry.path)"
    }
    $checked++
}
Write-Output "Verified $checked runtime files. Chromium $($manifest.browserVersion)."
