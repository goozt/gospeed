# Release Guide

## Quick Release

```bash
# 1. Ensure code is ready
task bump-ready

# 2. Bump version (creates RELEASE_VERSION file and git tag)
task bump VERSION=1.4.0

# 3. Release to all package managers
bash scripts/release.sh
```

`scripts/release.sh` will only run if `RELEASE_VERSION` exists and the version hasn't been released yet. It cleans up the file after a successful release.

---

## Package Managers

### Auto-published by GoReleaser (on `goreleaser release`)

These are all handled automatically when `scripts/release.sh` runs GoReleaser:

| Channel | What happens | Prerequisites |
|---|---|---|
| **GitHub Releases** | Archives (.tar.gz/.zip) + checksums uploaded to GitHub release | `GITHUB_TOKEN` env var |
| **Homebrew** | Cask formula pushed to `goozt/homebrew-tap` repo | `GITHUB_TOKEN` with repo access, `goozt/homebrew-tap` repo exists |
| **Scoop** | Manifest JSON pushed to `goozt/scoop-bucket` repo | `GITHUB_TOKEN` with repo access, `goozt/scoop-bucket` repo exists |
| **Snap** | Snaps built and published to Snap Store | `snapcraft login` completed, `SNAPCRAFT_STORE_CREDENTIALS` env var |
| **Docker Hub** | `goozt/gospeed-server` image pushed with version + latest tags | `docker login` to Docker Hub |
| **deb/rpm** | `.deb` and `.rpm` packages attached to GitHub release | None (attached as release artifacts) |

### Auto-published by npm (after GoReleaser)

| Channel | What happens | Prerequisites |
|---|---|---|
| **npm** | Platform-specific + wrapper packages published to npmjs.com | `npm login` or `NPM_TOKEN` env var |
| **yarn / pnpm** | Work automatically — they use the npm registry | Same as npm |

### Already available (no publish step)

| Channel | How users install | Notes |
|---|---|---|
| **go install** | `go install github.com/goozt/gospeed/cmd/gospeed@latest` | Pulls from Go module proxy automatically when tag is pushed |
| **Shell script (Unix)** | `curl -fsSL https://goozt.github.io/gospeed/install.sh \| bash` | Downloads from GitHub Releases |
| **Shell script (Windows)** | `irm https://goozt.github.io/gospeed/install.ps1 \| iex` | Downloads from GitHub Releases |

---

## Manual Publish Steps

These package managers require manual action after the GitHub release is created.

### AUR (Arch Linux)

The PKGBUILD is at `packaging/aur/PKGBUILD`.

```bash
# 1. Update version and checksums
cd packaging/aur
# Edit PKGBUILD: update pkgver to new version
updpkgsums  # updates sha256sums from GitHub release URLs

# 2. Generate .SRCINFO
makepkg --printsrcinfo > .SRCINFO

# 3. Push to AUR
# First time: git clone ssh://aur@aur.archlinux.org/gospeed.git
cd /path/to/aur/gospeed
cp /path/to/gospeed/packaging/aur/PKGBUILD .
cp /path/to/gospeed/packaging/aur/.SRCINFO .
git add PKGBUILD .SRCINFO
git commit -m "Update to v1.4.0"
git push
```

### Nix

The derivation is at `packaging/nix/default.nix`.

```bash
# 1. Update version and hashes in default.nix
#    - version = "1.4.0"
#    - hash: nix-prefetch-url --unpack https://github.com/goozt/gospeed/archive/v1.4.0.tar.gz
#    - vendorHash: build once and nix will tell you the expected hash

# 2. Submit to nixpkgs (first time)
# Fork https://github.com/NixOS/nixpkgs
# Add package to pkgs/by-name/go/gospeed/package.nix
# Submit PR

# 3. Update existing package
# Update the version and hashes in the nixpkgs fork, submit PR
```

### Chocolatey (Windows)

```bash
# 1. Create .nuspec manifest
cat > gospeed.nuspec <<'EOF'
<?xml version="1.0"?>
<package>
  <metadata>
    <id>gospeed</id>
    <version>1.4.0</version>
    <title>gospeed</title>
    <authors>goozt</authors>
    <projectUrl>https://github.com/goozt/gospeed</projectUrl>
    <license type="expression">MIT</license>
    <description>Fast, zero-dependency network speed testing tool</description>
    <tags>network speed test cli</tags>
  </metadata>
</package>
EOF

# 2. Create install script (tools/chocolateyinstall.ps1)
# Download binary from GitHub release and place in package

# 3. Pack and push
choco pack
choco push gospeed.1.4.0.nupkg --source https://push.chocolatey.org/ --api-key YOUR_API_KEY
```

