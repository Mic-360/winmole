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
    @{ Label = "Brave cache";                    Path = "$env:LOCALAPPDATA\BraveSoftware\Brave-Browser\User Data\Default\Cache" }
    @{ Label = "Brave GPU cache";                Path = "$env:LOCALAPPDATA\BraveSoftware\Brave-Browser\User Data\Default\GPUCache" }
    @{ Label = "Opera cache";                    Path = "$env:APPDATA\Opera Software\Opera Stable\Cache" }
    @{ Label = "Vivaldi cache";                  Path = "$env:LOCALAPPDATA\Vivaldi\User Data\Default\Cache" }
    @{ Label = "VS Code Insiders logs";          Path = "$env:APPDATA\Code - Insiders\logs" }
    @{ Label = "VS Code workspace storage";      Path = "$env:APPDATA\Code\User\workspaceStorage" }
    @{ Label = "Zoom cache";                     Path = "$env:APPDATA\Zoom\data" }
    @{ Label = "Docker desktop cache";           Path = "$env:LOCALAPPDATA\Docker\wsl\data\tmp" }
    @{ Label = "Gradle global cache";            Path = "$env:USERPROFILE\.gradle\caches" }
    @{ Label = "NuGet cache";                    Path = "$env:LOCALAPPDATA\NuGet\v3-cache" }
    @{ Label = "NuGet HTTP cache";               Path = "$env:LOCALAPPDATA\NuGet\plugins-cache" }
    @{ Label = "Go module cache";                Path = "$env:LOCALAPPDATA\go-build" }
    @{ Label = "Composer cache";                 Path = "$env:LOCALAPPDATA\Composer\cache" }
    @{ Label = "PowerShell help cache";          Path = "$env:LOCALAPPDATA\Microsoft\PowerShell\Help" }
    @{ Label = "TypeScript server cache";        Path = "$env:LOCALAPPDATA\Microsoft\TypeScript" }
    @{ Label = "Windows old downloads";          Path = "$env:LOCALAPPDATA\Microsoft\Windows\Explorer\*" }
    @{ Label = "Temp Internet Files";            Path = "$env:LOCALAPPDATA\Microsoft\Windows\Temporary Internet Files" }
    @{ Label = "Steam download cache";           Path = "$env:LOCALAPPDATA\Steam\htmlcache" }
    @{ Label = "Electron apps cache";            Path = "$env:APPDATA\*\Cache\Cache_Data" }
    @{ Label = "Electron GPU cache";             Path = "$env:APPDATA\*\GPUCache" }
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
    @{ Label = "Delivery Optimization cache";    Path = "C:\Windows\SoftwareDistribution\DeliveryOptimization" }
    @{ Label = "Windows setup logs";             Path = "C:\Windows\Panther" }
    @{ Label = "Windows Error Reporting";        Path = "C:\ProgramData\Microsoft\Windows\WER" }
    @{ Label = "System event trace logs";        Path = "C:\Windows\System32\LogFiles\WMI" }
    @{ Label = "Windows Defender scan data";     Path = "C:\ProgramData\Microsoft\Windows Defender\Scans\History" }
    @{ Label = "IIS logs";                       Path = "C:\inetpub\logs\LogFiles" }
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
$config = Get-WimoConfig

# ── Phase 1: Parallel scan user-level targets ──────────────────
Write-ColorLine "  $($C.Cyan)User-level caches:$($C.Reset)" -Color $C.Cyan
Write-Host ""

$poolSize = [Math]::Min([Math]::Max(1, $CleanTargets.Count), [Environment]::ProcessorCount * 2)
$pool = [runspacefactory]::CreateRunspacePool(1, $poolSize)
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

# Collect scan results
$userCleanItems = [System.Collections.ArrayList]::new()
foreach ($job in $jobs) {
    $size = $job.Pipe.EndInvoke($job.Handle)
    if ($null -eq $size) { $size = 0 } else { $size = [long]$size }
    $job.Pipe.Dispose()
    $target = $job.Target

    if ($size -eq 0) { continue }
    $totalItems++

    # Check whitelist
    $whitelisted = $false
    if ($config.whitelist) {
        foreach ($wp in $config.whitelist) {
            if ($target.Path -like "$wp*") { $whitelisted = $true; break }
        }
    }

    if ($whitelisted) {
        Show-ScanItem -Status Skip -Label $target.Label -Size (Format-FileSize $size) -Badge "Whitelist"
        continue
    }

    [void]$userCleanItems.Add(@{ Label = $target.Label; Path = $target.Path; Size = $size })
    Show-ScanItem -Status Success -Label $target.Label -Size (Format-FileSize $size)
}

$pool.Close()
$pool.Dispose()

