#!/usr/bin/env pwsh
# Run a command with elevated privileges. Prefers Windows sudo (Win11 24H2+),
# then gsudo, then an interactive RunAs (UAC) prompt. This is the manual
# authorization step for sandboxed environments where junction/symlink creation
# is blocked: the user consciously chooses to elevate.
#
# Usage: pwsh scripts/elevate.ps1 <command> [args...]
# Example: pwsh scripts/elevate.ps1 pwsh -NoProfile -Command "go test ./internal/platform/"
param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Cmd)

if (-not $Cmd -or $Cmd.Count -eq 0) {
    Write-Error "usage: elevate.ps1 <command> [args...]"
    exit 2
}
$exe = $Cmd[0]
$rest = @()
if ($Cmd.Count -gt 1) { $rest = $Cmd[1..($Cmd.Count - 1)] }

if (Get-Command sudo -ErrorAction SilentlyContinue) {
    & sudo $exe @rest
    exit $LASTEXITCODE
}
if (Get-Command gsudo -ErrorAction SilentlyContinue) {
    & gsudo $exe @rest
    exit $LASTEXITCODE
}
$proc = Start-Process -FilePath $exe -ArgumentList $rest -Verb RunAs -Wait -PassThru
exit $proc.ExitCode