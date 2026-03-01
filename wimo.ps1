#!/usr/bin/env pwsh
# WiMo — Windows Mole System Optimizer
# Invoke as: wimo [command] [flags]

Set-StrictMode -Version Latest
$ErrorActionPreference = 'SilentlyContinue'

# UTF-8 + ANSI colors
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8
chcp 65001 | Out-Null

# Enable ANSI Virtual Terminal Processing (Windows 10+)
Add-Type -MemberDefinition @'
[DllImport("kernel32.dll")] public static extern bool SetConsoleMode(IntPtr h, uint m);
[DllImport("kernel32.dll")] public static extern bool GetConsoleMode(IntPtr h, out uint m);
[DllImport("kernel32.dll")] public static extern IntPtr GetStdHandle(int h);
'@ -Name 'WiMoK32' -Namespace 'Win32' -PassThru -ErrorAction SilentlyContinue | Out-Null

try {
    $handle = [Win32.WiMoK32]::GetStdHandle(-11)
    $mode = 0
    [Win32.WiMoK32]::GetConsoleMode($handle, [ref]$mode) | Out-Null
    [Win32.WiMoK32]::SetConsoleMode($handle, $mode -bor 0x0004) | Out-Null  # ENABLE_VIRTUAL_TERMINAL_PROCESSING
} catch {}

# Hide cursor during rendering, restore on exit
[Console]::CursorVisible = $false
Register-EngineEvent PowerShell.Exiting -Action { [Console]::CursorVisible = $true } | Out-Null

# Dot-source core libraries
. "$PSScriptRoot\lib\core\common.ps1"

# Check for --debug flag in args
$debugMode = $false
$filteredArgs = @()
foreach ($a in $args) {
    if ($a -eq '--debug') {
        $debugMode = $true
    } else {
        $filteredArgs += $a
    }
}

Initialize-WimoLog -Debug:$debugMode

# Restore cursor before dispatching
[Console]::CursorVisible = $true

# Parse args and dispatch
$command = if ($filteredArgs.Count -gt 0) { $filteredArgs[0] } else { $null }
$remaining = if ($filteredArgs.Count -gt 1) { $filteredArgs[1..($filteredArgs.Count - 1)] } else { @() }

switch ($command) {
    'clean'     { & "$PSScriptRoot\bin\clean.ps1" $remaining }
    'uninstall' { & "$PSScriptRoot\bin\uninstall.ps1" $remaining }
    'optimize'  { & "$PSScriptRoot\bin\optimize.ps1" $remaining }
    'analyze'   { & "$PSScriptRoot\bin\analyze.ps1" $remaining }
    'status'    { & "$PSScriptRoot\bin\status.ps1" $remaining }
    'purge'     { & "$PSScriptRoot\bin\purge.ps1" $remaining }
    'update'    { Invoke-WimoUpdate }
    'remove'    { Invoke-WimoRemove }
    '--version' { Show-Version }
    '-v'        { Show-Version }
    '--help'    { Show-Help }
    '-h'        { Show-Help }
    default     { Show-InteractiveMenu }
}
