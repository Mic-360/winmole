# WiMo — clean.ps1
# Deep system cleanup — temp files, caches, browser leftovers, windows.old

# Parse args (supports both --dry-run and -DryRun styles)
$DryRun = $false
$Whitelist = $false
foreach ($a in $args) {
    if ($a -eq '--dry-run' -or $a -eq '-DryRun') { $DryRun = $true }
    if ($a -eq '--whitelist' -or $a -eq '-Whitelist') { $Whitelist = $true }
}

# Ensure core libs are loaded
if (-not $script:WIMO_VERSION) {
    . "$PSScriptRoot\..\lib\core\common.ps1"
}

# Whitelist management mode
if ($Whitelist) {
    Show-Banner -Compact
    Write-ColorLine "  $($C.Bold)Whitelist Management$($C.Reset)" -Color $C.White
    Write-Host ""

    $config = Get-WimoConfig
    if ($config.whitelist.Count -eq 0) {
        Write-ColorLine "  No whitelisted paths." -Color $C.Grey
    } else {
        Write-ColorLine "  Current whitelisted paths:" -Color $C.Cyan
        foreach ($p in $config.whitelist) {
            Write-ColorLine "    ○  $p" -Color $C.Grey
        }
    }

    Write-Host ""
    Write-Host "  $($C.Bold)Add a path to whitelist (or press Enter to cancel):$($C.Reset) " -NoNewline
    $newPath = Read-Host
    if ($newPath -and (Test-Path $newPath)) {
        $config.whitelist += $newPath
        Save-WimoConfig $config
        Write-ColorLine "  $($C.Green)✓$($C.Reset)  Added to whitelist: $newPath" -Color $C.White
    } elseif ($newPath) {
        Write-ColorLine "  $($C.Red)✗$($C.Reset)  Path does not exist: $newPath" -Color $C.White
    }
    return
}

# Clean targets — User-level (no admin)
$CleanTargets = @(
    @{ Label = "Temp files";                     Path = $env:TEMP }
    @{ Label = "Local app temp";                 Path = "$env:LOCALAPPDATA\Temp" }
    @{ Label = "Chrome cache";                   Path = "$env:LOCALAPPDATA\Google\Chrome\User Data\Default\Cache" }
    @{ Label = "Chrome GPU cache";               Path = "$env:LOCALAPPDATA\Google\Chrome\User Data\Default\GPUCache" }
    @{ Label = "Edge cache";                     Path = "$env:LOCALAPPDATA\Microsoft\Edge\User Data\Default\Cache" }
    @{ Label = "Firefox cache";                  Path = "$env:LOCALAPPDATA\Mozilla\Firefox\Profiles\*\cache2" }
    @{ Label = "IE / legacy cache";              Path = "$env:LOCALAPPDATA\Microsoft\Windows\INetCache" }
    @{ Label = "Thumbnail cache";                Path = "$env:LOCALAPPDATA\Microsoft\Windows\Explorer\thumbcache_*.db" }
    @{ Label = "Explorer icon cache";            Path = "$env:LOCALAPPDATA\Microsoft\Windows\Explorer\iconcache_*.db" }
    @{ Label = "Discord cache";                  Path = "$env:APPDATA\discord\Cache" }
    @{ Label = "Discord code cache";             Path = "$env:APPDATA\discord\Code Cache" }
    @{ Label = "Slack cache";                    Path = "$env:APPDATA\Slack\Cache" }
    @{ Label = "Slack code cache";               Path = "$env:APPDATA\Slack\Code Cache" }
    @{ Label = "Spotify cache";                  Path = "$env:LOCALAPPDATA\Spotify\Data" }
    @{ Label = "Teams cache";                    Path = "$env:APPDATA\Microsoft\Teams\Cache" }
    @{ Label = "Teams media cache";              Path = "$env:APPDATA\Microsoft\Teams\blob_storage" }
    @{ Label = "VS Code logs";                   Path = "$env:APPDATA\Code\logs" }
    @{ Label = "VS Code crash reports";          Path = "$env:APPDATA\Code\CrashpadMetrics-active.pma" }
    @{ Label = "JetBrains caches";               Path = "$env:LOCALAPPDATA\JetBrains\*\caches" }
    @{ Label = "npm cache";                      Path = "$env:APPDATA\npm-cache" }
    @{ Label = "pip cache";                      Path = "$env:LOCALAPPDATA\pip\Cache" }
    @{ Label = "Yarn cache";                     Path = "$env:LOCALAPPDATA\Yarn\Cache" }
    @{ Label = "pnpm store cache";               Path = "$env:LOCALAPPDATA\pnpm\store" }
    @{ Label = "Cargo registry cache";           Path = "$env:USERPROFILE\.cargo\registry\cache" }
    @{ Label = "Crash dumps";                    Path = "$env:LOCALAPPDATA\CrashDumps" }
    @{ Label = "UWP app temp state";             Path = "$env:LOCALAPPDATA\Packages\*\TempState" }
    @{ Label = "Recent docs (jump list)";        Path = "$env:APPDATA\Microsoft\Windows\Recent\*" }
    @{ Label = "Startup event logs (user)";      Path = "$env:LOCALAPPDATA\Diagnostics" }
)

