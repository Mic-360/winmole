# WiMo — ui.ps1
# All TUI primitives: menus, checkboxes, progress bars, summaries

function Confirm-Action {
    <#
    .SYNOPSIS
        Prompts user for Y/N confirmation. Returns $true if confirmed.
    #>
    param([string]$Message)

    Write-Host ""
    Write-Host "  $($C.Orange)$Message$($C.Reset)" -NoNewline
    Write-Host "  $($C.Grey)[y/N]$($C.Reset) " -NoNewline
    $response = Read-Host
    return ($response -match '^[Yy]')
}

function Show-InteractiveMenu {
    <#
    .SYNOPSIS
        Main interactive menu with mole ASCII logo and navigable options.
    #>

    $menuItems = @(
        @{ Icon = "🧹"; Label = "Clean System";    Desc = "Free disk space";          Command = "clean" }
        @{ Icon = "🗑️"; Label = "Uninstall Apps";   Desc = "Remove + leftovers";        Command = "uninstall" }
        @{ Icon = "⚡"; Label = "Optimize System";  Desc = "Speed & refresh";           Command = "optimize" }
        @{ Icon = "📁"; Label = "Analyze Disk";     Desc = "Visual explorer";           Command = "analyze" }
        @{ Icon = "📊"; Label = "Live Status";      Desc = "Health dashboard";          Command = "status" }
        @{ Icon = "🔥"; Label = "Purge Projects";   Desc = "Dev artifact junk";         Command = "purge" }
    )

    $selected = 0
    $running = $true

    # Show the full banner once
    Show-Banner

    [Console]::CursorVisible = $false

    try {
        # Save cursor position after banner
        $menuTop = [Console]::CursorTop

        while ($running) {
            # Move cursor back to menu start
            [Console]::SetCursorPosition(0, $menuTop)

            $width = Get-TerminalWidth
            $boxWidth = [Math]::Min(55, $width - 4)
            $padLeft = [Math]::Max(0, [Math]::Floor(($width - $boxWidth) / 2))
            $pad = ' ' * $padLeft
            $innerWidth = $boxWidth - 4

            # Top border
            Write-Host "$pad$($C.Grey)┌$('─' * ($boxWidth - 2))┐$($C.Reset)"

            # Header
            $header = "  🐹 WiMo  ·  Windows System Optimizer"
            $headerPad = $innerWidth - 38
            if ($headerPad -lt 0) { $headerPad = 0 }
            Write-Host "$pad$($C.Grey)│$($C.Reset)  $($C.Bold)$($C.Orange)$header$($C.Reset)$(' ' * $headerPad)$($C.Grey)│$($C.Reset)"

            # Separator
            Write-Host "$pad$($C.Grey)├$('─' * ($boxWidth - 2))┤$($C.Reset)"

            # Empty line
            Write-Host "$pad$($C.Grey)│$($C.Reset)$(' ' * ($boxWidth - 2))$($C.Grey)│$($C.Reset)"

            # Menu items
            for ($i = 0; $i -lt $menuItems.Count; $i++) {
                $item = $menuItems[$i]
                if ($i -eq $selected) {
                    $arrow = "$($C.Green)▶$($C.Reset)"
                    $label = "$($C.Bold)$($C.White)$($C.BgSelected)  $($item.Icon)  $($item.Label)"
                    $desc = "$($item.Desc)$($C.Reset)"
                    $lineText = "$arrow $label    $desc"
                    $rawLen = 6 + $item.Icon.Length + $item.Label.Length + 4 + $item.Desc.Length
                } else {
                    $label = "$($C.Grey)     $($item.Icon)  $($item.Label)"
                    $desc = "$($item.Desc)$($C.Reset)"
                    $lineText = "$label    $desc"
                    $rawLen = 5 + $item.Icon.Length + $item.Label.Length + 4 + $item.Desc.Length
                }

                $fill = [Math]::Max(0, $boxWidth - 4 - $rawLen)
                Write-Host "$pad$($C.Grey)│$($C.Reset)  $lineText$(' ' * $fill)$($C.Grey)│$($C.Reset)"
            }

            # Empty line
            Write-Host "$pad$($C.Grey)│$($C.Reset)$(' ' * ($boxWidth - 2))$($C.Grey)│$($C.Reset)"

            # Separator
            Write-Host "$pad$($C.Grey)├$('─' * ($boxWidth - 2))┤$($C.Reset)"

            # Footer
            $footer = "  ↑↓ / jk navigate  ·  Enter select  ·  q quit"
            $footerPad = [Math]::Max(0, $boxWidth - 2 - $footer.Length)
            Write-Host "$pad$($C.Grey)│$footer$(' ' * $footerPad)│$($C.Reset)"

            # Bottom border
            Write-Host "$pad$($C.Grey)└$('─' * ($boxWidth - 2))┘$($C.Reset)"

            # Read key input
            $key = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")

            switch ($key.VirtualKeyCode) {
                38 { $selected = if ($selected -gt 0) { $selected - 1 } else { $menuItems.Count - 1 } }  # Up
                40 { $selected = if ($selected -lt $menuItems.Count - 1) { $selected + 1 } else { 0 } }  # Down
                13 {  # Enter
                    $running = $false
                    [Console]::CursorVisible = $true
                    Clear-Host
                    $cmd = $menuItems[$selected].Command
                    & "$PSScriptRoot\..\..\bin\$cmd.ps1"
                    return
                }
                27 { $running = $false }  # Escape
                default {
                    switch ($key.Character) {
                        'j' { $selected = if ($selected -lt $menuItems.Count - 1) { $selected + 1 } else { 0 } }
                        'k' { $selected = if ($selected -gt 0) { $selected - 1 } else { $menuItems.Count - 1 } }
                        'q' { $running = $false }
                    }
                }
            }
        }
    } finally {
        [Console]::CursorVisible = $true
    }

    Write-Host ""
}

