# WiMo — uninstall.ps1
# Interactive app uninstaller (registry + winget integration)

# Ensure core libs are loaded
if (-not $script:WIMO_VERSION) {
    . "$PSScriptRoot\..\lib\core\common.ps1"
}

function Get-RegistryApps {
    $registryPaths = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
    )

    $apps = @()
    foreach ($regPath in $registryPaths) {
        $items = Get-ItemProperty $regPath -ErrorAction SilentlyContinue |
            Where-Object { $_.DisplayName -and $_.UninstallString }

        foreach ($item in $items) {
            $sizeKB = if ($item.EstimatedSize) { [long]$item.EstimatedSize * 1024 } else { 0 }
            $installDate = if ($item.InstallDate) {
                try { [datetime]::ParseExact($item.InstallDate, 'yyyyMMdd', $null).ToString('yyyy-MM-dd') } catch { "" }
            } else { "" }

            $apps += @{
                Name               = $item.DisplayName
                Version            = if ($item.DisplayVersion) { $item.DisplayVersion } else { "" }
                Publisher          = if ($item.Publisher) { $item.Publisher } else { "" }
                InstallDate        = $installDate
                Size               = $sizeKB
                SizeText           = if ($sizeKB -gt 0) { Format-FileSize $sizeKB } else { "Unknown" }
                UninstallString    = $item.UninstallString
                QuietUninstallString = if ($item.QuietUninstallString) { $item.QuietUninstallString } else { "" }
                Source             = "registry"
                WingetId           = ""
            }
        }
    }
    return $apps
}

function Get-WingetApps {
    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) { return @() }

    $apps = @()
    try {
        $raw = winget list --accept-source-agreements 2>$null
        if (-not $raw) { return @() }

        $lines = $raw -split "`n"
        $headerIdx = -1
        for ($i = 0; $i -lt $lines.Count; $i++) {
            if ($lines[$i] -match '^Name\s+') { $headerIdx = $i; break }
        }

        if ($headerIdx -lt 0) { return @() }

        # Find column positions from the separator line
        $sepLine = $lines[$headerIdx + 1]
        if (-not ($sepLine -match '^-')) { return @() }

        $dataLines = $lines[($headerIdx + 2)..($lines.Count - 1)]
        foreach ($line in $dataLines) {
            if ([string]::IsNullOrWhiteSpace($line)) { continue }
            if ($line -match '^\d+ upgrades available') { continue }

            $cols = $line -split '\s{2,}'
            if ($cols.Count -ge 2) {
                $ansi = "$([char]0x1b)\[[0-9;]*m"
                $name = ($cols[0].Trim()) -replace $ansi, ''
                $name = $name.Trim()
                if ([string]::IsNullOrWhiteSpace($name)) { continue }
                $apps += @{
                    Name    = $name
                    Id      = if ($cols.Count -ge 2) { ($cols[1].Trim()) -replace $ansi, '' } else { "" }
                    Version = if ($cols.Count -ge 3) { ($cols[2].Trim()) -replace $ansi, '' } else { "" }
                    Source  = "winget"
                }
            }
        }
    } catch {
        Write-WimoLog "Failed to get winget apps: $_" -Level Warn
    }

    return $apps
}

function Get-LocalApps {
    $apps = @()
    $localPrograms = "$env:LOCALAPPDATA\Programs"
    if (Test-Path $localPrograms) {
        Get-ChildItem $localPrograms -Directory -ErrorAction SilentlyContinue | ForEach-Object {
            $size = Get-FolderSize $_.FullName
            $apps += @{
                Name     = $_.Name
                Path     = $_.FullName
                Size     = $size
                SizeText = Format-FileSize $size
                Source   = "local"
            }
        }
    }
    return $apps
}

