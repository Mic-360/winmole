<p align="center">
	<img src="winmole-logo.png" alt="WiMo Logo" width="256">
  <h1 align="center">🐭 WiMo</h1>
  <p align="center"><strong>Windows System Optimizer — Mole for Windows</strong></p>
  <p align="center">
    <a href="#features">Features</a> ·
    <a href="INSTALLATION.md">Install</a> ·
    <a href="#usage">Usage</a> ·
    <a href="CONTRIBUTING.md">Contribute</a>
  </p>
  <p align="center">
    <img src="https://img.shields.io/badge/platform-Windows%2010%2B-blue?style=flat-square" alt="Platform">
    <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License">
    <img src="https://img.shields.io/badge/powershell-5.1%2B-5391FE?style=flat-square&logo=powershell&logoColor=white" alt="PowerShell">
    <img src="https://img.shields.io/badge/go-1.24-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/version-1.0.0-orange?style=flat-square" alt="Version">
  </p>
</p>

---

WiMo is an all-in-one **system maintenance CLI toolkit for Windows**. Think **CleanMyMac + AppCleaner + DaisyDisk + iStat Menus** — but for Windows, free, and open source.

Built with PowerShell and Go for a fast, colorful terminal experience with interactive menus, real-time dashboards, and safe file operations.

```
      ██████              ██████
    ██▓▓▓▓▓▓██          ██▓▓▓▓▓▓██
    ██▓▓▓▓██████████████████▓▓▓▓██
      ████░░░░░░░░░░░░░░░░░░████
      ██░░░░░░ ◉ ░░░░ ◉ ░░░░░░██
      ██░░░░░░░░░░░░░░░░░░░░░░██
        ██░░░░░░░░░░░░░░░░░░██
          ████░░░░░░░░░░████
            ██████████████
```

## Features

| Command          | What it does                                                                             |
| ---------------- | ---------------------------------------------------------------------------------------- |
| `wimo clean`     | Deep system cleanup — temp files, browser caches, Windows Update leftovers, system logs  |
| `wimo uninstall` | Interactive app uninstaller with registry + winget integration and leftover cleanup      |
| `wimo optimize`  | System optimization — flush DNS, clear icon/thumbnail caches, trim SSD, restart services |
| `wimo analyze`   | Visual disk space explorer — Go TUI with real-time directory scanning (bubbletea)        |
| `wimo status`    | Live system health dashboard — CPU, RAM, disk, network with auto-refresh (bubbletea)     |
| `wimo purge`     | Clean project build artifacts — node_modules, .next, target, **pycache**, Flutter, etc.  |

### Highlights

- **Interactive TUI** — vim-style navigation (`j`/`k`), checkboxes, progress bars, box-drawing menus
- **4-layer safe delete** — protected paths, user data patterns, recycle bin fallback, dry-run preview
- **28+ cleanup targets** — user-level caches, browser data, package managers, dev tools
- **Smart uninstaller** — merges apps from Windows Registry, winget, and local programs into one list
- **Admin-aware** — badges admin-required tasks, skips them gracefully when running as standard user
- **ANSI color output** — 256-color palette with sage green branding, works on Windows 10+ terminals
- **Winget ready** — includes a winget manifest for package distribution

## Usage

Launch the interactive menu:

```powershell
wimo
```

Or run commands directly:

```powershell
# Deep system cleanup
wimo clean

# Preview what would be deleted (no files removed)
wimo clean --dry-run

# Interactive app uninstaller
wimo uninstall

# System optimization
wimo optimize

# Visual disk analyzer (Go TUI)
wimo analyze

# Live system dashboard
wimo status

# Clean build artifacts from project directories
wimo purge
wimo purge --paths "C:\Projects"
wimo purge --depth 5

# Self-update
wimo update

# Uninstall WiMo
wimo remove
```

### Flags

| Flag              | Scope   | Description                                    |
| ----------------- | ------- | ---------------------------------------------- |
| `--help`, `-h`    | Global  | Show help message                              |
| `--version`, `-v` | Global  | Show version                                   |
| `--debug`         | Global  | Enable verbose debug logging                   |
| `--dry-run`       | `clean` | Preview cleanup plan without deleting anything |
| `--whitelist`     | `clean` | Manage protected paths                         |
| `--paths`         | `purge` | Specify custom scan directories                |
| `--depth N`       | `purge` | Limit recursion depth (default: 8)             |

## Quick Install