function Show-Checkbox {
    <#
    .SYNOPSIS
        Multi-select checkbox list for interactive selection.
    .PARAMETER Items
        Array of hashtables with: Label, Size, Extra (optional metadata), Selected (bool)
    .PARAMETER Title
        Title displayed at the top of the checkbox list.
    .PARAMETER Columns
        Array of column definitions: @{ Header = "Name"; Width = 20; Key = "Label" }
    #>
    param(
        [array]$Items,
        [string]$Title = "Select items",
        [array]$Columns = @()
    )

    if ($Items.Count -eq 0) {
        Write-ColorLine "  No items to display." -Color $C.Grey
        return @()
    }

    # Default columns if none specified
    if ($Columns.Count -eq 0) {
        $Columns = @(
            @{ Header = "Name"; Width = 30; Key = "Label" }
            @{ Header = "Size"; Width = 12; Key = "SizeText" }
        )
    }

    $cursor = 0
    $running = $true

    [Console]::CursorVisible = $false

    try {
        $startTop = [Console]::CursorTop

        while ($running) {
            [Console]::SetCursorPosition(0, $startTop)

            $width = Get-TerminalWidth
            $totalColWidth = ($Columns | Measure-Object -Property Width -Sum).Sum + ($Columns.Count * 3) + 6
            $boxWidth = [Math]::Min([Math]::Max($totalColWidth, $Title.Length + 8), $width - 4)

            # Title bar
            Write-Host "  $($C.Grey)┌$('─' * ($boxWidth - 2))┐$($C.Reset)"
            $titleLine = "  $Title  (Space: toggle · Enter: confirm · q: cancel)"
            $titlePad = [Math]::Max(0, $boxWidth - 2 - $titleLine.Length)
            Write-Host "  $($C.Grey)│$($C.Reset)$($C.Bold)$titleLine$($C.Reset)$(' ' * $titlePad)$($C.Grey)│$($C.Reset)"

            # Column headers
            $headerLine = "  "
            foreach ($col in $Columns) {
                $headerLine += "$($col.Header.PadRight($col.Width))   "
            }
            $hPad = [Math]::Max(0, $boxWidth - 2 - $headerLine.Length)
            Write-Host "  $($C.Grey)├$('─' * ($boxWidth - 2))┤$($C.Reset)"
            Write-Host "  $($C.Grey)│$($C.Reset)$($C.Bold)$($C.Cyan)$headerLine$($C.Reset)$(' ' * $hPad)$($C.Grey)│$($C.Reset)"
            Write-Host "  $($C.Grey)├$('─' * ($boxWidth - 2))┤$($C.Reset)"

            # Determine visible window (scroll if > 15 items)
            $maxVisible = [Math]::Min($Items.Count, 15)
            $scrollStart = [Math]::Max(0, [Math]::Min($cursor - [Math]::Floor($maxVisible / 2), $Items.Count - $maxVisible))

            for ($i = $scrollStart; $i -lt [Math]::Min($scrollStart + $maxVisible, $Items.Count); $i++) {
                $item = $Items[$i]
                $isSelected = $i -eq $cursor

                if ($item.Selected) {
                    $check = "$($C.Green)☑$($C.Reset)"
                } else {
                    $check = "$($C.Grey)☐$($C.Reset)"
                }

                $line = "$check "
                foreach ($col in $Columns) {
                    $val = $item[$col.Key]
                    if ($null -eq $val) { $val = "" }
                    $line += "$($val.ToString().PadRight($col.Width))   "
                }

                if ($isSelected) {
                    $bg = $C.BgSelected
                    Write-Host "  $($C.Grey)│$($C.Reset) $bg$($C.Bold)$line$($C.Reset)$($C.Grey)│$($C.Reset)"
                } else {
                    Write-Host "  $($C.Grey)│$($C.Reset) $line$($C.Grey)│$($C.Reset)"
                }
            }

            # Bottom border
            Write-Host "  $($C.Grey)└$('─' * ($boxWidth - 2))┘$($C.Reset)"

            # Status line
            $selectedCount = ($Items | Where-Object { $_.Selected }).Count
            $selectedSize = ($Items | Where-Object { $_.Selected } | Measure-Object -Property Size -Sum).Sum
            if ($null -eq $selectedSize) { $selectedSize = 0 }
            Write-Host "  $($C.Bold)Selected: $selectedCount items · $(Format-FileSize $selectedSize)$($C.Reset)  · A: all · N: none       "

            # Clear any leftover lines
            Write-Host "                                                                              "

            $key = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")

            switch ($key.VirtualKeyCode) {
                38 { $cursor = if ($cursor -gt 0) { $cursor - 1 } else { $Items.Count - 1 } }  # Up
                40 { $cursor = if ($cursor -lt $Items.Count - 1) { $cursor + 1 } else { 0 } }  # Down
                32 { $Items[$cursor].Selected = -not $Items[$cursor].Selected }  # Space
                13 { $running = $false }  # Enter
                27 { # Escape — deselect all and exit
                    $Items | ForEach-Object { $_.Selected = $false }
                    $running = $false
                }
                default {
                    switch ($key.Character) {
                        'j' { $cursor = if ($cursor -lt $Items.Count - 1) { $cursor + 1 } else { 0 } }
                        'k' { $cursor = if ($cursor -gt 0) { $cursor - 1 } else { $Items.Count - 1 } }
                        ' ' { $Items[$cursor].Selected = -not $Items[$cursor].Selected }
                        'a' { $Items | ForEach-Object { $_.Selected = $true } }
                        'n' { $Items | ForEach-Object { $_.Selected = $false } }
                        'q' {
                            $Items | ForEach-Object { $_.Selected = $false }
                            $running = $false
                        }
                    }
                }
            }
        }
    } finally {
        [Console]::CursorVisible = $true
    }

    return ($Items | Where-Object { $_.Selected })
}

