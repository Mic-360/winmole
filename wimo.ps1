param(
    [string]$BinaryPath = (Join-Path $PSScriptRoot 'winmole.exe')
)

if (Test-Path $BinaryPath) {
    & $BinaryPath @args
    exit $LASTEXITCODE
}

if (Get-Command go -ErrorAction SilentlyContinue) {
    Push-Location $PSScriptRoot
    try {
        go run . @args
        exit $LASTEXITCODE
    } finally {
        Pop-Location
    }
}

Write-Error 'winmole.exe was not found. Build the Go application with: go build -o bin\winmole.exe .'
exit 1
