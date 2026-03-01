# WiMo — install.ps1
# Windows installer (PATH, winget manifest, Start Menu shortcut)

param(
    [string]$InstallDir     = "$env:LOCALAPPDATA\WiMo",
    [switch]$AddToPath      = $true,
    [switch]$CreateShortcut = $false,
    [switch]$BuildFromSource = $false
)

$ErrorActionPreference = 'Stop'

# Minimal ANSI setup for installer output
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$e      = [char]0x1b
$green  = "$e[38;5;82m"
$cyan   = "$e[38;5;51m"
$orange = "$e[38;5;208m"
$grey   = "$e[38;5;245m"
$bold   = "$e[1m"
$reset  = "$e[0m"

Write-Host ""
Write-Host "  $bold${orange}🐹 WiMo Installer$reset"
Write-Host "  ${grey}Windows System Optimizer$reset"
Write-Host ""

# 1. Create install directory
Write-Host "  ${cyan}Creating install directory...$reset"
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
New-Item -ItemType Directory -Path "$InstallDir\bin" -Force | Out-Null
New-Item -ItemType Directory -Path "$InstallDir\lib\core" -Force | Out-Null

# 2. Build or download Go binaries
if ($BuildFromSource -and (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "  ${cyan}Building Go binaries from source...$reset"
    Push-Location $PSScriptRoot
    try {
        go build -ldflags="-s -w" -o "$InstallDir\bin\analyze-go.exe" ./cmd/analyze
        Write-Host "  ${green}✓$reset  Built analyze-go.exe"
        go build -ldflags="-s -w" -o "$InstallDir\bin\status-go.exe" ./cmd/status
        Write-Host "  ${green}✓$reset  Built status-go.exe"
    } catch {
        Write-Host "  ${orange}⚠$reset  Go build failed: $_"
        Write-Host "  ${grey}Go binaries will need to be built manually$reset"
    }
    Pop-Location
} else {
    Write-Host "  ${grey}Skipping Go binary build (use -BuildFromSource with Go installed)$reset"
}

# 3. Copy PowerShell scripts
Write-Host "  ${cyan}Installing PowerShell scripts...$reset"

# Core entry point
Copy-Item -Path "$PSScriptRoot\wimo.ps1" -Destination "$InstallDir\wimo.ps1" -Force

# Bin scripts
$binScripts = @("clean.ps1", "uninstall.ps1", "optimize.ps1", "purge.ps1", "analyze.ps1", "status.ps1")
foreach ($script in $binScripts) {
    $src = "$PSScriptRoot\bin\$script"
    if (Test-Path $src) {
        Copy-Item -Path $src -Destination "$InstallDir\bin\$script" -Force
    }
}

# Core libraries
$coreScripts = @("common.ps1", "base.ps1", "log.ps1", "file_ops.ps1", "ui.ps1")
foreach ($script in $coreScripts) {
    $src = "$PSScriptRoot\lib\core\$script"
    if (Test-Path $src) {
        Copy-Item -Path $src -Destination "$InstallDir\lib\core\$script" -Force
    }
}

Write-Host "  ${green}✓$reset  Scripts installed"

# 4. Create wimo.cmd wrapper
$cmdContent = @"
@echo off
powershell.exe -ExecutionPolicy Bypass -File "%~dp0wimo.ps1" %*
"@
Set-Content -Path "$InstallDir\wimo.cmd" -Value $cmdContent -Encoding ASCII
Write-Host "  ${green}✓$reset  Created wimo.cmd wrapper"

# 5. Add to PATH
if ($AddToPath) {
    $currentPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
    if ($currentPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable('PATH', "$currentPath;$InstallDir", 'User')
        Write-Host "  ${green}✓$reset  Added $InstallDir to PATH"
        Write-Host "  ${grey}  Restart your terminal to use 'wimo' command$reset"
    } else {
        Write-Host "  ${green}✓$reset  Already in PATH"
    }
}

# 6. Optional Start Menu shortcut
if ($CreateShortcut) {
    try {
        $shell = New-Object -ComObject WScript.Shell
        $shortcutPath = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\WiMo.lnk"
        $shortcut = $shell.CreateShortcut($shortcutPath)
        $shortcut.TargetPath = "powershell.exe"
        $shortcut.Arguments  = "-ExecutionPolicy Bypass -File `"$InstallDir\wimo.ps1`""
        $shortcut.Description = "WiMo — Windows System Optimizer"
        $shortcut.Save()
        Write-Host "  ${green}✓$reset  Created Start Menu shortcut"
    } catch {
        Write-Host "  ${orange}⚠$reset  Could not create Start Menu shortcut: $_"
    }
}

# 7. Create default config
$configDir = "$env:APPDATA\WiMo"
$configFile = "$configDir\config.json"
if (-not (Test-Path $configFile)) {
    New-Item -ItemType Directory -Path $configDir -Force | Out-Null
    $defaultConfig = @{
        version        = "1.0.0"
        whitelist      = @()
        scan_paths     = @("$env:USERPROFILE\Projects", "$env:USERPROFILE\Documents\dev")
        purge_depth    = 8
        theme          = "default"
        check_updates  = $true
        winget_enabled = $true
    } | ConvertTo-Json -Depth 4
    Set-Content -Path $configFile -Value $defaultConfig -Encoding UTF8
    Write-Host "  ${green}✓$reset  Created default config"
}

Write-Host ""
Write-Host "  ${bold}${green}🐹 WiMo installed successfully!$reset"
Write-Host "  ${cyan}Run: wimo$reset"
Write-Host ""