# Admin-level targets
$CleanTargetsAdmin = @(
    @{ Label = "System temp files";              Path = "C:\Windows\Temp" }
    @{ Label = "Windows Update download cache";  Path = "C:\Windows\SoftwareDistribution\Download" }
    @{ Label = "Prefetch files";                 Path = "C:\Windows\Prefetch" }
    @{ Label = "CBS logs";                       Path = "C:\Windows\Logs\CBS" }
    @{ Label = "DISM logs";                      Path = "C:\Windows\Logs\DISM" }
    @{ Label = "Memory dump files";              Path = "C:\Windows\Minidump" }
    @{ Label = "Old memory dump";                Path = "C:\Windows\MEMORY.DMP" }
)

$WindowsOldPath = "C:\Windows.old"

function Remove-WindowsOld {
    if (-not (Test-Path $WindowsOldPath)) {
        Show-ScanItem -Status Skip -Label "windows.old not found — already clean" -Size ""
        return 0
    }

    $size = Get-FolderSize $WindowsOldPath
    $sizeText = Format-FileSize $size

    Write-Host ""
    Write-Host "  $($C.Grey)┌─ Previous Windows Installation ─────────────────────────────┐$($C.Reset)"
    Write-Host "  $($C.Grey)│$($C.Reset)  $($C.Orange)⚠$($C.Reset)  C:\Windows.old  ·  $($C.Bold)$sizeText$($C.Reset)  ·  Safe to remove          $($C.Grey)│$($C.Reset)"
    Write-Host "  $($C.Grey)│$($C.Reset)     Removes the ability to roll back to previous Windows     $($C.Grey)│$($C.Reset)"
    Write-Host "  $($C.Grey)└──────────────────────────────────────────────────────────────┘$($C.Reset)"

    if ($DryRun) {
        Show-ScanItem -Status Warn -Label "windows.old" -Size $sizeText -Badge "dry-run: would remove via DISM"
        return $size
    }

    if (-not (Confirm-Action "Remove windows.old? This cannot be undone and removes the rollback option.")) {
        Show-ScanItem -Status Skip -Label "windows.old" -Size $sizeText -Badge "Skipped by user"
        return 0
    }

    Write-ColorLine "  ⟳  Running DISM cleanup (this may take a few minutes)..." -Color $C.Cyan
    $result = & dism.exe /Online /Cleanup-Image /SPSuperseded 2>&1

    if ($LASTEXITCODE -eq 0) {
        Show-ScanItem -Status Success -Label "windows.old removed via DISM" -Size $sizeText
        return $size
    } else {
        & cleanmgr.exe /sagerun:1 2>&1 | Out-Null
        Show-ScanItem -Status Success -Label "windows.old cleanup initiated via Disk Cleanup" -Size $sizeText
        return $size
    }
}

# Main execution
Show-Banner -Compact

if ($DryRun) {
    Write-ColorLine "  $($C.Orange)DRY RUN$($C.Reset) — Previewing cleanup plan (nothing will be deleted)" -Color $C.Orange
    Write-Host ""
}

Write-ColorLine "  $($C.Bold)🧹 WiMo Clean  ·  Scanning system...$($C.Reset)" -Color $C.White
Write-Host ""

$totalFreed = [long]0
$totalItems = 0
$isAdmin = Test-IsAdmin

# ── Parallel scan of user-level targets ──────────────────────
Write-ColorLine "  $($C.Cyan)User-level caches:$($C.Reset)" -Color $C.Cyan
Write-Host ""

$config = Get-WimoConfig

# Pre-scan sizes in parallel using runspaces for speed
$pool = [runspacefactory]::CreateRunspacePool(1, [Math]::Min(16, $CleanTargets.Count))
$pool.Open()

$jobs = @()
foreach ($target in $CleanTargets) {
    $ps = [powershell]::Create().AddScript({
        param($p)
        [long]$total = 0
        $items = Get-Item -Path $p -Force -ErrorAction SilentlyContinue
        foreach ($item in $items) {
            if ($item.PSIsContainer) {
                try {
                    foreach ($f in [System.IO.Directory]::EnumerateFiles($item.FullName, '*', [System.IO.SearchOption]::AllDirectories)) {
                        try { $total += ([System.IO.FileInfo]::new($f)).Length } catch {}
                    }
                } catch {}
            } else {
                $total += $item.Length
            }
        }
        return $total
    }).AddArgument($target.Path)
    $ps.RunspacePool = $pool
    $jobs += @{ Pipe = $ps; Handle = $ps.BeginInvoke(); Target = $target }
}

