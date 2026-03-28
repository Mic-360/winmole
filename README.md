# Winmole

Winmole is a Windows-first terminal maintenance application built with Go, Cobra, Bubble Tea, Lip Gloss, and the Charm ecosystem.

It provides a single keyboard-driven TUI for:
- live system status and health
- project analysis and artifact purge
- deep-clean target selection
- uninstall inventory and removal
- explicit optimization task selection
- runtime logs, settings, and help docs

The current app is centered around one unified `winmole.exe` binary. The old split PowerShell-plus-multiple-binaries flow has been replaced with one full-screen terminal application.

## Highlights

- Bubble Tea application shell with sidebar navigation, modal dialogs, and command palette
- Windows-optimized terminal layout for Windows Terminal, PowerShell, CMD, and WezTerm
- Selection-first workflows for clean, uninstall, optimize, and purge
- Project-aware artifact scanning for Node, Next.js, Python, Go, Rust, Java, Flutter, .NET, and more
- Searchable runtime logs and in-app markdown help
- Cobra CLI entrypoint with TUI-first command aliases

## Requirements

- Windows 10 or Windows 11
- PowerShell 5.1+ or PowerShell 7+
- Go 1.24+ if you want to build or run from source

## Install

### Option 1: Install locally with the provided installer

From the repository root:

```powershell
.\install.ps1
```

This installs `winmole.exe` into `%LOCALAPPDATA%\WiMo`, creates `winmole.cmd` and `wimo.cmd`, and can add the install directory to your user `PATH`.

After installation, open a new terminal and run:

```powershell
winmole
```

You can also use the legacy alias:

```powershell
wimo
```

### Option 2: Build and run the binary yourself

```powershell
go build -o .\bin\winmole.exe .
.\bin\winmole.exe
```

## Run Without Installing

### Run directly from source with Go

```powershell
go run .
```

### Run through the repository wrapper

```powershell
.\wimo.ps1
```

The wrapper will try `winmole.exe` first and fall back to `go run .` if Go is available.

### Run a locally built binary without adding it to PATH

```powershell
go build -o .\bin\winmole.exe .
.\bin\winmole.exe
```

## Usage

### Launch the main TUI

```powershell
winmole
```

### Open a specific screen directly

```powershell
winmole dashboard
winmole projects
winmole actions
winmole logs
winmole settings
winmole help
```

### Command aliases

These aliases currently route into the unified TUI:

```powershell
winmole status
winmole analyze
winmole purge
winmole clean
winmole uninstall
winmole optimize
```

## In-App Navigation

Core bindings:

- `Ctrl+P`: command palette
- `Tab`: switch between sidebar and screen content
- `Shift+Tab`: move focus back
- `↑/↓` or `j/k`: move
- `Enter`: select or drill in
- `Esc` / `Backspace`: back or close modal
- `Space`: toggle selection
- `x`: run selected workflow
- `/`: filter lists or search logs
- `?`: open help

Full keybindings live in [docs/KEYBINDINGS.md](docs/KEYBINDINGS.md).

## Workflows

### Dashboard

The dashboard replaces the old separate status window with a live system overview showing:
- CPU, memory, disk, and network activity
- health scoring
- environment details
- active alerts and quick commands

### Projects

The Projects screen combines the old analyze/purge concepts into one flow:
- detect project roots across configured scan paths
- analyze top space consumers per project
- detect build artifacts and caches
- select exactly what to purge before execution

### Actions

The Actions screen is split into three panes:
- `Clean`: select cleanup targets before removal
- `Uninstall`: review a filtered uninstallable app inventory and select apps to remove
- `Optimize`: choose which optimization tasks to run instead of applying everything automatically

### Logs

The Logs screen shows runtime events from scans and task execution, with filtering and pause/resume support.

### Settings

The Settings screen lets you edit:
- scan paths
- purge depth
- refresh interval
- winget integration
- update checks

## Build From Source

### Local build

```powershell
go build -o .\bin\winmole.exe .
```

### Cross-build Windows release binaries

```powershell
make build-windows-amd64
make build-windows-arm64
```

### GoReleaser

The repo includes `.goreleaser.yml` for Windows release packaging.

## Verification

The refactor has been verified with:

```powershell
gofmt -w main.go cmd internal pkg
go vet ./...
go test ./...
go build -o .\bin\winmole.exe .
```

## Repository Layout

```text
winmole/
├── cmd/                  Cobra entrypoints
├── internal/
│   ├── screens/          Screen renderers
│   ├── services/         Runtime, clean, uninstall, optimize, purge, config, logger
│   ├── state/            Shared DTOs and app state
│   ├── tui/              Bubble Tea app shell
│   └── ui/               Theme, layout, modal, keymap, components
├── pkg/util/             Shared formatting and fuzzy helpers
├── docs/                 Architecture, components, keybindings, developer guide
├── install.ps1           Installer for local Windows use
├── wimo.ps1              PowerShell wrapper
├── wimo.cmd              CMD wrapper / alias
├── winmole.cmd           Primary command wrapper
└── main.go               Main application entrypoint
```

## Documentation

- [docs/TUI_ARCHITECTURE.md](docs/TUI_ARCHITECTURE.md)
- [docs/UI_COMPONENTS.md](docs/UI_COMPONENTS.md)
- [docs/KEYBINDINGS.md](docs/KEYBINDINGS.md)
- [docs/DEVELOPER_GUIDE.md](docs/DEVELOPER_GUIDE.md)
- [docs/REPOSITORY_ANALYSIS.md](docs/REPOSITORY_ANALYSIS.md)

## License

[MIT](LICENSE)
