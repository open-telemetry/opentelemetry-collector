<#
.SYNOPSIS
    This script ensures that the ports required by the default configuration of the collector are available.
.DESCRIPTION
    Certain runs on GitHub Actions sometimes have ports required by the default configuration reserved by other
    applications via the WinNAT service.
#>

#Requires -RunAsAdministrator

netsh interface ip show excludedportrange protocol=tcp

Stop-Service winnat

# Only port in the dynamic range that is being, from time to time, reserved by other applications.
netsh interface ip add excludedportrange protocol=tcp startport=55679 numberofports=1

Start-Service winnat

netsh interface ip show excludedportrange protocol=tcp