function Merge-AppLists {
    param($RegistryApps, $WingetApps, $LocalApps)

    # Broad pattern: strip all ANSI CSI sequences (ESC[ ... letter) plus remaining control chars
    $ansiPattern = "$([char]0x1b)\[[^a-zA-Z]*[a-zA-Z]"
    $ctrlPattern = '[\x00-\x1F\x7F-\x9F]'
    $merged = @{}

    # Registry apps as base
    foreach ($app in $RegistryApps) {
        $rawName = (([string]$app.Name) -replace $ansiPattern, '') -replace $ctrlPattern, ''
        if ([string]::IsNullOrWhiteSpace($rawName)) { continue }
        $key = $rawName.ToLower().Trim()
        $merged[$key] = $app
    }

    # Enrich with winget IDs
    foreach ($wApp in $WingetApps) {
        $rawName = (([string]$wApp.Name) -replace $ansiPattern, '') -replace $ctrlPattern, ''
        if ([string]::IsNullOrWhiteSpace($rawName)) { continue }
        $key = $rawName.ToLower().Trim()
        if ($merged.ContainsKey($key)) {
            $merged[$key].WingetId = $wApp.Id
            $merged[$key].Source = "winget"
        } else {
            $merged[$key] = @{
                Name               = $wApp.Name
                Version            = $wApp.Version
                Publisher          = ""
                InstallDate        = ""
                Size               = 0
                SizeText           = "Unknown"
                UninstallString    = ""
                QuietUninstallString = ""
                Source             = "winget"
                WingetId           = $wApp.Id
            }
        }
    }

    # Add local apps not already found
    foreach ($lApp in $LocalApps) {
        $rawName = (([string]$lApp.Name) -replace $ansiPattern, '') -replace $ctrlPattern, ''
        if ([string]::IsNullOrWhiteSpace($rawName)) { continue }
        $key = $rawName.ToLower().Trim()
        if (-not $merged.ContainsKey($key)) {
            $merged[$key] = @{
                Name               = $lApp.Name
                Version            = ""
                Publisher          = ""
                InstallDate        = ""
                Size               = $lApp.Size
                SizeText           = $lApp.SizeText
                UninstallString    = ""
                QuietUninstallString = ""
                Source             = "local"
                WingetId           = ""
                LocalPath          = $lApp.Path
            }
        }
    }

    return $merged.Values | Sort-Object { $_.Name }
}

function Get-ScalarValue {
    param(
        $Value,
        $Default = ""
    )

    if ($null -eq $Value) { return $Default }
    if ($Value -is [string]) {
        if ([string]::IsNullOrWhiteSpace($Value)) { return $Default }
        return $Value
    }
    if ($Value -is [System.Collections.IEnumerable]) {
        $arr = @($Value)
        if ($arr.Count -eq 0) { return $Default }
        foreach ($entry in $arr) {
            if ($null -eq $entry) { continue }
            if ($entry -is [string]) {
                if ([string]::IsNullOrWhiteSpace($entry)) { continue }
                return $entry
            }
            return $entry
        }
        return $Default
    }
    return $Value
}

function Invoke-Uninstall {
    param($App)

    Write-Host ""
    Write-ColorLine "  Uninstalling: $($App.Name)" -Color $C.Orange

    # Method 1: winget (cleanest)
    if ($App.Source -eq 'winget' -and $App.WingetId -and (Get-Command winget -ErrorAction SilentlyContinue)) {
        Write-ColorLine "  ⟳  Using winget uninstall..." -Color $C.Cyan
        $output = winget uninstall --id $App.WingetId --silent --accept-source-agreements 2>&1
        Write-WimoLog "winget uninstall output: $output" -Level Debug
    }
    # Method 2: Silent uninstaller string from registry
    elseif ($App.QuietUninstallString) {
        Write-ColorLine "  ⟳  Using quiet uninstall..." -Color $C.Cyan
        Start-Process -FilePath "cmd.exe" -ArgumentList "/c `"$($App.QuietUninstallString)`"" -Wait -WindowStyle Hidden
    }
    # Method 3: Standard uninstaller
    elseif ($App.UninstallString) {
        Write-ColorLine "  ⟳  Using standard uninstall..." -Color $C.Cyan
        if ($App.UninstallString -match 'msiexec') {
            $msiArgs = $App.UninstallString -replace '(?i)msiexec\.exe\s*', '' -replace '/I', '/X'
            Start-Process msiexec.exe -ArgumentList "$msiArgs /quiet /norestart" -Wait -ErrorAction SilentlyContinue
        } else {
            Start-Process -FilePath "cmd.exe" -ArgumentList "/c `"$($App.UninstallString)`"" -Wait -WindowStyle Hidden
        }
    }
    # Method 4: Local folder removal
    elseif ($App.LocalPath) {
        Write-ColorLine "  ⟳  Removing local installation..." -Color $C.Cyan
        Remove-SafePath -Path $App.LocalPath
    }
    else {
        Write-ColorLine "  $($C.Red)✗$($C.Reset)  No uninstall method available" -Color $C.White
        return
    }

    # Clean leftovers
    Invoke-CleanLeftovers -AppName $App.Name -Publisher $App.Publisher

    Write-ColorLine "  $($C.Green)✓$($C.Reset)  Removed application" -Color $C.White
}

