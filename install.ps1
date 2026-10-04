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

# The fork publishes macOS/Linux assets; build Windows from this checkout.
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw "Go is required to build fzf-multicolumn on Windows."
}
New-Item -ItemType Directory -Force (Join-Path $fzf_base "bin") | Out-Null
Push-Location $fzf_base
try {
  & go build -ldflags "-s -w -X main.version=$version -X main.revision=go-build" -o $binary .
  if ($LASTEXITCODE -ne 0) { throw "Failed to build fzf-multicolumn." }
} finally {
  Pop-Location
}
Write-Host 'For more information, see: https://github.com/LPFchan/fzf-multicolumn'
