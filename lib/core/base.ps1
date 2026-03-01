# WiMo — base.ps1
# Constants, ANSI color definitions, ASCII logo, version info

$script:WIMO_VERSION = "1.0.0"
$script:WIMO_CONFIG_DIR = "$env:APPDATA\WiMo"
$script:WIMO_CONFIG_FILE = "$script:WIMO_CONFIG_DIR\config.json"
$script:WIMO_LOG_DIR = "$script:WIMO_CONFIG_DIR\logs"

# ESC character compatible with PowerShell 5.1+
$script:ESC = [char]0x1b

# ANSI Color Palette
$script:C = @{
    Reset     = "$([char]0x1b)[0m"
    Bold      = "$([char]0x1b)[1m"
    Dim       = "$([char]0x1b)[2m"

    # Brand colors
    Orange    = "$([char]0x1b)[38;5;208m"
    Brown     = "$([char]0x1b)[38;5;130m"
    DarkBrown = "$([char]0x1b)[38;5;94m"
    Yellow    = "$([char]0x1b)[38;5;226m"
    Tan       = "$([char]0x1b)[38;5;180m"
    SageGreen = "$([char]0x1b)[38;5;108m"
    SageLight = "$([char]0x1b)[38;5;151m"
    Rose      = "$([char]0x1b)[38;5;174m"

    # UI colors
    Green     = "$([char]0x1b)[38;5;82m"
    Red       = "$([char]0x1b)[38;5;196m"
    Cyan      = "$([char]0x1b)[38;5;51m"
    Blue      = "$([char]0x1b)[38;5;39m"
    Purple    = "$([char]0x1b)[38;5;141m"
    Grey      = "$([char]0x1b)[38;5;245m"
    White     = "$([char]0x1b)[38;5;255m"
    LightGrey = "$([char]0x1b)[38;5;250m"

    # Backgrounds
    BgSelected = "$([char]0x1b)[48;5;235m"
    BgHeader   = "$([char]0x1b)[48;5;232m"
}

# Protected paths that must NEVER be deleted
$script:PROTECTED_PATHS = @(
    "C:\",
    "C:\Windows",
    "C:\Program Files",
    "C:\Program Files (x86)",
    "C:\Users",
    "C:\System Volume Information"
)

# User data folder patterns that must not be deleted
$script:USER_DATA_PATTERNS = @(
    "\Documents\",
    "\Desktop\",
    "\Downloads\",
    "\Pictures\",
    "\Videos\",
    "\Music\",
    "\OneDrive\"
)

# Mole ASCII Art — Full Banner (detailed cartoon mole)
$script:MOLE_LOGO = @(
    "      ██████              ██████"
    "    ██▓▓▓▓▓▓██          ██▓▓▓▓▓▓██"
    "    ██▓▓▓▓██████████████████▓▓▓▓██"
    "      ████░░░░░░░░░░░░░░░░░░████"
    "      ██░░░░░░░░░░░░░░░░░░░░░░██"
    "      ██░░░░░░ ◉ ░░░░ ◉ ░░░░░░██"
    "      ██░░░░░░░░░░░░░░░░░░░░░░██"
    "      ██░░░░░░░░░░▓▓░░░░░░░░░░██"
    "        ██░░░░░░░░░░░░░░░░░░██"
    "          ████░░░░░░░░░░████"
    "            ██████████████"
)

# Compact logo (4-line version)
$script:MOLE_LOGO_COMPACT = @(
    "    ████████        ████████"
    "  ██▓▓▓▓██████████████▓▓▓▓██"
    "  ██░░░░ ◉ ░░░░ ◉ ░░░░██"
    "    ██████████████████████"
)

function Get-TerminalWidth {
    try {
        $width = $Host.UI.RawUI.WindowSize.Width
        if ($width -gt 0) { return $width }
    } catch {}
    return 120
}

function Get-PlainTextLength {
    <#
    .SYNOPSIS
        Returns the visible length of a string after stripping ANSI escape codes.
    #>
    param([string]$Text)
    $plain = $Text -replace "$([char]0x1b)\[[0-9;]*m", ''
    return $plain.Length
}

