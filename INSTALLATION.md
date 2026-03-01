# Installation Guide

## Quick Install

```powershell
git clone https://github.com/mic-360/winmole.git
cd winmole
.\install.ps1
```

This will:

1. Copy WiMo scripts to `%LOCALAPPDATA%\WiMo\`
2. Create a `wimo.cmd` launcher
3. Add WiMo to your system PATH
4. Create a default config at `%APPDATA%\WiMo\config.json`

---

## Prerequisites

| Requirement    | Version          | Notes                                                            |
| -------------- | ---------------- | ---------------------------------------------------------------- |
| **Windows**    | 10 (1809+) or 11 | ANSI/VT100 terminal support required                             |
| **PowerShell** | 5.1+             | Ships with Windows. PowerShell 7+ also supported                 |
| **Go**         | 1.24+            | _Optional_ — only needed for `analyze` and `status` TUI binaries |
| **Git**        | Any              | _Optional_ — for cloning the repo                                |

### Verify Prerequisites

```powershell
# Check PowerShell version
$PSVersionTable.PSVersion

# Check Go (optional)
go version

# Check Windows version
[System.Environment]::OSVersion.Version
```

---

## Install Options

### Option 1: Standard Install

```powershell
.\install.ps1
```

Installs to `%LOCALAPPDATA%\WiMo\` and adds to PATH. After installation, open a **new terminal** and run:

```powershell
wimo
```

### Option 2: Custom Install Directory

```powershell
.\install.ps1 -InstallDir "D:\Tools\WiMo"
```

### Option 3: Install with Go TUI Binaries

If you have Go 1.24+ installed, build the `analyze` and `status` interactive TUI tools:

```powershell
.\install.ps1 -BuildFromSource
```

This compiles:

- `analyze.exe` — visual disk space explorer
- `status.exe` — live system health dashboard

### Option 4: Install with Start Menu Shortcut

```powershell
.\install.ps1 -CreateShortcut
```

Creates a Start Menu shortcut for launching WiMo.

### Option 5: Full Install (All Options)

```powershell
.\install.ps1 -BuildFromSource -CreateShortcut
```

---

## Build Go Binaries Manually

If you skipped `-BuildFromSource` during install, you can build the Go TUI binaries later:

```powershell
# From the winmole directory
make build

# Or manually:
go build -ldflags="-s -w" -o bin/analyze.exe ./cmd/analyze
go build -ldflags="-s -w" -o bin/status.exe ./cmd/status
```

Then copy the resulting `.exe` files to your WiMo install directory:

```powershell
Copy-Item bin\analyze.exe "$env:LOCALAPPDATA\WiMo\bin\"
Copy-Item bin\status.exe  "$env:LOCALAPPDATA\WiMo\bin\"
```

---

## Install via Winget (Future)

WiMo includes a winget manifest. Once published:

```powershell
winget install mic-360.WiMo
```

---

## Run Without Installing

You can run WiMo directly from the cloned repo without installing:

```powershell
cd winmole
.\wimo.ps1
.\wimo.ps1 clean --dry-run
```

---

## Verify Installation

```powershell
# Check WiMo is in PATH
wimo --version

# Launch interactive menu
wimo

# Test a command with dry-run
wimo clean --dry-run
```

---

## File Locations

| What         | Path                           |
| ------------ | ------------------------------ |
| Scripts      | `%LOCALAPPDATA%\WiMo\`         |
| Config       | `%APPDATA%\WiMo\config.json`   |
| Go binaries  | `%LOCALAPPDATA%\WiMo\bin\`     |
| CMD launcher | `%LOCALAPPDATA%\WiMo\wimo.cmd` |

---

## Updating

```powershell
wimo update
```

Or manually pull the latest and re-run the installer:

```powershell
cd winmole
git pull
.\install.ps1
```

---

## Uninstalling

### Via WiMo

```powershell
wimo remove
```

### Manual Uninstall

1. Delete the install directory:

   ```powershell
   Remove-Item -Recurse -Force "$env:LOCALAPPDATA\WiMo"
   ```

2. Delete the config directory:

   ```powershell
   Remove-Item -Recurse -Force "$env:APPDATA\WiMo"
   ```

3. Remove WiMo from your PATH:
   ```powershell
   $path = [Environment]::GetEnvironmentVariable("Path", "User")
   $path = ($path -split ";" | Where-Object { $_ -notlike "*WiMo*" }) -join ";"
   [Environment]::SetEnvironmentVariable("Path", $path, "User")
   ```

---

## Troubleshooting

### "wimo is not recognized"

Your terminal needs to reload PATH. Close and reopen your terminal, or run:

```powershell
$env:Path = [Environment]::GetEnvironmentVariable("Path", "User") + ";" + [Environment]::GetEnvironmentVariable("Path", "Machine")
```

### Garbled or missing colors

Ensure your terminal supports ANSI escape sequences. Windows Terminal, PowerShell 7, and VS Code terminal all work. Legacy `cmd.exe` may have issues on older Windows 10 builds.

### "analyze" or "status" not working

These commands require Go TUI binaries. Build them with:

```powershell
# From the winmole source directory
make build
```

Or reinstall with:

```powershell
.\install.ps1 -BuildFromSource
```

### PowerShell execution policy error

If you get a script execution policy error:

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```
