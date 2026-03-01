# WiMo — file_ops.ps1
# 4-layer safe file operations

function Get-FolderSize {
    <#
    .SYNOPSIS
        Calculates the total size of a folder in bytes using fast .NET enumeration.
    #>
    param([string]$Path)

    if (-not (Test-Path $Path)) { return 0 }

    $item = Get-Item $Path -Force -ErrorAction SilentlyContinue
    if (-not $item) { return 0 }

    if (-not $item.PSIsContainer) {
        return $item.Length
    }

    try {
        [long]$size = 0
        foreach ($f in [System.IO.Directory]::EnumerateFiles($Path, '*', [System.IO.SearchOption]::AllDirectories)) {
            try { $size += ([System.IO.FileInfo]::new($f)).Length } catch {}
        }
        return $size
    } catch {
        return 0
    }
}

function Format-FileSize {
    <#
    .SYNOPSIS
        Formats a byte count into a human-readable string.
    #>
    param([long]$Bytes)

    if ($Bytes -lt 0) { $Bytes = 0 }

    switch ($Bytes) {
        { $_ -ge 1TB } { return '{0:N1} TB' -f ($_ / 1TB) }
        { $_ -ge 1GB } { return '{0:N1} GB' -f ($_ / 1GB) }
        { $_ -ge 1MB } { return '{0:N1} MB' -f ($_ / 1MB) }
        { $_ -ge 1KB } { return '{0:N1} KB' -f ($_ / 1KB) }
        default        { return "$Bytes B" }
    }
}

function Test-SafePath {
    <#
    .SYNOPSIS
        Validates a path through 4 safety layers before allowing deletion.
        Returns $true if safe to delete, $false otherwise.
    #>
    param([string]$Path)

    # Layer 1: Path must be non-empty
    if ([string]::IsNullOrWhiteSpace($Path)) {
        Write-WimoLog "Safety check failed: empty path" -Level Warn
        return $false
    }

    # Normalize the path
    $normalizedPath = $Path.TrimEnd('\', '/')

    # Layer 2: Must not be a protected system path
    foreach ($protected in $script:PROTECTED_PATHS) {
        $protectedNorm = $protected.TrimEnd('\', '/')
        if ($normalizedPath -ieq $protectedNorm) {
            Write-WimoLog "Safety check failed: protected path '$Path'" -Level Warn
            return $false
        }
    }

    # Layer 3: Path must exist
    if (-not (Test-Path $Path)) {
        Write-WimoLog "Safety check failed: path does not exist '$Path'" -Level Debug
        return $false
    }

    # Layer 4: Must not contain user data folder patterns
    foreach ($pattern in $script:USER_DATA_PATTERNS) {
        if ($normalizedPath -like "*$pattern*") {
            Write-WimoLog "Safety check failed: user data pattern '$pattern' in '$Path'" -Level Warn
            return $false
        }
    }

    # Check whitelist
    $config = Get-WimoConfig
    if ($config.whitelist -and $config.whitelist.Count -gt 0) {
        foreach ($whitelisted in $config.whitelist) {
            $whiteNorm = $whitelisted.TrimEnd('\', '/')
            if ($normalizedPath -ieq $whiteNorm -or $normalizedPath -like "$whiteNorm\*") {
                Write-WimoLog "Safety check: path '$Path' is whitelisted — skipping" -Level Info
                return $false
            }
        }
    }

    return $true
}

function Remove-SafePath {
    <#
    .SYNOPSIS
        Safely removes a path after passing 4-layer validation.
        Uses .NET methods for fast deletion of large directory trees.
    #>
    param(
        [string]$Path,
        [switch]$DryRun
    )

    if (-not (Test-SafePath $Path)) {
        return @{ Success = $false; BytesFreed = 0; Reason = "Failed safety check" }
    }

    $size = Get-FolderSize $Path

    if ($DryRun) {
        Write-WimoLog "Dry-run: would remove '$Path' ($(Format-FileSize $size))" -Level Info
        return @{ Success = $true; BytesFreed = $size; Reason = "Dry-run" }
    }

    try {
        Write-WimoLog "Removing: $Path" -Level Debug

        if (Test-Path $Path -PathType Container) {
            # .NET recursive delete is significantly faster than Remove-Item -Recurse
            [System.IO.Directory]::Delete($Path, $true)
        } else {
            [System.IO.File]::Delete($Path)
        }

        Write-WimoLog "Removed: $Path ($(Format-FileSize $size))" -Level Info
        return @{ Success = $true; BytesFreed = $size; Reason = "Removed" }
    } catch {
        # Fallback to Remove-Item for permission/long-path edge cases
        try {
            Remove-Item -Path $Path -Recurse -Force -ErrorAction Stop
            Write-WimoLog "Removed (fallback): $Path ($(Format-FileSize $size))" -Level Info
            return @{ Success = $true; BytesFreed = $size; Reason = "Removed" }
        } catch {
            Write-WimoLog "Failed to remove '$Path': $_" -Level Error
            return @{ Success = $false; BytesFreed = 0; Reason = $_.Exception.Message }
        }
    }
}

function Remove-SafeGlob {
    <#
    .SYNOPSIS
        Safely removes files matching a glob pattern.
    #>
    param(
        [string]$Pattern,
        [switch]$DryRun
    )

    $totalFreed = 0
    $items = Get-Item -Path $Pattern -ErrorAction SilentlyContinue

    foreach ($item in $items) {
        $result = Remove-SafePath -Path $item.FullName -DryRun:$DryRun
        if ($result.Success) {
            $totalFreed += $result.BytesFreed
        }
    }

    return $totalFreed
}

function Get-PathSize {
    <#
    .SYNOPSIS
        Gets the size of a path (supports glob patterns). Uses .NET for speed.
    #>
    param([string]$Path)

    [long]$total = 0
    $items = Get-Item -Path $Path -Force -ErrorAction SilentlyContinue

    foreach ($item in $items) {
        if ($item.PSIsContainer) {
            $total += Get-FolderSize $item.FullName
        } else {
            $total += $item.Length
        }
    }

    return $total
}
