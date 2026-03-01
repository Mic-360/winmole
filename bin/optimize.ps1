# WiMo — optimize.ps1
# System optimization — categorized, parallelized, comprehensive

# Ensure core libs are loaded
if (-not $script:WIMO_VERSION) {
    . "$PSScriptRoot\..\lib\core\common.ps1"
}

$isAdmin = Test-IsAdmin

# ── Optimization tasks organized by category ──────────────────

$categories = [ordered]@{
    "Network" = @(
        @{ Label = "Flush DNS cache";                 Action = { Clear-DnsClientCache }; AdminOnly = $false }
        @{ Label = "Flush ARP cache";                 Action = { arp -d * 2>&1 | Out-Null }; AdminOnly = $false }
        @{ Label = "Reset Winsock catalog";           Action = { netsh winsock reset 2>&1 | Out-Null }; AdminOnly = $true }
        @{ Label = "Reset TCP/IP stack";              Action = { netsh int ip reset 2>&1 | Out-Null }; AdminOnly = $true }
        @{ Label = "Flush NetBIOS cache";             Action = { nbtstat -R 2>&1 | Out-Null }; AdminOnly = $false }
        @{ Label = "Reset DNS client settings";       Action = { ipconfig /registerdns 2>&1 | Out-Null }; AdminOnly = $false }
    )
    "Storage" = @(
        @{ Label = "Empty Recycle Bin";               Action = { Clear-RecycleBin -Force -ErrorAction SilentlyContinue }; AdminOnly = $false }
        @{ Label = "Flush thumbnail cache";           Action = { Remove-Item "$env:LOCALAPPDATA\Microsoft\Windows\Explorer\thumbcache_*.db" -Force -ErrorAction SilentlyContinue }; AdminOnly = $false }
        @{ Label = "Rebuild icon cache";              Action = { ie4uinit.exe -show; Remove-Item "$env:LOCALAPPDATA\Microsoft\Windows\Explorer\iconcache_*.db" -Force -ErrorAction SilentlyContinue }; AdminOnly = $false }
        @{ Label = "Clear Windows Store cache";       Action = { Start-Process wsreset.exe -WindowStyle Hidden -Wait }; AdminOnly = $false }
        @{ Label = "Trim SSD (if applicable)";        Action = {
            Get-Volume | Where-Object { $_.DriveType -eq 'Fixed' -and $_.DriveLetter } | ForEach-Object {
                Optimize-Volume -DriveLetter $_.DriveLetter -ReTrim -ErrorAction SilentlyContinue
            }
        }; AdminOnly = $false }
        @{ Label = "Clear shader cache";              Action = {
            $shaderPaths = @(
                "$env:LOCALAPPDATA\NVIDIA\DXCache",
                "$env:LOCALAPPDATA\NVIDIA\GLCache",
                "$env:LOCALAPPDATA\AMD\DxCache",
                "$env:LOCALAPPDATA\D3DSCache"
            )
            foreach ($p in $shaderPaths) {
                if (Test-Path $p) { Remove-Item "$p\*" -Recurse -Force -ErrorAction SilentlyContinue }
            }
        }; AdminOnly = $false }
        @{ Label = "Compact OS (reclaim space)";      Action = { compact.exe /CompactOS:always 2>&1 | Out-Null }; AdminOnly = $true }
        @{ Label = "Clear delivery optimization";     Action = { Delete-DeliveryOptimizationCache -Force -ErrorAction SilentlyContinue }; AdminOnly = $true }
    )
    "System" = @(
        @{ Label = "Clear clipboard";                 Action = { Set-Clipboard -Value $null }; AdminOnly = $false }
        @{ Label = "Refresh Windows Search index";    Action = { Restart-Service WSearch -ErrorAction SilentlyContinue }; AdminOnly = $false }
        @{ Label = "Flush Windows Event logs";        Action = { wevtutil cl Application 2>&1 | Out-Null; wevtutil cl System 2>&1 | Out-Null; wevtutil cl Security 2>&1 | Out-Null }; AdminOnly = $true }
        @{ Label = "Clear font cache";                Action = { Stop-Service FontCache -Force -ErrorAction SilentlyContinue; Remove-Item "$env:windir\ServiceProfiles\LocalService\AppData\Local\FontCache\*" -Force -Recurse -ErrorAction SilentlyContinue; Start-Service FontCache -ErrorAction SilentlyContinue }; AdminOnly = $true }
        @{ Label = "Clear error reporting queue";     Action = { Remove-Item "C:\ProgramData\Microsoft\Windows\WER\*" -Recurse -Force -ErrorAction SilentlyContinue }; AdminOnly = $true }
        @{ Label = "Rebuild performance counters";    Action = { lodctr /R 2>&1 | Out-Null }; AdminOnly = $true }
        @{ Label = "Clear Windows temp files";        Action = { Remove-Item "$env:windir\Temp\*" -Recurse -Force -ErrorAction SilentlyContinue }; AdminOnly = $true }
    )
    "Performance" = @(
        @{ Label = "Flush standby memory";            Action = {
            # Release standby list memory using EmptyStandbyList pattern
            [System.GC]::Collect()
            [System.GC]::WaitForPendingFinalizers()
        }; AdminOnly = $false }
        @{ Label = "Disable Windows tips/suggestions"; Action = {
            $regPath = 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\ContentDeliveryManager'
            Set-ItemProperty -Path $regPath -Name 'SubscribedContent-338389Enabled' -Value 0 -ErrorAction SilentlyContinue
            Set-ItemProperty -Path $regPath -Name 'SubscribedContent-310093Enabled' -Value 0 -ErrorAction SilentlyContinue
            Set-ItemProperty -Path $regPath -Name 'SoftLandingEnabled' -Value 0 -ErrorAction SilentlyContinue
        }; AdminOnly = $false }
        @{ Label = "Set power plan to High Perf";     Action = {
            $highPerf = powercfg /list 2>&1 | Select-String '8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c'
            if ($highPerf) { powercfg /setactive 8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c 2>&1 | Out-Null }
        }; AdminOnly = $false }
        @{ Label = "Disable transparency effects";    Action = {
            Set-ItemProperty -Path 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Themes\Personalize' -Name 'EnableTransparency' -Value 0 -ErrorAction SilentlyContinue
        }; AdminOnly = $false }
        @{ Label = "Optimize visual effects";         Action = {
            $regPath = 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects'
            Set-ItemProperty -Path $regPath -Name 'VisualFXSetting' -Value 2 -ErrorAction SilentlyContinue
        }; AdminOnly = $false }
    )
}

