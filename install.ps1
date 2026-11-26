# Installs or updates rubio from the latest GitHub release (Windows, 64-bit).
#
#   irm https://raw.githubusercontent.com/dicapower/rubio-ia/main/install.ps1 | iex
#
# Run it again to upgrade. It verifies the SHA-256 checksum, needs no admin
# rights and adds the install folder to your user PATH. Settings:
#   $env:RUBIO_VERSION      tag to install, e.g. v0.1.0 (default: latest release)
#   $env:RUBIO_INSTALL_DIR  folder for the binary (default: %LOCALAPPDATA%\Programs\rubio)
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$repo = 'dicapower/rubio-ia'
$name = 'rubio'

if (-not [Environment]::Is64BitOperatingSystem) { throw "$name needs 64-bit Windows" }
$arch = $env:PROCESSOR_ARCHITECTURE
if ($arch -ne 'AMD64' -and $arch -ne 'ARM64') { throw "unsupported architecture $arch" }
# ARM64 Windows runs the amd64 build through built-in emulation.

$tag = $env:RUBIO_VERSION
if (-not $tag) {
  $tag = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name
}
if ($tag -notmatch '^v\d') { throw "could not work out the latest version (got '$tag')" }

$asset = "$name-$tag-windows-amd64"
$base = "https://github.com/$repo/releases/download/$tag"
$tmp = Join-Path ([IO.Path]::GetTempPath()) ("$name-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
  Write-Host "Downloading $name $tag (windows/amd64)..."
  Invoke-WebRequest "$base/$asset.zip" -OutFile "$tmp\$asset.zip"
  Invoke-WebRequest "$base/SHA256SUMS" -OutFile "$tmp\SHA256SUMS"

  $line = Get-Content "$tmp\SHA256SUMS" | Where-Object { $_ -match "  $([regex]::Escape($asset)).zip$" } | Select-Object -First 1
  if (-not $line) { throw "$asset.zip is not listed in SHA256SUMS" }
  $want = ($line -split '\s+')[0].ToLower()
  $got = (Get-FileHash "$tmp\$asset.zip" -Algorithm SHA256).Hash.ToLower()
  if ($got -ne $want) { throw 'checksum mismatch, nothing installed' }

  Expand-Archive "$tmp\$asset.zip" -DestinationPath $tmp
  $exe = Join-Path $tmp "$asset\$name.exe"
  if (-not (Test-Path $exe)) { throw "the archive does not contain $name.exe" }

  $dir = $env:RUBIO_INSTALL_DIR
  if (-not $dir) {
    $existing = Get-Command $name -ErrorAction SilentlyContinue
    if ($existing) { $dir = Split-Path $existing.Source } else { $dir = Join-Path $env:LOCALAPPDATA "Programs\$name" }
  }
  New-Item -ItemType Directory -Force -Path $dir | Out-Null
  $dest = Join-Path $dir "$name.exe"
  try {
    Copy-Item $exe "$dest.new" -Force
    Move-Item "$dest.new" $dest -Force
  } catch {
    Remove-Item "$dest.new" -Force -ErrorAction SilentlyContinue
    throw "could not replace $dest. Close your agent (it may be using $name.exe) and run this again."
  }
} finally {
  Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host "Installed $(& $dest version) in $dir"

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (-not (($userPath -split ';') -contains $dir)) {
  [Environment]::SetEnvironmentVariable('Path', ($userPath.TrimEnd(';') + ';' + $dir).TrimStart(';'), 'User')
  $env:Path = "$env:Path;$dir"
  Write-Host "Added $dir to your PATH. Open a new terminal for it to take effect."
}
Write-Host ''
Write-Host 'Restart your agent, then run: rubio setup'
