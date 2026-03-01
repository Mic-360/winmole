# WiMo — purge.ps1
# Clean project build artifacts (all major stacks including Flutter)

# Parse args
$Paths = @()
$Depth = 0
for ($i = 0; $i -lt $args.Count; $i++) {
    if ($args[$i] -eq '--paths' -and ($i + 1) -lt $args.Count) {
        $Paths = $args[$i + 1] -split ','
        $i++
    }
    if ($args[$i] -eq '--depth' -and ($i + 1) -lt $args.Count) {
        $Depth = [int]$args[$i + 1]
        $i++
    }
}

# Ensure core libs are loaded
if (-not $script:WIMO_VERSION) {
    . "$PSScriptRoot\..\lib\core\common.ps1"
}

# Artifact definitions: directory/file name => type label
$ArtifactDirs = @{
    # JavaScript / TypeScript / Node.js
    "node_modules"       = "Node.js dependencies"
    ".next"              = "Next.js build cache"
    ".nuxt"              = "Nuxt.js build cache"
    ".output"            = "Nuxt 3 / Nitro output"
    ".tanstack"          = "TanStack cache"
    ".svelte-kit"        = "SvelteKit cache"
    ".astro"             = "Astro build cache"
    ".remix"             = "Remix cache"
    ".turbo"             = "Turborepo cache"
    ".cache"             = "Generic cache dir"
    ".parcel-cache"      = "Parcel bundler cache"
    "storybook-static"   = "Storybook build output"
    ".docusaurus"        = "Docusaurus cache"
    "coverage"           = "Test coverage reports"
    ".nyc_output"        = "Istanbul/nyc coverage"

    # Python
    "__pycache__"        = "Python bytecode"
    ".pytest_cache"      = "pytest cache"
    ".mypy_cache"        = "mypy cache"
    ".ruff_cache"        = "Ruff cache"
    ".tox"               = "tox environments"
    "venv"               = "Python virtualenv"
    ".venv"              = "Python virtualenv"
    ".eggs"              = "Python eggs"

    # Rust
    "target"             = "Build artifacts"

    # Java / Kotlin / JVM
    ".gradle"            = "Gradle cache"

    # Go
    "vendor"             = "Go vendor"

    # Flutter / Dart
    ".dart_tool"         = "Dart tool cache"
    "ios/Pods"           = "CocoaPods cache"
    "ios/.symlinks"      = "Flutter iOS symlinks"

    # C / C++ / CMake
    "CMakeFiles"         = "CMake build files"
    "cmake-build-debug"  = "CMake debug build"
    "cmake-build-release"= "CMake release build"

    # Ruby
    ".bundle"            = "Bundler cache"

    # General
    ".DS_Store"          = "macOS metadata"
    "Thumbs.db"          = "Windows thumbnails"
}

# Context-sensitive artifacts — only match if specific project files exist nearby
$ContextArtifacts = @{
    "dist"  = @{ Label = "Build output"; Indicators = @("package.json", "tsconfig.json", "setup.py", "pyproject.toml") }
    "build" = @{ Label = "Build output"; Indicators = @("package.json", "build.gradle", "pom.xml", "pubspec.yaml", "CMakeLists.txt") }
    "out"   = @{ Label = "Build output"; Indicators = @("package.json", "tsconfig.json", "next.config.js") }
}

# Flutter-specific subdirectory artifacts
$FlutterSubArtifacts = @(
    "android/.gradle",
    "android/build",
    "android/app/build",
    "ios/Flutter/ephemeral",
    "linux/flutter/ephemeral",
    "macos/Flutter/ephemeral",
    "windows/flutter/ephemeral",
    "web/.dart_tool"
)