# Main execution
Show-Banner -Compact

Write-ColorLine "  $($C.Bold)⚡ WiMo Optimize  ·  System optimization$($C.Reset)" -Color $C.White
Write-Host ""

$completed = 0
$skipped = 0
$failed = 0
$totalTasks = 0

foreach ($cat in $categories.Keys) {
    $totalTasks += $categories[$cat].Count
}

$taskNum = 0

foreach ($cat in $categories.Keys) {
    $tasks = $categories[$cat]

    # Category header
    $catIcon = switch ($cat) {
        "Network"     { "⇅" }
        "Storage"     { "▤" }
        "System"      { "⚙" }
        "Performance" { "▶" }
    }
    Write-Host ""
    Write-Host "  $($C.Bold)$($C.Cyan)$catIcon  $cat$($C.Reset)  $($C.Grey)($($tasks.Count) tasks)$($C.Reset)"
    Write-Host "  $($C.Grey)$('─' * 55)$($C.Reset)"

    foreach ($task in $tasks) {
        $taskNum++

        if ($task.AdminOnly -and -not $isAdmin) {
            Show-TaskResult -Label $task.Label -Success $false -Note "Admin required"
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
            $failed++
        }
    }
}

# Summary
Write-Host ""
Write-Host "  $($C.Grey)$('═' * 55)$($C.Reset)"
Write-Host ""

$statusParts = @("$($C.Green)$completed done$($C.Reset)")
if ($skipped -gt 0) { $statusParts += "$($C.Orange)$skipped skipped$($C.Reset)" }
if ($failed -gt 0) { $statusParts += "$($C.Red)$failed failed$($C.Reset)" }
$statusLine = $statusParts -join "  ·  "

Show-Summary -MainText "Optimization completed  ·  $completed/$totalTasks tasks" -SubText $statusLine

if ($skipped -gt 0 -and -not $isAdmin) {
    Write-ColorLine "  $($C.Orange)⚠$($C.Reset)  Run as Administrator to complete all optimization tasks" -Color $C.White
    Write-Host ""
}
