# WiMo — analyze.ps1
# Wrapper that calls analyze-go.exe (Go TUI disk space analyzer)

# Ensure core libs are loaded
if (-not $script:WIMO_VERSION) {
    . "$PSScriptRoot\..\lib\core\common.ps1"
}

$goBinary = "$PSScriptRoot\analyze-go.exe"

if (-not (Test-Path $goBinary)) {
    # Try alternate locations
    $altPaths = @(
        "$PSScriptRoot\..\bin\analyze-go.exe",
        "$env:LOCALAPPDATA\WiMo\bin\analyze-go.exe"
    )

    $found = $false
    foreach ($alt in $altPaths) {
        if (Test-Path $alt) {
            $goBinary = $alt
            $found = $true
            break
        }
    }

    if (-not $found) {
        Show-Banner -Compact
        Write-ColorLine "  $($C.Red)✗$($C.Reset)  analyze-go.exe not found." -Color $C.White
        Write-Host ""
        Write-ColorLine "  Build it with: go build -ldflags `"-s -w`" -o bin\analyze-go.exe .\cmd\analyze" -Color $C.Grey
        Write-ColorLine "  Or run: make build" -Color $C.Grey
        Write-Host ""
        return
    }
}

# Forward all args to the Go binary
& $goBinary @args