function Find-Artifacts {
    param(
        [string]$ScanPath,
        [int]$MaxDepth
    )

    $found = [System.Collections.ArrayList]::new()

    if (-not (Test-Path $ScanPath)) { return @() }

    Write-WimoLog "Scanning: $ScanPath (depth: $MaxDepth)" -Level Debug

    # Fast .NET directory walk — collect candidate paths first, defer sizing
    $candidates = [System.Collections.ArrayList]::new()
    $artifactKeys = $ArtifactDirs.Keys
    $contextKeys  = $ContextArtifacts.Keys

    function Walk-Dir {
        param([string]$Dir, [int]$CurrentDepth)
        if ($CurrentDepth -gt $MaxDepth) { return }
        try {
            foreach ($sub in [System.IO.Directory]::EnumerateDirectories($Dir)) {
                $name = [System.IO.Path]::GetFileName($sub)

                if ($artifactKeys -contains $name) {
                    [void]$candidates.Add(@{ Path = $sub; Name = $name; Kind = 'direct' })
                    continue  # don't recurse into artifact dirs
                }

                if ($contextKeys -contains $name) {
                    $ctx = $ContextArtifacts[$name]
                    $parentDir = [System.IO.Path]::GetDirectoryName($sub)
                    foreach ($indicator in $ctx.Indicators) {
                        if ([System.IO.File]::Exists([System.IO.Path]::Combine($parentDir, $indicator))) {
                            [void]$candidates.Add(@{ Path = $sub; Name = $name; Kind = 'context' })
                            break
                        }
                    }
                    continue
                }

                # Flutter detection
                if ([System.IO.File]::Exists([System.IO.Path]::Combine($sub, 'pubspec.yaml'))) {
                    foreach ($subRel in $FlutterSubArtifacts) {
                        $fullSubPath = [System.IO.Path]::Combine($sub, $subRel)
                        if ([System.IO.Directory]::Exists($fullSubPath)) {
                            [void]$candidates.Add(@{ Path = $fullSubPath; Name = $subRel; Kind = 'flutter'; ProjectDir = $sub })
                        }
                    }
                }

                Walk-Dir -Dir $sub -CurrentDepth ($CurrentDepth + 1)
            }
        } catch {}
    }

    Walk-Dir -Dir $ScanPath -CurrentDepth 0

    if ($candidates.Count -eq 0) {
        Write-Host "`r$(' ' * 80)`r" -NoNewline
        return @()
    }

    # Parallel size calculation using runspace pool
    $poolSize = [Math]::Min(16, [Math]::Max(1, $candidates.Count))
    $pool = [runspacefactory]::CreateRunspacePool(1, $poolSize)
    $pool.Open()

    $sizeJobs = @()
    foreach ($c in $candidates) {
        $ps = [powershell]::Create().AddScript({
            param($p)
            [long]$sz = 0
            try {
                foreach ($f in [System.IO.Directory]::EnumerateFiles($p, '*', [System.IO.SearchOption]::AllDirectories)) {
                    try { $sz += ([System.IO.FileInfo]::new($f)).Length } catch {}
                }
            } catch {}
            return $sz
        }).AddArgument($c.Path)
        $ps.RunspacePool = $pool
        $sizeJobs += @{ Pipe = $ps; Handle = $ps.BeginInvoke(); Candidate = $c }
    }

    $totalCandidates = $sizeJobs.Count
    $processed = 0

    foreach ($job in $sizeJobs) {
        $size = $job.Pipe.EndInvoke($job.Handle)
        if ($null -eq $size) { $size = 0 } else { $size = [long]$size }
        $job.Pipe.Dispose()
        $c = $job.Candidate
        $processed++

        if ($processed % 20 -eq 0) {
            $pct = [Math]::Floor($processed / $totalCandidates * 100)
            Show-ProgressLine -Label "Sizing..." -Percent $pct -Detail "$processed / $totalCandidates dirs"
        }

        if ($size -le 0) { continue }

        $dirInfo = [System.IO.DirectoryInfo]::new($c.Path)
        $daysSinceModified = ([datetime]::Now - $dirInfo.LastWriteTime).Days
        $isRecent = $daysSinceModified -lt 7

        switch ($c.Kind) {
            'direct' {
                $parentProject = [System.IO.Path]::GetFileName($dirInfo.Parent.FullName)
                [void]$found.Add(@{
                    Label       = "$parentProject/$($c.Name)"
                    Path        = $c.Path
                    Size        = $size
                    SizeText    = Format-FileSize $size
                    Type        = $ArtifactDirs[$c.Name]
                    Recent      = $isRecent
                    Selected    = -not $isRecent
                    DaysOld     = $daysSinceModified
                })
            }
            'context' {
                $parentDir = [System.IO.Path]::GetDirectoryName($c.Path)
                $parentProject = [System.IO.Path]::GetFileName($parentDir)
                $ctx = $ContextArtifacts[$c.Name]
                [void]$found.Add(@{
                    Label       = "$parentProject/$($c.Name)"
                    Path        = $c.Path
                    Size        = $size
                    SizeText    = Format-FileSize $size
                    Type        = $ctx.Label
                    Recent      = $isRecent
                    Selected    = -not $isRecent
                    DaysOld     = $daysSinceModified
                })
            }
            'flutter' {
                $projectName = [System.IO.Path]::GetFileName($c.ProjectDir)
                [void]$found.Add(@{
                    Label       = "$projectName/$($c.Name)"
                    Path        = $c.Path
                    Size        = $size
                    SizeText    = Format-FileSize $size
                    Type        = "Flutter build artifact"
                    Recent      = $false
                    Selected    = $true
                    DaysOld     = $daysSinceModified
                })
            }
        }
    }

    $pool.Close()
    $pool.Dispose()

    # Clear progress line
    Write-Host "`r$(' ' * 80)`r" -NoNewline

    return $found.ToArray()
}

# Main execution
Show-Banner -Compact

Write-ColorLine "  $($C.Bold)🔥 WiMo Purge  ·  Scanning project directories...$($C.Reset)" -Color $C.White
Write-Host ""

