# WiMo — common.ps1
# Central module loader — dot-sources all core libraries

$script:WimoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)

# Load core modules in dependency order
. "$PSScriptRoot\base.ps1"
. "$PSScriptRoot\log.ps1"
. "$PSScriptRoot\file_ops.ps1"
. "$PSScriptRoot\ui.ps1"
