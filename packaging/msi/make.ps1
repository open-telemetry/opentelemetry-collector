# Copyright The OpenTelemetry Authors
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

<#
.SYNOPSIS
    Makefile like build commands for the Collector on Windows.
    
    Usage:   .\make.ps1 <Command> [-<Param> <Value> ...]
    Example: .\make.ps1 New-MSI -Config "./my-config.yaml"
.PARAMETER Target
    Build target to run (Install-Tools, New-MSI)
#>
Param(
    [Parameter(Mandatory=$true, ValueFromRemainingArguments=$true)][string]$Target
)

$ErrorActionPreference = "Stop"

function Install-Tools {
    choco install wixtoolset -y
    setx /m PATH "%PATH%;C:\Program Files (x86)\WiX Toolset v3.11\bin"
    refreshenv
}

function New-MSI(
    [string]$Version="0.0.1",
    [string]$Config="./examples/otel-local-config.yaml"
) {
    candle -arch x64 -dVersion="$Version" -dConfig="$Config" packaging/msi/opentelemetry-collector.wxs
    light opentelemetry-collector.wixobj
    Move-Item -Force opentelemetry-collector.msi bin/opentelemetry-collector.msi
}

function Confirm-MSI {
    # ensure system32 is in Path so we can use executables like msiexec & sc
    $env:Path += ";C:\Windows\System32"

    # install msi, validate service is installed & running
    Start-Process -Wait msiexec "/i `"$pwd\bin\opentelemetry-collector.msi`" /qn"
    sc.exe query state=all | findstr "otelcol" | Out-Null
    if ($LASTEXITCODE -ne 0) { Throw "otelcol service failed to install" }

    # stop service
    Stop-Service otelcol

    # start service
    Start-Service otelcol

    # uninstall msi, validate service is uninstalled
    Start-Process -Wait msiexec "/x `"$pwd\bin\opentelemetry-collector.msi`" /qn"
    sc.exe query state=all | findstr "otelcol" | Out-Null
    if ($LASTEXITCODE -ne 1) { Throw "otelcol service failed to uninstall" }
}

$sb = [scriptblock]::create("$Target")
Invoke-Command -ScriptBlock $sb