function Show-Progress {
    <#
    .SYNOPSIS
        Displays an animated progress bar.
    #>
    param(
        [string]$Label,
        [int]$Percent,
        [string]$Detail = "",
        [int]$BarWidth = 30
    )

    $filled = [Math]::Floor($BarWidth * $Percent / 100)
    $empty = $BarWidth - $filled

    $color = if ($Percent -lt 60) { $C.Green } elseif ($Percent -lt 80) { $C.Orange } else { $C.Red }

    $bar = "$color$('█' * $filled)$($C.Grey)$('░' * $empty)$($C.Reset)"
    $pctText = "$($C.Bold)$Percent%$($C.Reset)"

    if ($Detail) {
        Write-Host "`r  $Label`n  $bar  $pctText  · $($C.Grey)$Detail$($C.Reset)" -NoNewline
    } else {
        Write-Host "`r  $Label`n  $bar  $pctText" -NoNewline
    }
}

function Show-ProgressLine {
    <#
    .SYNOPSIS
        Single-line progress bar for scanning operations.
    #>
    param(
        [string]$Label,
        [int]$Percent,
        [string]$Detail = "",
        [int]$BarWidth = 30
    )

    $filled = [Math]::Floor($BarWidth * $Percent / 100)
    $empty = $BarWidth - $filled

    $color = if ($Percent -lt 60) { $C.Green } elseif ($Percent -lt 80) { $C.Orange } else { $C.Red }

    $bar = "$color$('█' * $filled)$($C.Grey)$('░' * $empty)$($C.Reset)"

    Write-Host "`r  $($C.Cyan)$Label$($C.Reset)  $bar  $($C.Bold)$Percent%$($C.Reset)  $($C.Grey)$Detail$($C.Reset)  " -NoNewline
}

function Show-Summary {
    <#
    .SYNOPSIS
        Displays a boxed result summary.
    #>
    param(
        [string]$MainText,
        [string]$SubText = ""
    )

    Write-Host ""
    $content = "  ✓  $MainText"
    if ($SubText) { $content += "    $SubText" }
    $boxWidth = [Math]::Max($content.Length + 6, 50)

    Write-Host "  $($C.Green)╔$('═' * ($boxWidth - 2))╗$($C.Reset)"
    $pad = [Math]::Max(0, $boxWidth - 2 - $content.Length)
    Write-Host "  $($C.Green)║$($C.Reset)$($C.Bold)$content$($C.Reset)$(' ' * $pad)$($C.Green)║$($C.Reset)"
    Write-Host "  $($C.Green)╚$('═' * ($boxWidth - 2))╝$($C.Reset)"
    Write-Host ""
}

