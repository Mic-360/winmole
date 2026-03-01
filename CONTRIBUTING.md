# Contributing to WiMo

Thank you for your interest in contributing to WiMo! This guide will help you get started.

## Getting Started

1. **Fork** the repository on GitHub
2. **Clone** your fork:
   ```powershell
   git clone https://github.com/YOUR-USERNAME/winmole.git
   cd winmole
   ```
3. **Create a branch** for your change:
   ```powershell
   git checkout -b feature/your-feature-name
   ```

## Development Setup

### Prerequisites

- Windows 10+ with PowerShell 5.1+
- Go 1.24+ (for TUI components)
- A terminal with ANSI color support (Windows Terminal recommended)

### Running from Source

```powershell
# Run directly without installing
.\wimo.ps1

# Test a specific command
.\wimo.ps1 clean --dry-run

# Build Go TUI binaries
make build

# Run with debug logging
.\wimo.ps1 --debug clean
```

## Project Structure

Understanding where things live:

```
wimo.ps1              Entry point — parses args, dispatches commands
lib/core/
├── common.ps1        Module loader (dot-sources all core libs in order)
├── base.ps1          Constants, ANSI color palette, ASCII logo, config
├── log.ps1           Debug logging (Write-WimoLog)
├── file_ops.ps1      Safe file operations with 4-layer protection
└── ui.ps1            TUI primitives (menus, checkboxes, progress bars)
bin/
├── clean.ps1         System cleanup — each target has scan + delete logic
├── uninstall.ps1     App uninstaller — registry + winget enumeration
├── optimize.ps1      System optimization tasks
├── analyze.ps1       Wrapper that launches the Go analyze binary
├── status.ps1        Wrapper that launches the Go status binary
└── purge.ps1         Build artifact scanner and cleaner
cmd/
├── analyze/main.go   Disk analyzer TUI (bubbletea)
└── status/main.go    System dashboard TUI (bubbletea + gopsutil)
```

### Key Conventions

- **Colors** are defined in `$script:C` in `base.ps1` using ANSI 256-color codes (sage green palette)
- **Safe deletion** always goes through `Remove-SafePath` or `Remove-SafeGlob` in `file_ops.ps1`
- **Fast I/O** uses `System.IO.Directory.EnumerateFiles/EnumerateDirectories` instead of `Get-ChildItem` for scanning
- **Parallel operations** use runspace pools in `clean.ps1` and `purge.ps1` (note: runspaces are isolated and cannot access session cmdlets)
- **UI elements** use box-drawing characters (`╭─╮│╰─╯`) and ANSI escape sequences
- **Interactive menu** is a persistent split-pane loop — menu returns after every command
- **Go TUI** components use the Bubble Tea architecture (Model → Update → View) with Lip Gloss styling

## How to Contribute

### Reporting Bugs

Open an issue with:

- What you expected vs. what happened
- Steps to reproduce
- Your Windows version and PowerShell version (`$PSVersionTable`)
- Terminal app (Windows Terminal, VS Code, cmd.exe, etc.)

### Adding a Cleanup Target

Cleanup targets live in `bin/clean.ps1`. Each target follows this pattern:

```powershell
# 1. Check if the path exists
$path = "$env:LOCALAPPDATA\SomeApp\Cache"
if (Test-Path $path) {
    # 2. Show scan result with size
    $size = Get-FolderSize $path
    Show-ScanItem "SomeApp Cache" (Format-FileSize $size) $true

    # 3. Delete safely (only if not dry-run)
    if (-not $DryRun) {
        Remove-SafePath $path
    }
} else {
    Show-ScanItem "SomeApp Cache" "not found" $false
}
```

### Adding an Optimize Task

Optimization tasks live in `bin/optimize.ps1`. Wrap admin-required tasks with an admin check:

```powershell
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole("Administrator")
if ($isAdmin) {
    # Admin-only operation
}
```

### Modifying Go TUI Components

The Go binaries in `cmd/` use [Bubble Tea](https://github.com/charmbracelet/bubbletea):

```powershell
# Build after changes
go build -ldflags="-s -w" -o bin/analyze-go.exe ./cmd/analyze
go build -ldflags="-s -w" -o bin/status-go.exe ./cmd/status

# Or use make
make build
```

## Code Style

### PowerShell

- Use `PascalCase` for function names (e.g., `Show-InteractiveMenu`)
- Use `$camelCase` for local variables
- Use `$script:UPPER_CASE` for module-level constants
- Always use `Remove-SafePath` / `Remove-SafeGlob` instead of `Remove-Item` directly
- Add `Show-ScanItem` output for any new scan targets so users see progress

### Go

- Follow standard Go formatting (`gofmt`)
- Use the Bubble Tea pattern: `Init()`, `Update()`, `View()`
- Use `lipgloss` for styling, not raw ANSI codes

## Pull Request Process

1. **Test your changes** — run the affected command and verify output
2. **Test with `--dry-run`** where applicable — never submit untested deletion logic
3. **Keep PRs focused** — one feature or fix per PR
4. **Update documentation** if you add new commands or flags
5. **Describe your change** in the PR description

### PR Checklist

- [ ] Tested on Windows 10 or 11
- [ ] No hardcoded absolute paths (use `$env:` variables)
- [ ] Destructive operations use `Remove-SafePath` / `Remove-SafeGlob`
- [ ] New cleanup targets have `--dry-run` support
- [ ] Admin-required operations are guarded with an admin check

## Questions?

Open an issue or start a discussion on GitHub. We're happy to help!