# Collect results and process
foreach ($job in $jobs) {
    $size = $job.Pipe.EndInvoke($job.Handle)
    if ($null -eq $size) { $size = 0 } else { $size = [long]$size }
    $job.Pipe.Dispose()
    $target = $job.Target

    if ($size -eq 0) { continue }

    $sizeText = Format-FileSize $size
    $totalItems++

    # Check whitelist
    $whitelisted = $false
    if ($config.whitelist) {
        foreach ($wp in $config.whitelist) {
            if ($target.Path -like "$wp*") { $whitelisted = $true; break }
        }
    }

    if ($whitelisted) {
        Show-ScanItem -Status Skip -Label $target.Label -Size $sizeText -Badge "Whitelist"
        continue
    }

    if ($DryRun) {
        Show-ScanItem -Status Success -Label $target.Label -Size $sizeText
        $totalFreed += $size
    } else {
        $freed = Remove-SafeGlob -Pattern $target.Path
        if ($freed -gt 0) {
            Show-ScanItem -Status Success -Label $target.Label -Size (Format-FileSize $freed)
            $totalFreed += $freed
        } else {
            $result = Remove-SafePath -Path $target.Path
            if ($result.Success) {
                Show-ScanItem -Status Success -Label $target.Label -Size (Format-FileSize $result.BytesFreed)
                $totalFreed += $result.BytesFreed
            } else {
                Show-ScanItem -Status Error -Label $target.Label -Size $sizeText -Badge $result.Reason
            }
        }
    }
}

$pool.Close()
$pool.Dispose()

# Scan admin-level targets
Write-Host ""
Write-ColorLine "  $($C.Cyan)System-level caches:$($C.Reset)" -Color $C.Cyan
Write-Host ""

foreach ($target in $CleanTargetsAdmin) {
    $size = Get-PathSize $target.Path

    if ($size -eq 0) {
        continue
    }

    $sizeText = Format-FileSize $size
    $totalItems++

    if (-not $isAdmin) {
        Show-ScanItem -Status Warn -Label $target.Label -Size $sizeText -Badge "Admin required"
        continue
    }

    if ($DryRun) {
        Show-ScanItem -Status Success -Label $target.Label -Size $sizeText
        $totalFreed += $size
    } else {
        $result = Remove-SafePath -Path $target.Path
        if ($result.Success) {
            Show-ScanItem -Status Success -Label $target.Label -Size (Format-FileSize $result.BytesFreed)
            $totalFreed += $result.BytesFreed
        } else {
            Show-ScanItem -Status Error -Label $target.Label -Size $sizeText -Badge $result.Reason
        }
    }
}

# windows.old — special handling
Write-Host ""
if ($isAdmin -or $DryRun) {
    $windowsOldFreed = Remove-WindowsOld
    $totalFreed += $windowsOldFreed
} else {
    if (Test-Path $WindowsOldPath) {
        $woSize = Get-FolderSize $WindowsOldPath
        Write-Host "  $($C.Grey)┌─ Previous Windows Installation ─────────────────────────────┐$($C.Reset)"
        Write-Host "  $($C.Grey)│$($C.Reset)  $($C.Orange)⚠$($C.Reset)  C:\Windows.old  ·  $($C.Bold)$(Format-FileSize $woSize)$($C.Reset)                         $($C.Grey)│$($C.Reset)"
        Write-Host "  $($C.Grey)│$($C.Reset)     Run as Administrator to remove                          $($C.Grey)│$($C.Reset)"
        Write-Host "  $($C.Grey)└──────────────────────────────────────────────────────────────┘$($C.Reset)"
    }
}

# Summary
Write-Host ""
$freeSpace = try { (Get-Volume -DriveLetter C -ErrorAction SilentlyContinue).SizeRemaining } catch { 0 }
$freeText = if ($freeSpace -gt 0) { "Free now: $(Format-FileSize $freeSpace)" } else { "" }

if ($DryRun) {
    Show-Summary -MainText "Would free: $(Format-FileSize $totalFreed)" -SubText $freeText
} else {
    Show-Summary -MainText "Space freed: $(Format-FileSize $totalFreed)" -SubText $freeText
}

if (-not $isAdmin) {
    Write-ColorLine "  $($C.Orange)⚠$($C.Reset)  Run as Administrator to clean system-level caches" -Color $C.White
    Write-Host ""
}
