# gospeed installer — Windows (PowerShell)
# Usage:  irm https://gospeed.goozt.org/install.ps1 | iex
#         $env:INSTALL_SERVER=1; irm https://gospeed.goozt.org/install.ps1 | iex
$ErrorActionPreference = "Stop"

$Repo = "goozt/gospeed"
$ProjectName = "gospeed"
$GoPath = if ($env:GOPATH) { $env:GOPATH } elseif (Get-Command go -ErrorAction SilentlyContinue) { (go env GOPATH) } else { Join-Path $env:USERPROFILE "go" }
$InstallDir = Join-Path $GoPath "bin"

$Binaries = @("gospeed")
if ($env:INSTALL_SERVER -eq "1") {
    $Binaries += "gospeed-server"
}

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

# Download (single archive contains all binaries)
$Archive = "${ProjectName}_${Version}_windows_${Arch}.zip"
$Url = "https://github.com/$Repo/releases/download/v$Version/$Archive"
$ChecksumUrl = "https://github.com/$Repo/releases/download/v$Version/checksums.txt"
$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "gospeed-install"

if (Test-Path $TmpDir) { Remove-Item $TmpDir -Recurse -Force }
New-Item -ItemType Directory -Path $TmpDir | Out-Null

$ArchivePath = Join-Path $TmpDir $Archive
$ChecksumPath = Join-Path $TmpDir "checksums.txt"

Write-Host "Downloading $Archive..."
Invoke-WebRequest -Uri $Url -OutFile $ArchivePath
Invoke-WebRequest -Uri $ChecksumUrl -OutFile $ChecksumPath

# Verify checksum
$ExpectedLine = (Get-Content $ChecksumPath | Where-Object { $_ -match "\s\*?$([regex]::Escape($Archive))\s*$" })
if ($ExpectedLine) {
    $Expected = ($ExpectedLine -split '\s+')[0]
    $Actual = (Get-FileHash -Algorithm SHA256 -Path $ArchivePath).Hash.ToLower()
    if ($Actual -ne $Expected) {
        Write-Error "Checksum mismatch for $Archive : got $Actual, want $Expected"
        exit 1
    }
} else {
    Write-Warning "$Archive not found in checksums.txt — skipping verification"
}

# Extract
Expand-Archive -Path $ArchivePath -DestinationPath $TmpDir -Force

# Install
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir | Out-Null
}
foreach ($bin in $Binaries) {
    Copy-Item (Join-Path $TmpDir "$bin.exe") -Destination $InstallDir -Force
    Write-Host "Installed $bin v$Version to $InstallDir\$bin.exe"
}

# Add to PATH if not already there
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "Added $InstallDir to PATH (restart your terminal to use)"
}

# Cleanup
Remove-Item $TmpDir -Recurse -Force

Write-Host ""
Write-Host "Run 'gospeed -v' to verify (you may need to restart your terminal)"
