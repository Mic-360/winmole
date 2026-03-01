# WiMo — optimize.ps1
# System optimization (clear caches, refresh services)

# Ensure core libs are loaded
if (-not $script:WIMO_VERSION) {
    . "$PSScriptRoot\..\lib\core\common.ps1"
}

$isAdmin = Test-IsAdmin

$tasks = @(
    @{ Label = "Clear DNS cache";                Action = { Clear-DnsClientCache }; AdminOnly = $false }
    @{ Label = "Flush thumbnail cache";          Action = { Remove-Item "$env:LOCALAPPDATA\Microsoft\Windows\Explorer\thumbcache_*.db" -Force -ErrorAction SilentlyContinue }; AdminOnly = $false }
    @{ Label = "Flush icon cache";               Action = { ie4uinit.exe -show }; AdminOnly = $false }
    @{ Label = "Empty Recycle Bin";              Action = { Clear-RecycleBin -Force -ErrorAction SilentlyContinue }; AdminOnly = $false }
    @{ Label = "Clear clipboard";                Action = { Set-Clipboard -Value $null }; AdminOnly = $false }
    @{ Label = "Clear Windows Store cache";      Action = { Start-Process wsreset.exe -WindowStyle Hidden -Wait }; AdminOnly = $false }
    @{ Label = "Trim SSD (if applicable)";       Action = {
        Get-Volume | Where-Object { $_.DriveType -eq 'Fixed' -and $_.DriveLetter } | ForEach-Object {
            Optimize-Volume -DriveLetter $_.DriveLetter -ReTrim -ErrorAction SilentlyContinue
        }
    }; AdminOnly = $false }
    @{ Label = "Refresh Windows Search index";   Action = { Restart-Service WSearch -ErrorAction SilentlyContinue }; AdminOnly = $false }
    @{ Label = "Reset Winsock";                  Action = { netsh winsock reset 2>&1 | Out-Null }; AdminOnly = $true }
    @{ Label = "Flush Windows Event logs";       Action = { wevtutil cl Application 2>&1 | Out-Null; wevtutil cl System 2>&1 | Out-Null }; AdminOnly = $true }
    @{ Label = "Clear font cache";               Action = { Stop-Service FontCache -Force -ErrorAction SilentlyContinue; Remove-Item "$env:windir\ServiceProfiles\LocalService\AppData\Local\FontCache\*" -Force -Recurse -ErrorAction SilentlyContinue; Start-Service FontCache -ErrorAction SilentlyContinue }; AdminOnly = $true }
)

# Main execution
Show-Banner -Compact

Write-ColorLine "  $($C.Bold)⚡ Optimizing Windows...$($C.Reset)" -Color $C.White
Write-Host ""

$completed = 0
$skipped = 0
$total = $tasks.Count

foreach ($task in $tasks) {
    if ($task.AdminOnly -and -not $isAdmin) {
        Show-TaskResult -Label $task.Label -Success $false -Note "Admin required — run as Administrator"
        $skipped++
        continue
    }

    $sw = [System.Diagnostics.Stopwatch]::StartNew()

    try {
        & $task.Action
        $sw.Stop()
        $elapsed = "{0}ms" -f $sw.ElapsedMilliseconds
        Show-TaskResult -Label $task.Label -Success $true -Time $elapsed
        $completed++
    } catch {
        $sw.Stop()
        $elapsed = "{0}ms" -f $sw.ElapsedMilliseconds
        Show-TaskResult -Label $task.Label -Success $false -Time $elapsed -Note $_.Exception.Message
        $skipped++
    }
}

# Separator
Write-Host "  $($C.Grey)$('─' * 55)$($C.Reset)"

# Summary
Write-Host ""
Show-Summary -MainText "System optimization completed  ·  $completed/$total tasks"

if ($skipped -gt 0 -and -not $isAdmin) {
    Write-ColorLine "  $($C.Orange)⚠$($C.Reset)  Run as Administrator to complete all optimization tasks" -Color $C.White
    Write-Host ""
}