```powershell
# Clone and install
git clone https://github.com/mic-360/winmole.git
cd winmole
.\install.ps1
```

For detailed installation options, build-from-source instructions, and troubleshooting, see **[INSTALLATION.md](INSTALLATION.md)**.

## Requirements

- **Windows 10** version 1809+ (for ANSI/VT100 terminal support)
- **PowerShell 5.1+** (ships with Windows) or PowerShell 7+
- **Go 1.24+** (optional — only needed to build the `analyze` and `status` TUI binaries)

## Project Structure

```
winmole/
├── wimo.ps1              # Entry point — arg parser & command dispatcher
├── wimo.cmd              # CMD wrapper for PATH-based invocation
├── install.ps1           # Installer (PATH, config, optional Go build)
├── Makefile              # Go binary build targets
├── go.mod                # Go module definition
├── bin/
│   ├── clean.ps1         # Deep system cleanup (28+ targets)
│   ├── uninstall.ps1     # Interactive app uninstaller
│   ├── optimize.ps1      # System optimization (11 tasks)
│   ├── analyze.ps1       # Go TUI wrapper — disk analyzer
│   ├── status.ps1        # Go TUI wrapper — health dashboard
│   └── purge.ps1         # Build artifact cleaner
├── lib/core/
│   ├── common.ps1        # Module loader (dot-sources all core libs)
│   ├── base.ps1          # Constants, colors, ASCII logo, config
│   ├── log.ps1           # Debug logging
│   ├── file_ops.ps1      # Safe file operations (4-layer protection)
│   └── ui.ps1            # TUI primitives (menus, checkboxes, progress)
├── cmd/
│   ├── analyze/main.go   # Disk analyzer TUI (bubbletea + lipgloss)
│   └── status/main.go    # System health dashboard (bubbletea + gopsutil)
└── winget-manifest/
    └── mic-360.WiMo.yaml # Winget package manifest
```

## How It Works

### Safe File Operations

WiMo uses a **4-layer safety system** before deleting any file:

1. **Protected path check** — system-critical directories (`C:\Windows`, `C:\Program Files`, etc.) are never touched
2. **User data pattern check** — Documents, Desktop, Downloads, Pictures, etc. are always preserved
3. **Recycle bin fallback** — files go to Recycle Bin when possible instead of permanent deletion
4. **Dry-run preview** — `--dry-run` shows exactly what would be removed before committing

### Go TUI Components

The `analyze` and `status` commands use Go binaries built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for rich terminal UIs:

- **Analyze** — concurrent directory scanning with goroutines, live-updating size display, alt-screen mode
- **Status** — 2-second refresh tick, CPU/memory/disk/network monitoring via [gopsutil](https://github.com/shirou/gopsutil), health score calculation

These binaries are optional. If not built, the PowerShell wrappers will show a helpful message with build instructions.

## Screenshots

### Interactive Menu

```
┌──────────────────────────────────────────────────────┐
│  🐭 WiMo  · Windows System Optimizer                 │
├──────────────────────────────────────────────────────┤
│  ▶  🧹  Clean System      Free disk space           │
│     🗑️  Uninstall Apps     Remove + leftovers       │
│     ⚡  Optimize System    Speed & refresh           │
│     📁  Analyze Disk       Visual explorer          │
│     📊  Live Status        Health dashboard         │
│     🔥  Purge Projects    Dev artifact junk         │
├──────────────────────────────────────────────────────┤
│  ↑↓ / jk navigate  ·  Enter select  ·  q quit        │
└──────────────────────────────────────────────────────┘
```

### Clean (Dry Run)

```
  🧹  Scanning system caches...

  ✓ Windows Temp ············· 3.0 GB
  ✓ User Temp ················ 512.4 MB
  ✓ Chrome Cache ············· 359.0 MB
  ✓ Edge Cache ··············· 377.1 MB
  ⊘ Firefox Cache ··········· not found

  ┌──────────────────────────────────────┐
  │  Would free: 7.0 GB                 │
  │  Free now: 653.1 GB                 │
  └──────────────────────────────────────┘
```

## License

[MIT](LICENSE) — free for personal and commercial use.

## Contributing

Contributions are welcome! See **[CONTRIBUTING.md](CONTRIBUTING.md)** for guidelines.

## Acknowledgments

- Inspired by [Mole](https://github.com/nicehash/mole) (macOS system toolkit by tw93)
- Go TUI powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea) & [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- System metrics via [gopsutil](https://github.com/shirou/gopsutil)
