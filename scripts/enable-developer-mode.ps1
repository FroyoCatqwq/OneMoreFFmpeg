#!/usr/bin/env pwsh
# Enable Windows developer mode so directory symlinks can be created without
# elevation (Win10 1809+ / Win11). Must be run elevated, e.g.:
#   pwsh scripts/elevate.ps1 pwsh -NoProfile -File scripts/enable-developer-mode.ps1
# This modifies HKLM; running it is an explicit administrator action.
$key = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\AppModelUnlock'
if (-not (Test-Path $key)) { New-Item -Path $key -Force | Out-Null }
New-ItemProperty -Path $key -Name AllowDevelopmentWithoutDevLicense -Value 1 -PropertyType DWord -Force | Out-Null
New-ItemProperty -Path $key -Name AllowAllTrustedApps -Value 1 -PropertyType DWord -Force | Out-Null
Write-Output "developer mode enabled (AllowDevelopmentWithoutDevLicense=1)"