function Invoke-CleanLeftovers {
    param(
        [string]$AppName,
        [string]$Publisher
    )

    $searchNames = @($AppName)
    if ($Publisher) { $searchNames += $Publisher }

    $leftoverTemplates = @(
        "$env:APPDATA\{name}",
        "$env:LOCALAPPDATA\{name}",
        "$env:LOCALAPPDATA\Programs\{name}",
        "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\{name}",
        "C:\ProgramData\{name}"
    )

    $found = @()
    foreach ($name in $searchNames) {
        foreach ($template in $leftoverTemplates) {
            $path = $template -replace '\{name\}', $name
            if (Test-Path $path) { $found += $path }
        }
        Remove-RegistryStartupEntry -AppName $name
    }

    if ($found.Count -gt 0) {
        Write-ColorLine "  $($C.Green)✓$($C.Reset)  Cleaned $($found.Count) leftover locations" -Color $C.White
        foreach ($p in $found) {
            Write-ColorLine "     - $p" -Color $C.Grey
            Remove-SafePath $p | Out-Null
        }
    }
}

function Remove-RegistryStartupEntry {
    param([string]$AppName)

    $startupPaths = @(
        'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run',
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run'
    )

    foreach ($regPath in $startupPaths) {
        try {
            $props = Get-ItemProperty $regPath -ErrorAction SilentlyContinue
            if (-not $props) { continue }

            $props.PSObject.Properties | Where-Object {
                $_.Name -like "*$AppName*" -or $_.Value -like "*$AppName*"
            } | ForEach-Object {
                Remove-ItemProperty -Path $regPath -Name $_.Name -ErrorAction SilentlyContinue
                Write-WimoLog "Removed startup entry: $($_.Name)" -Level Info
            }
        } catch {
            Write-WimoLog "Could not clean startup entries for $AppName in $regPath" -Level Debug
        }
    }
}

# Main execution
Show-Banner -Compact

Write-ColorLine "  $($C.Bold)🗑️  WiMo Uninstall  ·  Discovering installed apps...$($C.Reset)" -Color $C.White
Write-Host ""

# ── Parallel scanning of all sources ─────────────────────
$scanPool = [runspacefactory]::CreateRunspacePool(1, 3)
$scanPool.Open()

