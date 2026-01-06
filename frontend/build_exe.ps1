param(
  [switch]$Clean
)

$ErrorActionPreference = 'Stop'

$here = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $here

if ($Clean) {
  if (Test-Path .\build) { Remove-Item -Recurse -Force .\build }
  if (Test-Path .\dist) { Remove-Item -Recurse -Force .\dist }
}

Write-Host "Installing deps..."
python -m pip install --upgrade pip
python -m pip install -r .\requirements.txt
python -m pip install pyinstaller
python -m pip install pillow

Write-Host "Generating icon..."
python .\tools\make_ico.py

Write-Host "Building exe..."
python -m PyInstaller --noconfirm --clean .\echo_trace_client.spec

Write-Host "Done: dist\\Echo_Trace_Client.exe"
