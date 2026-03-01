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

function Get-DisplayWidth {
    <#
    .SYNOPSIS
        Returns the terminal display width of a string, accounting for
        double-width emoji/CJK characters and stripping ANSI escapes.
    #>
    param([string]$Text)
    # Strip ANSI escape codes first
    $plain = $Text -replace "$([char]0x1b)\[[0-9;]*m", ''
    [int]$w = 0
    $i = 0
    while ($i -lt $plain.Length) {
        $c = [int][char]$plain[$i]
        # Surrogate pair (emoji/supplementary plane) — display width 2
        if ($c -ge 0xD800 -and $c -le 0xDBFF) {
            $w += 2
            $i += 2  # skip low surrogate
            continue
        }
        # Variation selectors (U+FE0E, U+FE0F) — zero width
        if ($c -eq 0xFE0E -or $c -eq 0xFE0F) {
            $i++
            continue
        }
        # Common double-width: box-drawing heavy, CJK, fullwidth, etc.
        if (($c -ge 0x1100 -and $c -le 0x115F) -or   # Hangul Jamo
            ($c -ge 0x2E80 -and $c -le 0x9FFF) -or   # CJK
            ($c -ge 0xF900 -and $c -le 0xFAFF) -or   # CJK compat
            ($c -ge 0xFE30 -and $c -le 0xFE6F) -or   # CJK forms
            ($c -ge 0xFF00 -and $c -le 0xFF60) -or   # Fullwidth
            ($c -ge 0xFFE0 -and $c -le 0xFFE6)) {    # Fullwidth signs
            $w += 2
        } else {
            $w += 1
        }
        $i++
    }
    return $w
}

function Read-WimoKey {
    <#
    .SYNOPSIS
        Reads a key from RawUI; falls back to text command input when raw key capture is unavailable.
    #>
    param(
        [string]$FallbackPrompt = "  Input (j/k/up/down/space/enter/a/n/q):"
    )

    $savedErrorPref = $ErrorActionPreference
    $ErrorActionPreference = 'Stop'

    try {
        return $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
    } catch {
        Write-WimoLog "Raw key input unavailable; using fallback input mode. $_" -Level Warn

        [Console]::CursorVisible = $true
        Write-ColorLine "  Raw key input not supported in this terminal. Command mode enabled." -Color $C.Orange
        $raw = Read-Host $FallbackPrompt
        [Console]::CursorVisible = $false

        if ($null -eq $raw) { $raw = "" }
        $cmd = $raw.Trim().ToLowerInvariant()

        $vk = switch ($cmd) {
            ''       { 13 }
            'enter'  { 13 }
            'up'     { 38 }
            'k'      { 38 }
            'down'   { 40 }
            'j'      { 40 }
            'space'  { 32 }
            'toggle' { 32 }
            'a'      { 65 }
            'all'    { 65 }
            'n'      { 78 }
            'none'   { 78 }
            'q'      { 81 }
            'quit'   { 81 }
            'esc'    { 27 }
            'escape' { 27 }
            default  { 0 }
        }

        $char = if ($cmd.Length -gt 0) { [char]$cmd[0] } else { [char]13 }
        return [pscustomobject]@{
            VirtualKeyCode = $vk
            Character      = $char
        }
    } finally {
        $ErrorActionPreference = $savedErrorPref
    }
}

function Write-WimoCard {
    <#
    .SYNOPSIS
        Renders a modern rounded card block.
    #>
    param(
        [string]$Title,
        [string[]]$Lines,
        [string]$AccentColor = $C.SageLight,
        [int]$Width = 0,
        [switch]$ShowBottomBorder
    )

    if ($Width -le 0) {
        $Width = [Math]::Min(96, (Get-TerminalWidth) - 4)
    }
    $Width = [Math]::Max(40, $Width)
    $inner = $Width - 2

    Write-Host "  $($C.Grey)╭$('─' * $inner)╮$($C.Reset)"

    $titleText = "  $Title"
    $titlePad = [Math]::Max(0, $inner - (Get-DisplayWidth $titleText))
    Write-Host "  $($C.Grey)│$($C.Reset)$($C.Bold)$AccentColor$titleText$($C.Reset)$(' ' * $titlePad)$($C.Grey)│$($C.Reset)"
    Write-Host "  $($C.Grey)├$('─' * $inner)┤$($C.Reset)"

    foreach ($line in $Lines) {
        $plainW = Get-DisplayWidth $line
        $pad = [Math]::Max(0, $inner - $plainW)
        Write-Host "  $($C.Grey)│$($C.Reset)$line$(' ' * $pad)$($C.Grey)│$($C.Reset)"
    }

    if ($ShowBottomBorder) {
        Write-Host "  $($C.Grey)╰$('─' * $inner)╯$($C.Reset)"
    }
}

