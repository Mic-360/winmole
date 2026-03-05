# Winmole installer
param(
    [string]$InstallDir = "$env:LOCALAPPDATA\WiMo",
    [switch]$AddToPath = $true,
    [switch]$CreateShortcut = $false,
    [switch]$BuildFromSource = $true
)

$ErrorActionPreference = 'Stop'
$binaryName = 'winmole.exe'
$sourceBinary = Join-Path $PSScriptRoot 'bin\winmole.exe'
$installBinary = Join-Path $InstallDir $binaryName

Write-Host ''
Write-Host '  Winmole installer'
Write-Host ''

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

if ($BuildFromSource -and (Get-Command go -ErrorAction SilentlyContinue)) {
    Push-Location $PSScriptRoot
    try {
        go build -ldflags='-s -w' -o $installBinary .
    } finally {
        Pop-Location
    }
} elseif (Test-Path $sourceBinary) {
    Copy-Item $sourceBinary $installBinary -Force
} else {
    throw 'No built winmole.exe found and Go was not available to build from source.'
}

Copy-Item (Join-Path $PSScriptRoot 'wimo.ps1') (Join-Path $InstallDir 'wimo.ps1') -Force
Copy-Item (Join-Path $PSScriptRoot 'wimo.cmd') (Join-Path $InstallDir 'wimo.cmd') -Force
Copy-Item (Join-Path $PSScriptRoot 'winmole.cmd') (Join-Path $InstallDir 'winmole.cmd') -Force

if ($AddToPath) {
    $currentPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($currentPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable('Path', "$currentPath;$InstallDir", 'User')
    }
}

if ($CreateShortcut) {
    $shell = New-Object -ComObject WScript.Shell
    $shortcutPath = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Winmole.lnk"
    $shortcut = $shell.CreateShortcut($shortcutPath)
    $shortcut.TargetPath = $installBinary
    $shortcut.Description = 'Winmole terminal maintenance dashboard'
    $shortcut.Save()
}

Write-Host '  Installed Winmole successfully.'
Write-Host '  Run: winmole'
Write-Host ''
