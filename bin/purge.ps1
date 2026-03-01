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

    $found = @()

    if (-not (Test-Path $ScanPath)) { return $found }

    Write-WimoLog "Scanning: $ScanPath (depth: $MaxDepth)" -Level Debug

    # Walk directories
    $dirs = Get-ChildItem -Path $ScanPath -Directory -Recurse -Depth $MaxDepth -Force -ErrorAction SilentlyContinue

    $totalDirs = $dirs.Count
    $processed = 0

    foreach ($dir in $dirs) {
        $processed++
        $pct = if ($totalDirs -gt 0) { [Math]::Floor($processed / $totalDirs * 100) } else { 0 }

        if ($processed % 50 -eq 0) {
            Show-ProgressLine -Label "Scanning..." -Percent $pct -Detail "$processed / $totalDirs dirs"
        }

        $dirName = $dir.Name

        # Direct artifact match
        if ($ArtifactDirs.ContainsKey($dirName)) {
            $size = Get-FolderSize $dir.FullName
            if ($size -gt 0) {
                $parentProject = Split-Path $dir.Parent.FullName -Leaf
                $daysSinceModified = ([datetime]::Now - $dir.LastWriteTime).Days
                $isRecent = $daysSinceModified -lt 7

                $found += @{
                    Label       = "$parentProject/$dirName"
                    Path        = $dir.FullName
                    Size        = $size
                    SizeText    = Format-FileSize $size
                    Type        = $ArtifactDirs[$dirName]
                    Recent      = $isRecent
                    Selected    = -not $isRecent  # Leave recent items unchecked
                    DaysOld     = $daysSinceModified
                }
            }
        }

        # Context-sensitive artifact match
        if ($ContextArtifacts.ContainsKey($dirName)) {
            $ctx = $ContextArtifacts[$dirName]
            $parentDir = $dir.Parent.FullName
            $hasIndicator = $false
            foreach ($indicator in $ctx.Indicators) {
                if (Test-Path (Join-Path $parentDir $indicator)) {
                    $hasIndicator = $true
                    break
                }
            }
            if ($hasIndicator) {
                $size = Get-FolderSize $dir.FullName
                if ($size -gt 0) {
                    $parentProject = Split-Path $parentDir -Leaf
                    $daysSinceModified = ([datetime]::Now - $dir.LastWriteTime).Days
                    $isRecent = $daysSinceModified -lt 7

                    $found += @{
                        Label       = "$parentProject/$dirName"
                        Path        = $dir.FullName
                        Size        = $size
                        SizeText    = Format-FileSize $size
                        Type        = $ctx.Label
                        Recent      = $isRecent
                        Selected    = -not $isRecent
                        DaysOld     = $daysSinceModified
                    }
                }
            }
        }

        # Flutter project detection
        if (Test-Path (Join-Path $dir.FullName "pubspec.yaml")) {
            foreach ($subPath in $FlutterSubArtifacts) {
                $fullSubPath = Join-Path $dir.FullName $subPath
                if (Test-Path $fullSubPath) {
                    $size = Get-FolderSize $fullSubPath
                    if ($size -gt 0) {
                        $projectName = $dir.Name
                        $found += @{
                            Label       = "$projectName/$subPath"
                            Path        = $fullSubPath
                            Size        = $size
                            SizeText    = Format-FileSize $size
                            Type        = "Flutter build artifact"
                            Recent      = $false
                            Selected    = $true
                            DaysOld     = ([datetime]::Now - (Get-Item $fullSubPath).LastWriteTime).Days
                        }
                    }
                }
            }
        }
    }

    # Clear progress line
    Write-Host "`r$(' ' * 80)`r" -NoNewline

    return $found
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

foreach ($artifact in $selectedArtifacts) {
    $result = Remove-SafePath -Path $artifact.Path
    if ($result.Success) {
        Show-ScanItem -Status Success -Label $artifact.Label -Size $artifact.SizeText
        $freedTotal += $result.BytesFreed
        $removedCount++
    } else {
        Show-ScanItem -Status Error -Label $artifact.Label -Size $artifact.SizeText -Badge $result.Reason
    }
}

Write-Host ""
Show-Summary -MainText "Purged $removedCount directories" -SubText "Freed: $(Format-FileSize $freedTotal)"
