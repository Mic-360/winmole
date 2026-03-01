# WiMo — log.ps1
# Logging with --debug support

$script:WIMO_DEBUG = $false
$script:WIMO_LOG_FILE = $null

function Initialize-WimoLog {
    param([switch]$Debug)
    $script:WIMO_DEBUG = $Debug.IsPresent

    if ($script:WIMO_DEBUG) {
        $logDir = "$env:APPDATA\WiMo\logs"
        New-Item -ItemType Directory -Path $logDir -Force | Out-Null
        $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
        $script:WIMO_LOG_FILE = "$logDir\wimo_$timestamp.log"
        Write-WimoLog "Debug logging enabled" -Level Debug
        Write-WimoLog "WiMo version: $script:WIMO_VERSION" -Level Debug
        Write-WimoLog "PowerShell: $($PSVersionTable.PSVersion)" -Level Debug
        Write-WimoLog "OS: $([System.Environment]::OSVersion.VersionString)" -Level Debug
    }
}

function Write-WimoLog {
    param(
        [string]$Message,
        [ValidateSet('Debug','Info','Warn','Error')]
        [string]$Level = 'Info'
    )

    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss.fff"
    $logEntry = "[$timestamp] [$Level] $Message"

    if ($script:WIMO_LOG_FILE) {
        Add-Content -Path $script:WIMO_LOG_FILE -Value $logEntry -ErrorAction SilentlyContinue
    }

    if ($script:WIMO_DEBUG) {
        $color = switch ($Level) {
            'Debug' { $C.Grey }
            'Info'  { $C.Cyan }
            'Warn'  { $C.Orange }
            'Error' { $C.Red }
        }
        Write-Host "  $color[$Level]$($C.Reset) $($C.Grey)$Message$($C.Reset)"
    }
}

function Write-ColorLine {
    param(
        [string]$Text,
        [string]$Color = $C.White
    )
    Write-Host "$Color$Text$($C.Reset)"
}