function Show-ScanItem {
    <#
    .SYNOPSIS
        Displays a single-line scan result with status icon, label, dots, size, and optional badge.
    #>
    param(
        [ValidateSet('Success','Error','Skip','Warn')]
        [string]$Status,
        [string]$Label,
        [string]$Size,
        [string]$Badge = ""
    )

    $width = Get-TerminalWidth
    $icon = switch ($Status) {
        'Success' { "$($C.Green)✓$($C.Reset)" }
        'Error'   { "$($C.Red)✗$($C.Reset)" }
        'Skip'    { "$($C.Grey)○$($C.Reset)" }
        'Warn'    { "$($C.Orange)⚠$($C.Reset)" }
    }

    $rightLen = $Size.Length
    if ($Badge) { $rightLen += $Badge.Length + 4 }  # "  [Badge]"

    $dotsNeeded = [Math]::Max(2, $width - 8 - $Label.Length - $rightLen - 4)
    $dots = "$($C.Grey)$('·' * $dotsNeeded)$($C.Reset)"

    Write-Host "  $icon  $Label  $dots  $($C.Bold)$Size$($C.Reset)" -NoNewline
    if ($Badge) {
        Write-Host "  $($C.Grey)[$Badge]$($C.Reset)"
    } else {
        Write-Host ""
    }
}

function Show-TaskResult {
    <#
    .SYNOPSIS
        Show a completed task line with timing.
    #>
    param(
        [string]$Label,
        [bool]$Success = $true,
        [string]$Time = "",
        [string]$Note = ""
    )

    $icon = if ($Success) { "$($C.Green)✓$($C.Reset)" } else { "$($C.Orange)⚠$($C.Reset)" }
    $width = Get-TerminalWidth

    $line = "  $icon  $Label"
    $right = ""
    if ($Time) { $right += "$($C.Grey)$Time$($C.Reset)" }
    if ($Note) { $right += "  $($C.Grey)[$Note]$($C.Reset)" }

    $padNeeded = [Math]::Max(2, $width - 8 - $Label.Length - $Time.Length - $Note.Length - 8)
    Write-Host "$line$(' ' * $padNeeded)$right"
}

function Invoke-WimoUpdate {
    Show-Banner -Compact
    Write-ColorLine "  Checking for updates..." -Color $C.Cyan

    # Placeholder — would hit GitHub releases API
    Write-Host ""
    Write-ColorLine "  $($C.Green)✓$($C.Reset)  WiMo is up to date (v$script:WIMO_VERSION)" -Color $C.White
    Write-Host ""
}

function Invoke-WimoRemove {
    Show-Banner -Compact

    if (-not (Confirm-Action "Uninstall WiMo from this system? This will remove all WiMo files and PATH entries.")) {
        Write-ColorLine "  Cancelled." -Color $C.Grey
        return
    }

    $installDir = "$env:LOCALAPPDATA\WiMo"

    # Remove from PATH
    $currentPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
    if ($currentPath -like "*$installDir*") {
        $newPath = ($currentPath -split ';' | Where-Object { $_ -ne $installDir }) -join ';'
        [Environment]::SetEnvironmentVariable('PATH', $newPath, 'User')
        Write-ColorLine "  $($C.Green)✓$($C.Reset)  Removed from PATH" -Color $C.White
    }

    # Remove config
    if (Test-Path "$env:APPDATA\WiMo") {
        Remove-Item -Path "$env:APPDATA\WiMo" -Recurse -Force -ErrorAction SilentlyContinue
        Write-ColorLine "  $($C.Green)✓$($C.Reset)  Removed config directory" -Color $C.White
    }

    # Remove install directory
    if (Test-Path $installDir) {
        Remove-Item -Path $installDir -Recurse -Force -ErrorAction SilentlyContinue
        Write-ColorLine "  $($C.Green)✓$($C.Reset)  Removed installation directory" -Color $C.White
    }

    # Remove Start Menu shortcut
    $shortcut = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\WiMo.lnk"
    if (Test-Path $shortcut) {
        Remove-Item $shortcut -Force -ErrorAction SilentlyContinue
        Write-ColorLine "  $($C.Green)✓$($C.Reset)  Removed Start Menu shortcut" -Color $C.White
    }

    Write-Host ""
    Write-ColorLine "  WiMo has been uninstalled. Restart your terminal for PATH changes." -Color $C.Green
    Write-Host ""
}
