$ErrorActionPreference = "Stop"
$version = "0.74.0-multicolumn.3"
$fzf_base = Split-Path -Parent $MyInvocation.MyCommand.Definition
$binary = Join-Path $fzf_base "bin\fzf.exe"

if (Test-Path $binary) {
  $help = & $binary --help 2>&1
  if ($LASTEXITCODE -eq 0 -and ($help | Select-String -SimpleMatch "--grid=COLS")) {
    Write-Host "Keeping the existing multicolumn binary."
    exit 0
  }
}

New-Item -ItemType Directory -Force (Join-Path $fzf_base "bin") | Out-Null
$architecture = $env:PROCESSOR_ARCHITECTURE
if ($env:PROCESSOR_ARCHITEW6432) { $architecture = $env:PROCESSOR_ARCHITEW6432 }
$arch = switch ($architecture) {
  "AMD64" { "amd64" }
  "ARM64" { "arm64" }
  "ARM" { "armv7" }
  default { $null }
}
if ($arch) {
  $temporary = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
  New-Item -ItemType Directory $temporary | Out-Null
  try {
    $archive = Join-Path $temporary "fzf.zip"
    $url = "https://github.com/LPFchan/fzf-multicolumn/releases/download/v$version/fzf-multicolumn-$version-windows_$arch.zip"
    Invoke-WebRequest -Uri $url -OutFile $archive -UseBasicParsing
    Expand-Archive -Path $archive -DestinationPath $temporary
    $downloaded = Join-Path $temporary "fzf-multicolumn.exe"
    $help = & $downloaded --help 2>&1
    if ($LASTEXITCODE -ne 0 -or -not ($help | Select-String -SimpleMatch "--grid=COLS")) {
      throw "Downloaded executable does not support multicolumn grid mode."
    }
    Copy-Item -Force $downloaded $binary
    Write-Host "Installed the multicolumn release binary."
    exit 0
  } catch {
    Write-Host "Prebuilt fork binary unavailable; trying a source build."
  } finally {
    Remove-Item -Recurse -Force $temporary
  }
}

# Build this checkout when a fork release asset is unavailable.
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw "Go is required to build fzf-multicolumn on Windows."
}
Push-Location $fzf_base
try {
  & go build -ldflags "-s -w -X main.version=$version -X main.revision=go-build" -o $binary .
  if ($LASTEXITCODE -ne 0) { throw "Failed to build fzf-multicolumn." }
} finally {
  Pop-Location
}
Write-Host 'For more information, see: https://github.com/LPFchan/fzf-multicolumn'
