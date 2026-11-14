#!/usr/bin/env pwsh
# External-caller example (PowerShell): consume OneMoreFFmpeg's CLI JSON contract.
# Needs only the published binary; no Go import / no Go runtime linkage.
#
# Usage:
#   $env:OMF_BINARY = "C:\path\one-more-ffmpeg.exe"; ./call_omf.ps1
#   ./call_omf.ps1 -Binary C:\path\one-more-ffmpeg.exe
param([string]$Binary = $env:OMF_BINARY)
if (-not $Binary) { $Binary = "one-more-ffmpeg" }

function Invoke-Omf {
    param([string[]]$Arguments)
    $out = & $Binary @Arguments --json 2>$null
    $envelope = $out | ConvertFrom-Json
    if ($envelope.schema -eq "onemoreffmpeg.error/v1" -or -not $envelope.ok) {
        throw "$($envelope.code): $($envelope.message)"
    }
    if ($envelope.schema -ne "onemoreffmpeg.result/v1") {
        throw "unexpected envelope schema: $($envelope.schema)"
    }
    return $envelope.data
}

$version = Invoke-Omf @("version")
Write-Output "OneMoreFFmpeg $($version.version) ($($version.os)/$($version.arch))"

$report = Invoke-Omf @("doctor")
Write-Output "healthy: $($report.healthy)"
foreach ($check in $report.checks) {
    Write-Output "  - $($check.name): $($check.ok) ($($check.detail))"
}

$detected = Invoke-Omf @("detect")
if (-not $detected.installations -or $detected.installations.Count -eq 0) {
    Write-Output "no FFmpeg discovered; install one first (see docs/HOWTO.md)"
    exit 0
}
$root = $detected.installations[0].root_dir
$encoders = Invoke-Omf @("capabilities", "--dir", $root, "--only", "encoders", "--no-cache")
Write-Output "encoders at ${root}: $($encoders.entries.Count)"