# Determine scan paths
$config = Get-WimoConfig
$scanPaths = if ($Paths -and $Paths.Count -gt 0) {
    $Paths
} elseif ($config.scan_paths) {
    $config.scan_paths
} else {
    @("$env:USERPROFILE\Projects", "$env:USERPROFILE\Documents\dev")
}

$maxDepth = if ($Depth -gt 0) { $Depth } elseif ($config.purge_depth) { $config.purge_depth } else { 8 }

$activePaths = @()
foreach ($sp in $scanPaths) {
    if (Test-Path $sp) {
        $activePaths += $sp
        Write-ColorLine "  Scan path: $($C.Cyan)$sp$($C.Reset)" -Color $C.White
    } else {
        Write-ColorLine "  Scan path: $($C.Grey)$sp (not found — skipping)$($C.Reset)" -Color $C.Grey
    }
}

if ($activePaths.Count -eq 0) {
    Write-Host ""
    Write-ColorLine "  No valid scan paths found. Use --paths to specify directories." -Color $C.Orange
    Write-ColorLine "  Example: wimo purge --paths `"C:\MyProjects`"" -Color $C.Grey
    return
}

Write-Host ""

# Scan all paths
$allArtifacts = @()
foreach ($scanPath in $activePaths) {
    $artifacts = Find-Artifacts -ScanPath $scanPath -MaxDepth $maxDepth
    $allArtifacts += $artifacts
}

if ($allArtifacts.Count -eq 0) {
    Write-ColorLine "  No build artifacts found. Your projects are clean!" -Color $C.Green
    return
}

# Sort by size descending
$allArtifacts = $allArtifacts | Sort-Object { $_.Size } -Descending

$totalSize = ($allArtifacts | Measure-Object -Property Size -Sum).Sum
$projectCount = ($allArtifacts | ForEach-Object { Split-Path (Split-Path $_.Path -Parent) -Leaf } | Sort-Object -Unique).Count

Write-ColorLine "  Found $($allArtifacts.Count) artifact directories across $projectCount projects" -Color $C.White
Write-Host ""

# Show checkbox
$columns = @(
    @{ Header = "Path";  Width = 35; Key = "Label" }
    @{ Header = "Size";  Width = 12; Key = "SizeText" }
    @{ Header = "Type";  Width = 25; Key = "Type" }
)

$selectedArtifacts = Show-Checkbox -Items $allArtifacts -Title "Select artifacts to purge" -Columns $columns

if ($selectedArtifacts.Count -eq 0) {
    Write-Host ""
    Write-ColorLine "  No items selected. Cancelled." -Color $C.Grey
    return
}

$selectedSize = ($selectedArtifacts | Measure-Object -Property Size -Sum).Sum
Write-Host ""

if (-not (Confirm-Action "Remove $($selectedArtifacts.Count) artifact directories ($(Format-FileSize $selectedSize))?")) {
    Write-ColorLine "  Cancelled." -Color $C.Grey
    return
}

Write-Host ""
$freedTotal = [long]0
$removedCount = 0

# Parallel deletion using runspace pool for speed
$delPool = [runspacefactory]::CreateRunspacePool(1, [Math]::Min(8, $selectedArtifacts.Count))
$delPool.Open()

$delJobs = @()
foreach ($artifact in $selectedArtifacts) {
    $ps = [powershell]::Create().AddScript({
        param($p)
        try {
            if ([System.IO.Directory]::Exists($p)) {
                [System.IO.Directory]::Delete($p, $true)
                return @{ Success = $true; Error = $null }
            }
            return @{ Success = $false; Error = "Not found" }
        } catch {
            # Fallback
            try {
                Remove-Item -Path $p -Recurse -Force -ErrorAction Stop
                return @{ Success = $true; Error = $null }
            } catch {
                return @{ Success = $false; Error = $_.Exception.Message }
            }
        }
    }).AddArgument($artifact.Path)
    $ps.RunspacePool = $delPool
    $delJobs += @{ Pipe = $ps; Handle = $ps.BeginInvoke(); Artifact = $artifact }
}

foreach ($job in $delJobs) {
    $result = $job.Pipe.EndInvoke($job.Handle)
    $job.Pipe.Dispose()
    $a = $job.Artifact
    if ($null -ne $result -and $result.Success) {
        Show-ScanItem -Status Success -Label $a.Label -Size $a.SizeText
        $freedTotal += $a.Size
        $removedCount++
    } else {
        $reason = if ($null -ne $result) { $result.Error } else { "Unknown error" }
        Show-ScanItem -Status Error -Label $a.Label -Size $a.SizeText -Badge $reason
    }
}

$delPool.Close()
$delPool.Dispose()

Write-Host ""
Show-Summary -MainText "Purged $removedCount directories" -SubText "Freed: $(Format-FileSize $freedTotal)"