# Registry scan (runs in parallel)
$regPs = [powershell]::Create().AddScript({
    $registryPaths = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
    )
    $apps = @()
    foreach ($regPath in $registryPaths) {
        $items = Get-ItemProperty $regPath -ErrorAction SilentlyContinue |
            Where-Object { $_.DisplayName -and $_.UninstallString }
        foreach ($item in $items) {
            $sizeKB = if ($item.EstimatedSize) { [long]$item.EstimatedSize * 1024 } else { 0 }
            $installDate = if ($item.InstallDate) {
                try { [datetime]::ParseExact($item.InstallDate, 'yyyyMMdd', $null).ToString('yyyy-MM-dd') } catch { "" }
            } else { "" }
            $sizeText = if ($sizeKB -gt 0) {
                switch ($sizeKB) {
                    { $_ -ge 1TB } { '{0:N1} TB' -f ($_ / 1TB) }
                    { $_ -ge 1GB } { '{0:N1} GB' -f ($_ / 1GB) }
                    { $_ -ge 1MB } { '{0:N1} MB' -f ($_ / 1MB) }
                    { $_ -ge 1KB } { '{0:N1} KB' -f ($_ / 1KB) }
                    default        { "$_ B" }
                }
            } else { "Unknown" }
            $apps += @{
                Name               = $item.DisplayName
                Version            = if ($item.DisplayVersion) { $item.DisplayVersion } else { "" }
                Publisher          = if ($item.Publisher) { $item.Publisher } else { "" }
                InstallDate        = $installDate
                Size               = $sizeKB
                SizeText           = $sizeText
                UninstallString    = $item.UninstallString
                QuietUninstallString = if ($item.QuietUninstallString) { $item.QuietUninstallString } else { "" }
                Source             = "registry"
                WingetId           = ""
            }
        }
    }
    # Dedup by DisplayName (same app appears in HKLM, HKCU, WOW6432Node)
    $seen = @{}
    $unique = @()
    foreach ($a in $apps) {
        $k = ([string]$a.Name).ToLower().Trim()
        if ($k -and -not $seen.ContainsKey($k)) {
            $seen[$k] = $true
            $unique += $a
        }
    }
    return ,$unique
})
$regPs.RunspacePool = $scanPool
$regHandle = $regPs.BeginInvoke()

# Winget scan (runs in parallel)
$wingetPs = [powershell]::Create().AddScript({
    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) { return ,@() }
    $apps = @()
    try {
        $raw = winget list --accept-source-agreements 2>$null
        if (-not $raw) { return ,@() }
        $lines = $raw -split "`n"
        $headerIdx = -1
        for ($i = 0; $i -lt $lines.Count; $i++) {
            if ($lines[$i] -match '^Name\s+') { $headerIdx = $i; break }
        }
        if ($headerIdx -lt 0) { return ,@() }
        $sepLine = $lines[$headerIdx + 1]
        if (-not ($sepLine -match '^-')) { return ,@() }
        $dataLines = $lines[($headerIdx + 2)..($lines.Count - 1)]
        foreach ($line in $dataLines) {
            if ([string]::IsNullOrWhiteSpace($line)) { continue }
            if ($line -match '^\d+ upgrades available') { continue }
            $cols = $line -split '\s{2,}'
            if ($cols.Count -ge 2) {
                $ansi = "$([char]0x1b)\[[0-9;]*m"
                $name = ($cols[0].Trim()) -replace $ansi, ''
                $name = $name.Trim()
                if ([string]::IsNullOrWhiteSpace($name)) { continue }
                $apps += @{
                    Name    = $name
                    Id      = if ($cols.Count -ge 2) { ($cols[1].Trim()) -replace $ansi, '' } else { "" }
                    Version = if ($cols.Count -ge 3) { ($cols[2].Trim()) -replace $ansi, '' } else { "" }
                    Source  = "winget"
                }
            }
        }
    } catch {}
    # Dedup by normalized name
    $ansiRx = "$([char]0x1b)\[[^a-zA-Z]*[a-zA-Z]"
    $seen = @{}
    $unique = @()
    foreach ($a in $apps) {
        $k = (([string]$a.Name) -replace $ansiRx, '' -replace '[\x00-\x1F\x7F-\x9F]', '').ToLower().Trim()
        if ($k -and -not $seen.ContainsKey($k)) {
            $seen[$k] = $true
            $unique += $a
        }
    }
    return ,$unique
})
$wingetPs.RunspacePool = $scanPool
$wingetHandle = $wingetPs.BeginInvoke()