function Center-Text {
    param([string]$Text, [int]$Width = 0)
    if ($Width -eq 0) { $Width = Get-TerminalWidth }
    $visibleLen = Get-PlainTextLength $Text
    $padding = [Math]::Max(0, [Math]::Floor(($Width - $visibleLen) / 2))
    return (' ' * $padding) + $Text
}

function Show-Banner {
    param([switch]$Compact)

    $width = Get-TerminalWidth

    if ($Compact) {
        Write-Host ""
        $maxLen = 0
        foreach ($l in $script:MOLE_LOGO_COMPACT) { if ($l.Length -gt $maxLen) { $maxLen = $l.Length } }
        foreach ($line in $script:MOLE_LOGO_COMPACT) {
            $colored = $line
            $colored = $colored -replace '◉', "$($C.Yellow)◉$($C.Rose)"
            $colored = $colored -replace '▓▓', "$($C.Rose)▓▓$($C.Reset)"
            $colored = $colored -replace '██', "$($C.SageGreen)██$($C.Reset)"
            $colored = $colored -replace '░░', "$($C.SageLight)░░$($C.Reset)"
            $pad = $maxLen - $line.Length
            if ($pad -gt 0) { $colored = $colored + (' ' * $pad) }
            Write-Host (Center-Text $colored $width)
        }
        Write-Host (Center-Text "$($C.Bold)$($C.Orange)W i M o$($C.Reset)  $($C.Grey)v$script:WIMO_VERSION$($C.Reset)" $width)
        Write-Host ""
        return
    }

    Write-Host ""
    $maxLen = 0
    foreach ($l in $script:MOLE_LOGO) { if ($l.Length -gt $maxLen) { $maxLen = $l.Length } }
    foreach ($line in $script:MOLE_LOGO) {
        $colored = $line
        $colored = $colored -replace '◉', "$($C.Yellow)◉$($C.Rose)"
        $colored = $colored -replace '▓▓', "$($C.Rose)▓▓$($C.Reset)"
        $colored = $colored -replace '██', "$($C.SageGreen)██$($C.Reset)"
        $colored = $colored -replace '░░', "$($C.SageLight)░░$($C.Reset)"
        $pad = $maxLen - $line.Length
        if ($pad -gt 0) { $colored = $colored + (' ' * $pad) }
        Write-Host (Center-Text $colored $width)
        Start-Sleep -Milliseconds 15
    }
    Write-Host ""
    Write-Host (Center-Text "$($C.Bold)$($C.Orange)W i M o$($C.Reset)  �  $($C.Green)v$script:WIMO_VERSION$($C.Reset)" $width)
    Write-Host (Center-Text "$($C.Grey)Windows System Optimizer$($C.Reset)" $width)
    Write-Host ""
}

function Show-Version {
    Show-Banner -Compact
}

function Show-Help {
    Show-Banner -Compact

    $help = @"
$($C.Bold)USAGE$($C.Reset)
    wimo [command] [flags]

$($C.Bold)COMMANDS$($C.Reset)
    $($C.Cyan)clean$($C.Reset)        Deep system cleanup — temp files, caches, browser leftovers
    $($C.Cyan)uninstall$($C.Reset)    Interactive app uninstaller (registry + winget integration)
    $($C.Cyan)optimize$($C.Reset)     System optimization (clear caches, refresh services)
    $($C.Cyan)analyze$($C.Reset)      Visual disk space explorer (Go TUI)
    $($C.Cyan)status$($C.Reset)       Live system health dashboard (CPU, RAM, Disk, Network)
    $($C.Cyan)purge$($C.Reset)        Clean project build artifacts (all major stacks)
    $($C.Cyan)update$($C.Reset)       Self-update from GitHub releases
    $($C.Cyan)remove$($C.Reset)       Uninstall WiMo from the system

$($C.Bold)FLAGS$($C.Reset)
    $($C.Grey)--help$($C.Reset)       Show this help message
    $($C.Grey)--version$($C.Reset)    Show version
    $($C.Grey)--debug$($C.Reset)      Enable verbose debug logging

$($C.Bold)CLEAN FLAGS$($C.Reset)
    $($C.Grey)--dry-run$($C.Reset)    Preview cleanup plan without deleting
    $($C.Grey)--whitelist$($C.Reset)  Manage protected paths

$($C.Bold)PURGE FLAGS$($C.Reset)
    $($C.Grey)--paths$($C.Reset)      Specify custom scan directories
    $($C.Grey)--depth N$($C.Reset)    Limit recursion depth (default: 8)

$($C.Bold)EXAMPLES$($C.Reset)
    wimo                    Interactive main menu
    wimo clean --dry-run    Preview what would be cleaned
    wimo purge --paths "C:\Projects"
    wimo status             Live system dashboard
"@
    Write-Host $help
    Write-Host ""
}

