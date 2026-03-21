# Future Package Manager Support

This document tracks deferred package manager integrations and future distribution plans.

## Currently Supported

| Channel | Status |
|---|---|
| GoReleaser (GitHub Releases) | Active |
| npm (`@goozt/gospeed`, `@goozt/gospeed-server`) | Active |
| Docker (distroless) | Active |
| `go install` | Active |
| Shell scripts (`install.sh`, `install.ps1`) | Active |
| Homebrew | Added (pending tap repo creation) |
| Scoop | Added (pending bucket repo creation) |
| Snap | Added (pending snapcraft publish) |
| Docker Hub | Added (pending `goozt/gospeed-server` push) |
| deb/rpm packages | Added (attached to GitHub releases) |
| AUR (Arch Linux) | PKGBUILD ready |
| Nix | Derivation ready |

## Deferred: Needs External Repo Hosting

### apt / dpkg Repository

Generating `.deb` packages is handled by GoReleaser `nfpms`. To enable `apt install gospeed`, a hosted apt repository is needed:

- **GPG key**: Generate a signing key for package signatures
- **Hosting options**: Cloudsmith (free for open source), GitHub Pages with `reprepro`, or Packagecloud
- **User setup**: Users add the repo with `curl | gpg --dearmor` + `sources.list.d` entry
- **Maintenance**: Each release needs repo metadata regenerated

### yum / dnf / zypper Repository

Same as apt but for RPM-based distros. GoReleaser generates `.rpm` files. Hosting requires:

- **GPG signing**: Same key can be reused
- **Hosting**: Cloudsmith, GitHub Pages with `createrepo`, or Packagecloud
- **Metadata**: `repodata/repomd.xml` must be regenerated per release

### Chocolatey

Windows package manager with moderated community repository:

- Create `.nuspec` manifest describing the package
- Package as `.nupkg` (NuGet format)
- Submit to [community.chocolatey.org](https://community.chocolatey.org) (moderated review)
- Each version requires re-submission
- Can automate with `choco push` in CI

### WinGet

Microsoft's package manager for Windows:

- Create a manifest YAML (installer URL, hash, metadata)
- Submit PR to [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs)
- Each version requires a new PR
- Can automate with [wingetcreate](https://github.com/microsoft/winget-create)

### MacPorts

Alternative to Homebrew for macOS:

- Write a `Portfile` with build instructions
- Submit to MacPorts repository via PR
- Less popular than Homebrew, more formal review process

### asdf

Version manager supporting multiple tools:

- Create an asdf plugin repository (shell scripts)
- Plugin defines how to download/install each version
- Publish to [asdf-plugins](https://github.com/asdf-vm/asdf-plugins) index
- Another repository to maintain

### Conda (conda-forge)

- Write a conda recipe (`meta.yaml`)
- Submit to [conda-forge/staged-recipes](https://github.com/conda-forge/staged-recipes)
- Unusual for non-Python tools but technically possible
- Review process required

## Deferred: Low Priority

### Flatpak

Possible but unusual for CLI tools. Requires:
- Flatpak manifest with freedesktop metadata
- Submit to Flathub
- Designed for GUI applications

### AppImage

Possible but very unusual for CLI tools. Requires:
- AppImage packaging with AppDir structure
- Desktop file and icon (GUI-oriented)
- No central repository

## Deferred: Requires New Project

### pip / pipx / poetry / uv (PyPI)

**Blocked on `setuptools-go` — a general-purpose Python build backend for Go binaries.**

**Background:** PyPI distributes Python packages as wheels. Ruff (a Rust binary) proved that native binaries can be distributed via PyPI using platform-specific wheels with a thin Python wrapper.

**Precedent — how ruff does it:**
- Uses `maturin` (Rust build tool) as PEP 517 build backend with `bindings = "bin"`
- Wheels named `ruff-{ver}-py3-none-{platform}.whl`
- Python package contains:
  - `__main__.py` — entry point that `os.execvp(binary)`
  - `_find_ruff.py` — locates binary in Python's scripts directory
- Binary lands in `~/.local/bin/` (or equivalent)

**What `setuptools-go` would do:**
- Custom PEP 517 build backend (Python, no Rust dependency)
- Reads Go module path + build config from `[tool.setuptools-go]` in `pyproject.toml`
- Runs `go build` with `GOOS`/`GOARCH` cross-compilation during wheel build
- Places binary in wheel's scripts directory
- Generates platform-specific wheel tags (`py3-none-{platform}`)
- Auto-generates thin Python wrapper (find binary + execvp)
- Defaults to `CGO_ENABLED=0` static builds

**Example pyproject.toml for gospeed:**
```toml
[build-system]
requires = ["setuptools-go"]
build-backend = "setuptools_go"

[project]
name = "gospeed"
version = "1.3.2"

[project.scripts]
gospeed = "gospeed:main"

[tool.setuptools-go]
module = "github.com/goozt/gospeed/cmd/gospeed"
go-version = "1.25"
cgo = false
strip = true
ldflags = ["-s", "-w"]
```

**Why not a Rust shim:** Using a minimal Rust crate + maturin to wrap the Go binary was considered but rejected — maturin expects Rust compilation, cross-compilation would need both toolchains, and the Rust layer adds nothing. A pure Python build backend calling `go build` is simpler.

**Prior art:** [asottile/setuptools-golang](https://github.com/asottile-archive/setuptools-golang) (archived) — compiled Go into CPython extensions via shared objects. Broke in Go 1.21 because Go doesn't support multiple `.so` in one process. Our approach is fundamentally different (pre-built binaries, not extensions).

Once `setuptools-go` is built, all pip-compatible tools (pip, pipx, poetry, uv) get gospeed support automatically.

### Cargo (crates.io)

**Not feasible.** crates.io only accepts Rust crates compiled from Rust source. There is no mechanism to publish Go binaries. No known workaround.

## Already Covered

| Manager | Notes |
|---|---|
| **yarn** | npm-compatible — works with existing `@goozt/gospeed` packages |
| **pnpm** | npm-compatible — works with existing `@goozt/gospeed` packages |