# Local programs scan (runs in parallel)
$localPs = [powershell]::Create().AddScript({
    $apps = @()
    $localPrograms = "$env:LOCALAPPDATA\Programs"
    if (Test-Path $localPrograms) {
        Get-ChildItem $localPrograms -Directory -ErrorAction SilentlyContinue | ForEach-Object {
            [long]$size = 0
            try {
                foreach ($f in [System.IO.Directory]::EnumerateFiles($_.FullName, '*', [System.IO.SearchOption]::AllDirectories)) {
                    try { $size += ([System.IO.FileInfo]::new($f)).Length } catch {}
                }
            } catch {}
            $sizeText = switch ($size) {
                { $_ -ge 1TB } { '{0:N1} TB' -f ($_ / 1TB) }
                { $_ -ge 1GB } { '{0:N1} GB' -f ($_ / 1GB) }
                { $_ -ge 1MB } { '{0:N1} MB' -f ($_ / 1MB) }
                { $_ -ge 1KB } { '{0:N1} KB' -f ($_ / 1KB) }
                default        { "$_ B" }
            }
            $apps += @{
                Name     = $_.Name
                Path     = $_.FullName
                Size     = $size
                SizeText = $sizeText
                Source   = "local"
            }
        }
    }
    return ,$apps
})
$localPs.RunspacePool = $scanPool
$localHandle = $localPs.BeginInvoke()

# Show progress while waiting
Write-Host "  $($C.Cyan)Scanning sources in parallel...$($C.Reset)"
$sw = [System.Diagnostics.Stopwatch]::StartNew()

# Collect registry results
$registryApps = @($regPs.EndInvoke($regHandle))
if ($registryApps.Count -eq 1 -and $registryApps[0] -is [System.Object[]]) { $registryApps = $registryApps[0] }
$regPs.Dispose()
Write-Host "  $($C.Green)✓$($C.Reset)  Registry: $($registryApps.Count) apps"

# Collect winget results
$wingetApps = @($wingetPs.EndInvoke($wingetHandle))
if ($wingetApps.Count -eq 1 -and $wingetApps[0] -is [System.Object[]]) { $wingetApps = $wingetApps[0] }
$wingetPs.Dispose()
Write-Host "  $($C.Green)✓$($C.Reset)  Winget: $($wingetApps.Count) apps"

# Collect local results
$localApps = @($localPs.EndInvoke($localHandle))
if ($localApps.Count -eq 1 -and $localApps[0] -is [System.Object[]]) { $localApps = $localApps[0] }
$localPs.Dispose()
Write-Host "  $($C.Green)✓$($C.Reset)  Local: $($localApps.Count) apps"

$sw.Stop()
$scanPool.Close()
$scanPool.Dispose()

Write-Host "  $($C.Grey)Scan completed in $($sw.ElapsedMilliseconds)ms$($C.Reset)"
Write-Host ""

$allApps = Merge-AppLists -RegistryApps $registryApps -WingetApps $wingetApps -LocalApps $localApps

# Belt-and-suspenders dedup by normalized name (catches any edge cases the merge missed)
$_ansiRx = "$([char]0x1b)\[[^a-zA-Z]*[a-zA-Z]"
$_seenNames = @{}
$allApps = @($allApps | Where-Object {
    $raw = [string](Get-ScalarValue -Value $_.Name -Default "")
    $n = (($raw -replace $_ansiRx, '') -replace '[\x00-\x1F\x7F-\x9F]', '').ToLower().Trim()
    if ($n -and -not $_seenNames.ContainsKey($n)) { $_seenNames[$n] = $true; $true } else { $false }
})

if ($allApps.Count -eq 0) {
    Write-ColorLine "  No installed applications found." -Color $C.Grey
    return
}

# Search/filter prompt
Write-Host "  $($C.Bold)Found $($allApps.Count) installed applications$($C.Reset)"
Write-Host "  $($C.Grey)Type a search term to filter, or press Enter to show all:$($C.Reset) " -NoNewline
$searchTerm = Read-Host

if ($searchTerm) {
    $allApps = @($allApps | Where-Object {
        ([string](Get-ScalarValue -Value $_.Name -Default "") -like "*$searchTerm*") -or
        ([string](Get-ScalarValue -Value $_.Publisher -Default "") -like "*$searchTerm*")
    })
    if ($allApps.Count -eq 0) {
        Write-ColorLine "  No apps matching '$searchTerm'." -Color $C.Grey
        return
    }
    Write-ColorLine "  Showing $($allApps.Count) matching apps" -Color $C.Cyan
}

Write-Host ""

