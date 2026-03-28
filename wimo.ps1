param(
    [string]$BinaryPath = (Join-Path $PSScriptRoot 'winmole.exe')
)

if (Test-Path $BinaryPath) {
    & $BinaryPath @args
    exit $LASTEXITCODE
}

if ((Get-Command go -ErrorAction SilentlyContinue) -and (Test-Path (Join-Path $PSScriptRoot 'go.mod'))) {
    Push-Location $PSScriptRoot
    try {
        go run . @args
        exit $LASTEXITCODE
    } finally {
        Pop-Location
    }
}

Write-Error "$BinaryPath was not found. Build the Go application with: go build -o $BinaryPath ."
exit 1