function Test-IsAdmin {
    ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
        [Security.Principal.WindowsBuiltInRole]::Administrator
    )
}

function Get-WimoConfig {
    if (-not (Test-Path $script:WIMO_CONFIG_FILE)) {
        $defaultConfig = @{
            version        = $script:WIMO_VERSION
            whitelist      = @()
            scan_paths     = @("$env:USERPROFILE\Projects", "$env:USERPROFILE\Documents\dev")
            purge_depth    = 8
            theme          = "default"
            check_updates  = $true
            winget_enabled = $true
        }
        New-Item -ItemType Directory -Path $script:WIMO_CONFIG_DIR -Force | Out-Null
        $defaultConfig | ConvertTo-Json -Depth 4 | Set-Content $script:WIMO_CONFIG_FILE -Encoding UTF8
        return $defaultConfig
    }
    try {
        return (Get-Content $script:WIMO_CONFIG_FILE -Raw | ConvertFrom-Json)
    } catch {
        Write-WimoLog "Failed to load config: $_" -Level Error
        return @{ whitelist = @(); scan_paths = @("$env:USERPROFILE\Projects"); purge_depth = 8 }
    }
}

function Save-WimoConfig {
    <#
    .SYNOPSIS
        Saves the WiMo configuration to disk.
    #>
    param($Config)

    try {
        New-Item -ItemType Directory -Path $script:WIMO_CONFIG_DIR -Force | Out-Null
        $Config | ConvertTo-Json -Depth 4 | Set-Content $script:WIMO_CONFIG_FILE -Encoding UTF8
        Write-WimoLog "Config saved" -Level Debug
    } catch {
        Write-WimoLog "Failed to save config: $_" -Level Error
    }
}

function Save-WimoConfig {
    param($Config)
    New-Item -ItemType Directory -Path $script:WIMO_CONFIG_DIR -Force | Out-Null
    $Config | ConvertTo-Json -Depth 4 | Set-Content $script:WIMO_CONFIG_FILE -Encoding UTF8
}

function Format-FileSize {
    param([long]$Bytes)
    if ($Bytes -lt 1KB) { return "{0} B" -f $Bytes }
    if ($Bytes -lt 1MB) { return "{0:N1} KB" -f ($Bytes / 1KB) }
    if ($Bytes -lt 1GB) { return "{0:N1} MB" -f ($Bytes / 1MB) }
    return "{0:N1} GB" -f ($Bytes / 1GB)
}

function Get-FolderSize {
    param([string]$Path)
    if (-not (Test-Path $Path)) { return 0 }
    try {
        $size = (Get-ChildItem -Path $Path -Recurse -Force -ErrorAction SilentlyContinue |
                 Measure-Object -Property Length -Sum -ErrorAction SilentlyContinue).Sum
        if ($null -eq $size) { return 0 }
        return [long]$size
    } catch {
        return 0
    }
}

function Confirm-Action {
    param([string]$Message)
    Write-Host ""
    Write-Host "  $($C.Orange)⚠  $Message$($C.Reset)"
    Write-Host "  $($C.Grey)Press$($C.Reset) $($C.Bold)Y$($C.Reset) $($C.Grey)to confirm,$($C.Reset) $($C.Bold)N$($C.Reset) $($C.Grey)to cancel:$($C.Reset) " -NoNewline
    $key = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
    Write-Host $key.Character
    return ($key.Character -eq 'Y' -or $key.Character -eq 'y')
}