# Prepare items for checkbox — use plain text source badges for correct alignment
$checkboxItems = @()
foreach ($app in $allApps) {
    $name = [string](Get-ScalarValue -Value $app.Name -Default "")
    if ([string]::IsNullOrWhiteSpace($name)) { continue }

    [long]$size = 0
    try {
        $size = [long](Get-ScalarValue -Value $app.Size -Default 0)
    } catch {
        $size = 0
    }

    $installDate = [string](Get-ScalarValue -Value $app.InstallDate -Default "")
    $source = [string](Get-ScalarValue -Value $app.Source -Default "registry")
    if ([string]::IsNullOrWhiteSpace($source)) { $source = "registry" }

    $sizeText = [string](Get-ScalarValue -Value $app.SizeText -Default "")
    if ([string]::IsNullOrWhiteSpace($sizeText)) {
        $sizeText = if ($size -gt 0) { Format-FileSize $size } else { "Unknown" }
    }

    $normalizedApp = @{
        Name                 = $name.Trim()
        Version              = [string](Get-ScalarValue -Value $app.Version -Default "")
        Publisher            = [string](Get-ScalarValue -Value $app.Publisher -Default "")
        InstallDate          = $installDate
        Size                 = $size
        SizeText             = $sizeText
        UninstallString      = [string](Get-ScalarValue -Value $app.UninstallString -Default "")
        QuietUninstallString = [string](Get-ScalarValue -Value $app.QuietUninstallString -Default "")
        Source               = $source
        WingetId             = [string](Get-ScalarValue -Value $app.WingetId -Default "")
        LocalPath            = [string](Get-ScalarValue -Value $app.LocalPath -Default "")
    }

    $age = ""
    if ($installDate) {
        try {
            $installed = [datetime]::Parse($installDate)
            $daysSince = ([datetime]::Now - $installed).Days
            if ($daysSince -gt 180) { $age = "Old" } else { $age = "Recent" }
        } catch {}
    }

    $sourceBadge = switch ($source) {
        'winget'   { "winget" }
        'registry' { "registry" }
        'local'    { "local" }
        default    { $source }
    }

    $checkboxItems += @{
        Label      = $normalizedApp.Name
        SizeText   = $normalizedApp.SizeText
        InstDate   = $normalizedApp.InstallDate
        SourceBadge = $sourceBadge
        Age        = $age
        Size       = $normalizedApp.Size
        Selected   = $false
        AppData    = $normalizedApp
    }
}

if ($checkboxItems.Count -eq 0) {
    Write-ColorLine "  No installed applications found after normalization." -Color $C.Grey
    return
}

$columns = @(
    @{ Header = "App Name"; Width = 30; Key = "Label" }
    @{ Header = "Size";     Width = 12; Key = "SizeText" }
    @{ Header = "Installed"; Width = 12; Key = "InstDate" }
    @{ Header = "Source";   Width = 10; Key = "SourceBadge" }
    @{ Header = "Age";      Width = 8;  Key = "Age" }
)

$selectedApps = Show-Checkbox -Items $checkboxItems -Title "Select apps to remove" -Columns $columns
$selectedApps = @($selectedApps | Where-Object { $null -ne $_ -and $null -ne $_.AppData })

if ($selectedApps.Count -eq 0) {
    Write-Host ""
    Write-ColorLine "  No apps selected. Cancelled." -Color $C.Grey
    return
}

Write-Host ""
$totalSize = ($selectedApps | Measure-Object -Property Size -Sum).Sum
if ($null -eq $totalSize) { $totalSize = 0 }
Write-ColorLine "  Selected $($selectedApps.Count) apps ($(Format-FileSize $totalSize))" -Color $C.Orange

if (-not (Confirm-Action "Uninstall $($selectedApps.Count) selected applications?")) {
    Write-ColorLine "  Cancelled." -Color $C.Grey
    return
}

$removedCount = 0
foreach ($item in $selectedApps) {
    Invoke-Uninstall -App $item.AppData
    $removedCount++
}

Write-Host ""
Show-Summary -MainText "Removed $removedCount applications" -SubText "Approximate freed: $(Format-FileSize $totalSize)"
