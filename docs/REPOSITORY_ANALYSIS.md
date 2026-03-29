# Repository Analysis

## Repository Overview

### Purpose
The original repository is a Windows maintenance toolkit that mixes system cleanup, uninstall, optimization, disk analysis, status telemetry, and project artifact purge features.

### Current Architecture (Before Refactor)
- PowerShell dispatcher in `wimo.ps1`
- PowerShell TUI primitives in `lib/core/ui.ps1`
- Operational scripts in `bin/*.ps1`
- Two standalone Go binaries for `analyze` and `status`
- No unified runtime state or shared application shell

### Current CLI Commands (Before Refactor)
- `clean`
- `uninstall`
- `optimize`
- `analyze`
- `status`
- `purge`
- `update`
- `remove`

### Issues Detected
- The entrypoint was not a single application; it was a launcher for unrelated scripts and binaries.
- Selecting TUI functions could drop into separate processes and exit back to a blank terminal.
- Uninstall inventory merged sources too aggressively and produced unrealistic app counts.
- Cleanup and optimization flows were execution-first instead of selection-first.
- Analyze and status were isolated binaries with no shared design system or routing.
- Build/install flow still targeted the old mixed PowerShell architecture.

### Refactoring Opportunities
- Consolidate all UX into one Bubble Tea app shell.
- Move system operations into Go services with explicit state models.
- Use one design system for navigation, cards, modal overlays, logs, and command palette.
- Treat `winget` as enrichment for uninstallable apps rather than as the base source of truth.
- Add project-aware artifact scanning for modern polyglot repos.

### Missing Features (Before Refactor)
- Unified sidebar navigation
- Command palette
- Searchable logs viewer
- Reusable modal framework
- Settings editor
- Screen-specific state persistence
- Windows build/release pipeline for the new binary