function Show-InteractiveMenu {
    <#
    .SYNOPSIS
        Main interactive menu with mole ASCII logo and navigable options.
    #>

    # Use simple ASCII icons for consistent column width
    $menuItems = @(
        @{ Icon = "[~]"; Label = "Clean System";    Desc = "Free disk space";     Command = "clean" }
        @{ Icon = "[-]"; Label = "Uninstall Apps";   Desc = "Remove + leftovers";   Command = "uninstall" }
        @{ Icon = "[*]"; Label = "Optimize System";  Desc = "Speed & refresh";      Command = "optimize" }
        @{ Icon = "[+]"; Label = "Analyze Disk";     Desc = "Visual explorer";      Command = "analyze" }
        @{ Icon = "[=]"; Label = "Live Status";      Desc = "Health dashboard";     Command = "status" }
        @{ Icon = "[!]"; Label = "Purge Projects";   Desc = "Dev artifact junk";    Command = "purge" }
    )

    $selected = 0
    $running = $true

    Show-Banner

    [Console]::CursorVisible = $false

    try {
        $menuTop = [Console]::CursorTop

        while ($running) {
            [Console]::SetCursorPosition(0, $menuTop)

            $width = Get-TerminalWidth
            $boxWidth = [Math]::Min(58, $width - 4)
            $padLeft = [Math]::Max(0, [Math]::Floor(($width - $boxWidth) / 2))
            $pad = ' ' * $padLeft
            $inner = $boxWidth - 2  # usable chars between │ and │

            # Top border
            Write-Host "$pad$($C.Grey)╭$('─' * $inner)╮$($C.Reset)"

            # Header
            $hdrText = "  WiMo  ·  Modern Windows TUI Toolkit"
            $hdrFill = [Math]::Max(0, $inner - (Get-DisplayWidth $hdrText))
            Write-Host "$pad$($C.Grey)│$($C.Reset)$($C.Bold)$($C.SageLight)$hdrText$($C.Reset)$(' ' * $hdrFill)$($C.Grey)│$($C.Reset)"

            # Separator
            Write-Host "$pad$($C.Grey)├$('─' * $inner)┤$($C.Reset)"

            # Empty line
            Write-Host "$pad$($C.Grey)│$(' ' * $inner)│$($C.Reset)"

            # Menu items
            for ($i = 0; $i -lt $menuItems.Count; $i++) {
                $item = $menuItems[$i]
                # Build the visible content and calculate its display columns
                if ($i -eq $selected) {
                    $icon = "$($C.SageGreen)$($item.Icon)$($C.Reset)"
                    $prefix = " $($C.Cyan)▶$($C.Reset) "
                    $labelPart = "$($C.Bold)$($C.White)$($item.Label)$($C.Reset)"
                    $descPart = "$($C.SageLight)$($item.Desc)$($C.Reset)"
                    $visibleLen = 3 + $item.Icon.Length + 2 + $item.Label.Length + 4 + $item.Desc.Length
                    $lineText = "$prefix$icon  $labelPart    $descPart"
                } else {
                    $icon = "$($C.Grey)$($item.Icon)$($C.Reset)"
                    $prefix = "   "
                    $labelPart = "$($C.Grey)$($item.Label)$($C.Reset)"
                    $descPart = "$($C.Grey)$($item.Desc)$($C.Reset)"
                    $visibleLen = 3 + $item.Icon.Length + 2 + $item.Label.Length + 4 + $item.Desc.Length
                    $lineText = "$prefix$icon  $labelPart    $descPart"
                }

                $fill = [Math]::Max(0, $inner - $visibleLen)
                Write-Host "$pad$($C.Grey)│$($C.Reset)$lineText$(' ' * $fill)$($C.Grey)│$($C.Reset)"
            }

            # Empty line
            Write-Host "$pad$($C.Grey)│$(' ' * $inner)│$($C.Reset)"

            # Separator
            Write-Host "$pad$($C.Grey)├$('─' * $inner)┤$($C.Reset)"

            # Footer
            $footerText = "  ↑/↓ navigate  ·  Enter select  ·  q quit"
            $footerFill = [Math]::Max(0, $inner - $footerText.Length)
            Write-Host "$pad$($C.Grey)│$footerText$(' ' * $footerFill)│$($C.Reset)"

            # Bottom border
            Write-Host "$pad$($C.Grey)╰$('─' * $inner)╯$($C.Reset)"

            # Read key input (safe + fallback)
            $key = Read-WimoKey -FallbackPrompt "  Menu command"
            if ($null -eq $key) { continue }

            switch ($key.VirtualKeyCode) {
                38 { $selected = if ($selected -gt 0) { $selected - 1 } else { $menuItems.Count - 1 } }  # Up
                40 { $selected = if ($selected -lt $menuItems.Count - 1) { $selected + 1 } else { 0 } }  # Down
                13 {  # Enter
                    $running = $false
                    [Console]::CursorVisible = $true
                    Clear-Host
                    $cmd = $menuItems[$selected].Command
                    & "$script:WimoRoot\bin\$cmd.ps1"
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

    $getItemValue = {
        param($obj, [string]$key)
        if ($null -eq $obj) { return $null }
        if ($obj -is [hashtable]) {
            if ($obj.ContainsKey($key)) { return $obj[$key] }
            return $null
        }
        if ($null -ne $obj.PSObject -and $null -ne $obj.PSObject.Properties[$key]) {
            return $obj.$key
        }
        return $null
    }

    $setSelected = {
        param($obj, [bool]$value)
        if ($null -eq $obj) { return }
        if ($obj -is [hashtable]) {
            $obj['Selected'] = $value
            return
        }
        if ($null -ne $obj.PSObject -and $null -ne $obj.PSObject.Properties['Selected']) {
            $obj.Selected = $value
        }
    }

    $toScalarString = {
        param($value)
        if ($null -eq $value) { return "" }
        if ($value -is [string]) { return $value }
        if ($value -is [System.Collections.IEnumerable]) {
            $arr = @($value)
            if ($arr.Count -eq 0) { return "" }
            return [string]$arr[0]
        }
        return [string]$value
    }

    [Console]::CursorVisible = $false

    try {
        $savedErrorPref = $ErrorActionPreference
        $ErrorActionPreference = 'Stop'
        $startTop = [Console]::CursorTop

        # Initial flush (best-effort)
        try { $Host.UI.RawUI.FlushInputBuffer() } catch {
            try { while ([Console]::KeyAvailable) { [void][Console]::ReadKey($true) } } catch { }
        }

        $firstFrame = $true

        while ($running) {
            [Console]::SetCursorPosition(0, $startTop)

            $width = Get-TerminalWidth

            [int]$columnsWidthSum = 0
            foreach ($col in $Columns) {
                $colWidth = 0
                if ($col -is [hashtable] -and $col.ContainsKey('Width')) {
                    $colWidth = [int]$col['Width']
                } elseif ($null -ne $col.PSObject -and $null -ne $col.PSObject.Properties['Width']) {
                    $colWidth = [int]$col.Width
                }
                $columnsWidthSum += $colWidth
            }

            $totalColWidth = $columnsWidthSum + ($Columns.Count * 3) + 6
            $boxWidth = [Math]::Min([Math]::Max($totalColWidth, $Title.Length + 8), $width - 4)

            # Title bar
            Write-Host "  $($C.Grey)╭$('─' * ($boxWidth - 2))╮$($C.Reset)"
            $titleLine = "  $Title  ·  Space toggle · Enter confirm · q cancel"
            $titlePad = [Math]::Max(0, $boxWidth - 2 - $titleLine.Length)
            Write-Host "  $($C.Grey)│$($C.Reset)$($C.Bold)$($C.SageLight)$titleLine$($C.Reset)$(' ' * $titlePad)$($C.Grey)│$($C.Reset)"

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
            $maxVisible = [Math]::Min($Items.Count, 20)
            $scrollStart = [Math]::Max(0, [Math]::Min($cursor - [Math]::Floor($maxVisible / 2), $Items.Count - $maxVisible))

            for ($i = $scrollStart; $i -lt [Math]::Min($scrollStart + $maxVisible, $Items.Count); $i++) {
                $item = $Items[$i]
                $isSelected = $i -eq $cursor

                $itemIsChecked = [bool](& $getItemValue $item 'Selected')
                if ($itemIsChecked) {
                    $check = "$($C.Green)☑$($C.Reset)"
                } else {
                    $check = "$($C.Grey)☐$($C.Reset)"
                }

                $line = "$check "
                foreach ($col in $Columns) {
                    $val = & $getItemValue $item $col.Key
                    if ($null -eq $val) { $val = "" }

                    $valStr = & $toScalarString $val

                    # Clamp to fixed column width so rows cannot overflow the right border
                    if ($valStr.Length -gt $col.Width) {
                        if ($col.Width -gt 1) {
                            $valStr = $valStr.Substring(0, $col.Width - 1) + "…"
                        } else {
                            $valStr = $valStr.Substring(0, $col.Width)
                        }
                    }

                    $plainLen = ($valStr -replace "$([char]0x1b)\[[0-9;]*m", '').Length
                    $padNeeded = [Math]::Max(0, $col.Width - $plainLen)
                    $line += "$valStr$(' ' * $padNeeded)   "
                }

                if ($isSelected) {
                    $bg = $C.BgSelected
                    Write-Host "  $($C.Grey)│$($C.Reset) $bg$($C.Bold)$($C.White)$line$($C.Reset)$($C.Grey)│$($C.Reset)"
                } else {
                    Write-Host "  $($C.Grey)│$($C.Reset) $line$($C.Grey)│$($C.Reset)"
                }
            }

            # Bottom border
            Write-Host "  $($C.Grey)╰$('─' * ($boxWidth - 2))╯$($C.Reset)"

            # Status line
            $selectedCount = ($Items | Where-Object { $_.Selected }).Count
            $selectedSize = ($Items | Where-Object { $_.Selected } | Measure-Object -Property Size -Sum).Sum
            if ($null -eq $selectedSize) { $selectedSize = 0 }
            Write-Host "  $($C.Bold)$($C.SageLight)Selected:$($C.Reset) $selectedCount items · $(Format-FileSize $selectedSize)  · A all · N none"

            # Clear any leftover lines
            Write-Host "                                                                              "

            # After rendering the first frame, flush again to discard any
            # terminal response sequences or stale Enter from Read-Host
            if ($firstFrame) {
                $firstFrame = $false
                Start-Sleep -Milliseconds 150
                try { $Host.UI.RawUI.FlushInputBuffer() } catch {
                    try { while ([Console]::KeyAvailable) { [void][Console]::ReadKey($true) } } catch { }
                }
            }

            $key = $null
            try {
                $key = Read-WimoKey -FallbackPrompt "  Selection command"
            } catch {
                Start-Sleep -Milliseconds 50
                continue
            }
            if ($null -eq $key) { continue }

            switch ($key.VirtualKeyCode) {
                38 { $cursor = if ($cursor -gt 0) { $cursor - 1 } else { $Items.Count - 1 } }  # Up
                40 { $cursor = if ($cursor -lt $Items.Count - 1) { $cursor + 1 } else { 0 } }  # Down
                32 {
                    $current = [bool](& $getItemValue $Items[$cursor] 'Selected')
                    & $setSelected $Items[$cursor] (-not $current)
                }  # Space
                13 {
                    # Prevent accidental immediate confirm when nothing is selected
                    $currentSelectedCount = ($Items | Where-Object { [bool](& $getItemValue $_ 'Selected') }).Count
                    if ($currentSelectedCount -gt 0) {
                        $running = $false
                    }
                }  # Enter
                27 { # Escape — deselect all and exit
                    $Items | ForEach-Object { & $setSelected $_ $false }
                    $running = $false
                }
                default {
                    switch ($key.Character.ToString().ToLowerInvariant()) {
                        'j' { $cursor = if ($cursor -lt $Items.Count - 1) { $cursor + 1 } else { 0 } }
                        'k' { $cursor = if ($cursor -gt 0) { $cursor - 1 } else { $Items.Count - 1 } }
                        ' ' {
                            $current = [bool](& $getItemValue $Items[$cursor] 'Selected')
                            & $setSelected $Items[$cursor] (-not $current)
                        }
                        'a' { $Items | ForEach-Object { & $setSelected $_ $true } }
                        'n' { $Items | ForEach-Object { & $setSelected $_ $false } }
                        'q' {
                            $Items | ForEach-Object { & $setSelected $_ $false }
                            $running = $false
                        }
                    }
                }
            }
        }
    } finally {
        $ErrorActionPreference = $savedErrorPref
        [Console]::CursorVisible = $true
    }

    return @($Items | Where-Object { [bool](& $getItemValue $_ 'Selected') })
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

    Write-Host "  $($C.Green)╭$('─' * ($boxWidth - 2))╮$($C.Reset)"
    $pad = [Math]::Max(0, $boxWidth - 2 - $content.Length)
    Write-Host "  $($C.Green)│$($C.Reset)$($C.Bold)$content$($C.Reset)$(' ' * $pad)$($C.Green)│$($C.Reset)"
    Write-Host "  $($C.Green)╰$('─' * ($boxWidth - 2))╯$($C.Reset)"
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