# Parallel deletion of user-level targets
if (-not $DryRun -and $userCleanItems.Count -gt 0) {
    $delPoolSize = [Math]::Min([Math]::Max(1, $userCleanItems.Count), [Environment]::ProcessorCount * 2)
    $delPool = [runspacefactory]::CreateRunspacePool(1, $delPoolSize)
    $delPool.Open()

    $delJobs = @()
    foreach ($item in $userCleanItems) {
        $ps = [powershell]::Create().AddScript({
            param($path)
            [long]$freed = 0
            try {
                $targets = Get-Item -Path $path -Force -ErrorAction SilentlyContinue
                foreach ($t in $targets) {
                    try {
                        if ($t.PSIsContainer) {
                            foreach ($f in [System.IO.Directory]::EnumerateFiles($t.FullName, '*', [System.IO.SearchOption]::AllDirectories)) {
                                try { $freed += ([System.IO.FileInfo]::new($f)).Length } catch {}
                            }
                            [System.IO.Directory]::Delete($t.FullName, $true)
                        } else {
                            $freed += $t.Length
                            [System.IO.File]::Delete($t.FullName)
                        }
                    } catch {
                        try { Remove-Item -Path $t.FullName -Recurse -Force -ErrorAction Stop } catch {}
                    }
                }
            } catch {}
            return $freed
        }).AddArgument($item.Path)
        $ps.RunspacePool = $delPool
        $delJobs += @{ Pipe = $ps; Handle = $ps.BeginInvoke(); Item = $item }
    }

    foreach ($dj in $delJobs) {
        $freed = $dj.Pipe.EndInvoke($dj.Handle)
        if ($null -eq $freed) { $freed = 0 } else { $freed = [long]$freed }
        $dj.Pipe.Dispose()
        $totalFreed += $freed
    }

    $delPool.Close()
    $delPool.Dispose()
} elseif ($DryRun) {
    foreach ($item in $userCleanItems) { $totalFreed += $item.Size }
}

# ── Phase 2: Parallel scan admin-level targets ──────────────────
Write-Host ""
Write-ColorLine "  $($C.Cyan)System-level caches:$($C.Reset)" -Color $C.Cyan
Write-Host ""

if ($isAdmin) {
    $adminPoolSize = [Math]::Min([Math]::Max(1, $CleanTargetsAdmin.Count), [Environment]::ProcessorCount * 2)
    $adminPool = [runspacefactory]::CreateRunspacePool(1, $adminPoolSize)
    $adminPool.Open()

    $adminJobs = @()
    foreach ($target in $CleanTargetsAdmin) {
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
        $ps.RunspacePool = $adminPool
        $adminJobs += @{ Pipe = $ps; Handle = $ps.BeginInvoke(); Target = $target }
    }

    $adminCleanItems = [System.Collections.ArrayList]::new()
    foreach ($job in $adminJobs) {
        $size = $job.Pipe.EndInvoke($job.Handle)
        if ($null -eq $size) { $size = 0 } else { $size = [long]$size }
        $job.Pipe.Dispose()
        $target = $job.Target

        if ($size -eq 0) { continue }
        $totalItems++

        [void]$adminCleanItems.Add(@{ Label = $target.Label; Path = $target.Path; Size = $size })
        Show-ScanItem -Status Success -Label $target.Label -Size (Format-FileSize $size)
    }

    $adminPool.Close()
    $adminPool.Dispose()

    # Parallel deletion of admin-level targets
    if (-not $DryRun -and $adminCleanItems.Count -gt 0) {
        $admDelPoolSize = [Math]::Min([Math]::Max(1, $adminCleanItems.Count), [Environment]::ProcessorCount * 2)
        $admDelPool = [runspacefactory]::CreateRunspacePool(1, $admDelPoolSize)
        $admDelPool.Open()

        $admDelJobs = @()
        foreach ($item in $adminCleanItems) {
            $ps = [powershell]::Create().AddScript({
                param($path)
                [long]$freed = 0
                try {
                    $targets = Get-Item -Path $path -Force -ErrorAction SilentlyContinue
                    foreach ($t in $targets) {
                        try {
                            if ($t.PSIsContainer) {
                                foreach ($f in [System.IO.Directory]::EnumerateFiles($t.FullName, '*', [System.IO.SearchOption]::AllDirectories)) {
                                    try { $freed += ([System.IO.FileInfo]::new($f)).Length } catch {}
                                }
                                [System.IO.Directory]::Delete($t.FullName, $true)
                            } else {
                                $freed += $t.Length
                                [System.IO.File]::Delete($t.FullName)
                            }
                        } catch {
                            try { Remove-Item -Path $t.FullName -Recurse -Force -ErrorAction Stop } catch {}
                        }
                    }
                } catch {}
                return $freed
            }).AddArgument($item.Path)
            $ps.RunspacePool = $admDelPool
            $admDelJobs += @{ Pipe = $ps; Handle = $ps.BeginInvoke(); Item = $item }
        }

        foreach ($dj in $admDelJobs) {
            $freed = $dj.Pipe.EndInvoke($dj.Handle)
            if ($null -eq $freed) { $freed = 0 } else { $freed = [long]$freed }
            $dj.Pipe.Dispose()
            $totalFreed += $freed
        }

        $admDelPool.Close()
        $admDelPool.Dispose()
    } elseif ($DryRun) {
        foreach ($item in $adminCleanItems) { $totalFreed += $item.Size }
    }
} else {
    foreach ($target in $CleanTargetsAdmin) {
        $size = Get-PathSize $target.Path
        if ($size -eq 0) { continue }
        $totalItems++
        Show-ScanItem -Status Warn -Label $target.Label -Size (Format-FileSize $size) -Badge "Admin required"
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