### WinGet (Windows)

```bash
# 1. Install wingetcreate
winget install wingetcreate

# 2. Create/update manifest (auto-generates from installer URL)
wingetcreate update goozt.gospeed \
  --urls "https://github.com/goozt/gospeed/releases/download/v1.4.0/gospeed_1.4.0_windows_amd64.zip" \
  --version 1.4.0 \
  --submit

# This creates a PR to microsoft/winget-pkgs automatically
```

### MacPorts

```bash
# 1. Write a Portfile
cat > Portfile <<'EOF'
PortSystem          1.0
PortGroup           golang 1.0

go.setup            github.com/goozt/gospeed 1.4.0 v
categories          net
license             MIT
maintainers         {goozt @goozt}
description         Fast, zero-dependency network speed testing tool

build.cmd           go build -trimpath -ldflags="-s -w" -o gospeed ./cmd/gospeed

destroot {
    xinstall -m 0755 ${worksrcpath}/gospeed ${destroot}${prefix}/bin/
}
EOF

# 2. Submit PR to https://github.com/macports/macports-ports
```

### apt Repository (self-hosted)

The `.deb` files are already generated by GoReleaser and attached to GitHub releases. To host a proper apt repo:

```bash
# 1. Generate GPG key (one-time)
gpg --full-generate-key
gpg --export --armor YOUR_KEY_ID > gpg-public-key.asc

# 2. Set up repo with reprepro
mkdir -p apt-repo/conf
cat > apt-repo/conf/distributions <<'EOF'
Origin: gospeed
Label: gospeed
Codename: stable
Architectures: amd64 arm64 armhf i386
Components: main
SignWith: YOUR_KEY_ID
EOF

# 3. Add packages
reprepro -b apt-repo includedeb stable dist/gospeed_1.4.0_amd64.deb
reprepro -b apt-repo includedeb stable dist/gospeed-server_1.4.0_amd64.deb

# 4. Host on GitHub Pages or Cloudsmith
# Users add repo with:
#   curl -fsSL https://goozt.github.io/gospeed/gpg-public-key.asc | sudo gpg --dearmor -o /usr/share/keyrings/gospeed.gpg
#   echo "deb [signed-by=/usr/share/keyrings/gospeed.gpg] https://goozt.github.io/gospeed/apt stable main" | sudo tee /etc/apt/sources.list.d/gospeed.list
#   sudo apt update && sudo apt install gospeed
```

### yum / dnf Repository (self-hosted)

The `.rpm` files are already generated by GoReleaser. To host a proper rpm repo:

```bash
# 1. Set up repo directory
mkdir -p rpm-repo

# 2. Copy RPMs and generate metadata
cp dist/*.rpm rpm-repo/
createrepo rpm-repo/

# 3. Sign with GPG
gpg --detach-sign --armor rpm-repo/repodata/repomd.xml

# 4. Host and configure
# Users add repo with:
#   sudo rpm --import https://goozt.github.io/gospeed/gpg-public-key.asc
#   cat > /etc/yum.repos.d/gospeed.repo <<'EOF'
#   [gospeed]
#   name=gospeed
#   baseurl=https://goozt.github.io/gospeed/rpm
#   enabled=1
#   gpgcheck=1
#   gpgkey=https://goozt.github.io/gospeed/gpg-public-key.asc
#   EOF
#   sudo dnf install gospeed
```

---

## Environment Variables Summary

| Variable | Required for |
|---|---|
| `GITHUB_TOKEN` | GitHub Releases, Homebrew tap, Scoop bucket |
| `SNAPCRAFT_STORE_CREDENTIALS` | Snap Store publishing |
| `NPM_TOKEN` | npm publishing |
| Docker Hub login (`docker login`) | Docker Hub publishing |

## Setup Checklist (one-time)

- [ ] Create `goozt/homebrew-tap` GitHub repo
- [ ] Create `goozt/scoop-bucket` GitHub repo
- [ ] Register `gospeed` and `gospeed-server` names on Snap Store
- [ ] Create Docker Hub account/org `goozt`
- [ ] Set up npm org `@goozt` on npmjs.com
- [ ] (Optional) Generate GPG key for apt/rpm repo signing
- [ ] (Optional) Register on AUR
- [ ] (Optional) Register on Chocolatey community repo
