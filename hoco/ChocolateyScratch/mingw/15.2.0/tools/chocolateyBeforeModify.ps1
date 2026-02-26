
# Copyright © 2018 Chocolatey Software, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# For internal use by Install-ChocolateyPath and Uninstall-ChocolateyPath.

function Parse-EnvPathList([string] $rawPathVariableValue) {
  # Using regex (for performance) which correctly splits at each semicolon unless the semicolon is inside double quotes.
  # Unlike semicolons, quotes are not allowed inside paths so there is thankfully no need to unescape them.
  # (Verified using Windows 10’s environment variable editor.)
  # Blank path entries are preserved, such as those caused by a trailing semicolon.
  # This enables reserializing without gratuitous reformatting.
  $paths = $rawPathVariableValue -split '(?<=\G(?:[^;"]|"[^"]*")*);'

  # Remove quotes from each path if they are present
  for ($i = 0; $i -lt $paths.Length; $i++) {
    $path = $paths[$i]
    if ($path.Length -ge 2 -and $path.StartsWith('"', [StringComparison]::Ordinal) -and $path.EndsWith('"', [StringComparison]::Ordinal)) {
      $paths[$i] = $path.Substring(1, $path.Length - 2)
    }
  }

  return $paths
}

function Format-EnvPathList([string[]] $paths) {
  # Don’t mutate the original (externally visible if the argument is not type-coerced),
  # but don’t clone if mutation is unnecessary.
  $createdDefensiveCopy = $false

  # Add quotes to each path if necessary
  for ($i = 0; $i -lt $paths.Length; $i++) {
    $path = $paths[$i]
    if ($path -ne $null -and $path.Contains(';')) {
      if (-not $createdDefensiveCopy) {
        $createdDefensiveCopy = $true
        $paths = $paths.Clone()
      }
      $paths[$i] = '"' + $path + '"'
    }
  }

  return $paths -join ';'
}

function IndexOf-EnvPath([System.Collections.Generic.List[string]] $list, [string] $value) {
  $list.FindIndex({
    $value.Equals($args[0], [StringComparison]::OrdinalIgnoreCase)
  })
}

# Copyright © 2018 Chocolatey Software, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

function Uninstall-ChocolateyPath {
<#
.SYNOPSIS
**NOTE:** Administrative Access Required when `-PathType 'Machine'.`
This puts a directory to the PATH environment variable.
.DESCRIPTION
Ensures that the given path is not present in the given type of PATH
environment variable as well as in the current session.
.NOTES
This command will assert UAC/Admin privileges on the machine if
`-PathType 'Machine'`.
This is used when the application/tool is not being linked by Chocolatey
(not in the lib folder).
.PARAMETER PathToUninstall
The exact path to remove from the environment PATH.
.PARAMETER PathType
Which PATH to add it to. If specifying `Machine`, this requires admin
privileges to run correctly.
.PARAMETER IgnoredArguments
Allows splatting with arguments that do not apply. Do not use directly.
.EXAMPLE
Uninstall-ChocolateyPath -PathToUninstall "$($env:SystemDrive)\tools\gittfs"
.EXAMPLE
Uninstall-ChocolateyPath "$($env:SystemDrive)\Program Files\MySQL\MySQL Server 5.5\bin" -PathType 'Machine'
.LINK
Install-ChocolateyPath
.LINK
Get-EnvironmentVariable
.LINK
Set-EnvironmentVariable
.LINK
Get-ToolsLocation
#>
param(
  [parameter(Mandatory=$true, Position=0)][string] $pathToUninstall,
  [parameter(Mandatory=$false, Position=1)][System.EnvironmentVariableTarget] $pathType = [System.EnvironmentVariableTarget]::User,
  [parameter(ValueFromRemainingArguments = $true)][Object[]] $ignoredArguments
)

  Write-FunctionCallLogMessage -Invocation $MyInvocation -Parameters $PSBoundParameters

  $paths = Parse-EnvPathList (Get-EnvironmentVariable -Name 'PATH' -Scope $pathType -PreserveVariables)
  $removeIndex = (IndexOf-EnvPath $paths $pathToUninstall)
  if ($removeIndex -ge 0) {
    Write-Host "Found $pathToUninstall in PATH environment variable. Removing..."

    if ($pathType -eq [EnvironmentVariableTarget]::Machine -and -not (Test-ProcessAdminRights)) {
      $psArgs = "Uninstall-ChocolateyPath -pathToUninstall `'$pathToUninstall`' -pathType `'$pathType`'"
      Start-ChocolateyProcessAsAdmin "$psArgs"
    } else {
      $paths = [System.Collections.ArrayList] $paths
      $paths.RemoveAt($removeIndex)
      Set-EnvironmentVariable -Name 'PATH' -Value $(Format-EnvPathList $paths) -Scope $pathType
    }
  }

  # Make change immediately available
  $paths = Parse-EnvPathList $env:PATH
  $removeIndex = (IndexOf-EnvPath $paths $pathToUninstall)
  if ($removeIndex -ge 0) {
    $paths = [System.Collections.ArrayList] $paths
    $paths.RemoveAt($removeIndex)
    $env:Path = Format-EnvPathList $paths
  }
}
If (-Not (Test-Path $env:ProgramData)) {
  $ProgramData = "C:\ProgramData"
} else {
  $ProgramData = $env:ProgramData
}

Write-Output "Using ProgramData: $ProgramData"

$install = Join-Path "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)" "install"
$("mingw32", "mingw64") | ForEach {
  Uninstall-ChocolateyPath (Join-Path $install (Join-Path $_ "bin"))
  Uninstall-ChocolateyPath (Join-Path $install (Join-Path $_ "bin")) -PathType 'Machine'
}

$install = "$ProgramData\mingw64"
$("mingw32", "mingw64") | ForEach {
  Uninstall-ChocolateyPath (Join-Path $install (Join-Path $_ "bin"))
  Uninstall-ChocolateyPath (Join-Path $install (Join-Path $_ "bin")) -PathType 'Machine'
}

Remove-Item -Force -Recurse $install
