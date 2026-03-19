# gospeed-server installer — Windows (PowerShell)
# Usage: irm https://gospeed.goozt.org/install-server.ps1 | iex
$ErrorActionPreference = "Stop"

$Repo = "goozt/gospeed"
$GoPath = if ($env:GOPATH) { $env:GOPATH } elseif (Get-Command go -ErrorAction SilentlyContinue) { (go env GOPATH) } else { Join-Path $env:USERPROFILE "go" }
$InstallDir = Join-Path $GoPath "bin"

# Detect architecture
$Arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
    "X64"   { "amd64" }
    "Arm64" { "arm64" }
    "X86"   { "386" }
    default { Write-Error "Unsupported architecture: $_"; exit 1 }
}

# Get latest version
Write-Host "Fetching latest version..."
$Release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
$Version = $Release.tag_name -replace '^v', ''
Write-Host "Latest version: v$Version"

# Download
$Archive = "gospeed-server_${Version}_windows_${Arch}.zip"
$Url = "https://github.com/$Repo/releases/download/v$Version/$Archive"
$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "gospeed-server-install"

if (Test-Path $TmpDir) { Remove-Item $TmpDir -Recurse -Force }
New-Item -ItemType Directory -Path $TmpDir | Out-Null

Write-Host "Downloading $Archive..."
Invoke-WebRequest -Uri $Url -OutFile (Join-Path $TmpDir $Archive)

# Extract
Expand-Archive -Path (Join-Path $TmpDir $Archive) -DestinationPath $TmpDir -Force

# Install
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir | Out-Null
}
Copy-Item (Join-Path $TmpDir "gospeed-server.exe") -Destination $InstallDir -Force

# Add to PATH if not already there
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "Added $InstallDir to PATH (restart your terminal to use)"
}

# Cleanup
Remove-Item $TmpDir -Recurse -Force

Write-Host "Installed gospeed-server v$Version to $InstallDir\gospeed-server.exe"
Write-Host ""
Write-Host "Run 'gospeed-server -v' to verify (you may need to restart your terminal)"
