"""Entry points for the ``gospeed`` and ``gospeed-server`` console scripts.

The wheel is universal (no binary inside); the real binaries are fetched from
GitHub Releases on first invocation and cached under
``$XDG_CACHE_HOME/gospeed/<version>/`` (or ``~/.cache/gospeed/...``).
"""

from __future__ import annotations

import hashlib
import os
import platform
import re
import shutil
import stat
import sys
import tarfile
import tempfile
import urllib.request
import zipfile
from pathlib import Path

OWNER = "goozt"
REPO = "gospeed"
BINARIES = ["gospeed", "gospeed-server"]
PROJECT_NAME = "gospeed"

from . import __version__  # noqa: E402

_PLATFORMS = {
    "linux": "linux",
    "darwin": "darwin",
    "win32": "windows",
    "freebsd": "freebsd",
}

_ARCHES = {
    "x86_64": "amd64",
    "amd64": "amd64",
    "aarch64": "arm64",
    "arm64": "arm64",
    "armv7l": "arm",
    "i386": "386",
    "i686": "386",
    "s390x": "s390x",
    "ppc64le": "ppc64le",
}


def _platform_arch() -> tuple[str, str]:
    sysname = sys.platform
    if sysname.startswith("linux"):
        sysname = "linux"
    if sysname.startswith("freebsd"):
        sysname = "freebsd"
    if sysname not in _PLATFORMS:
        raise SystemExit(f"unsupported platform: {sys.platform}")
    machine = platform.machine().lower()
    if machine not in _ARCHES:
        raise SystemExit(f"unsupported arch: {machine}")
    return _PLATFORMS[sysname], _ARCHES[machine]


def _cache_dir() -> Path:
    base = os.environ.get("XDG_CACHE_HOME") or str(Path.home() / ".cache")
    d = Path(base) / PROJECT_NAME / __version__
    d.mkdir(parents=True, exist_ok=True)
    return d


def _download(url: str, dest: Path) -> None:
    req = urllib.request.Request(url, headers={"User-Agent": f"{PROJECT_NAME}-pypi/{__version__}"})
    with urllib.request.urlopen(req, timeout=120) as resp:  # noqa: S310 — trusted github.com
        if resp.status != 200:
            raise SystemExit(f"HTTP {resp.status} for {url}")
        with dest.open("wb") as fh:
            shutil.copyfileobj(resp, fh)


def _verify_sha256(archive: Path, sums: Path) -> None:
    pattern = re.compile(rf"^([0-9a-f]{{64}})\s+\*?{re.escape(archive.name)}\s*$", re.M)
    text = sums.read_text()
    m = pattern.search(text)
    if not m:
        print(f"warning: {archive.name} not found in checksums.txt — skipping verification", file=sys.stderr)
        return
    want = m.group(1)
    h = hashlib.sha256()
    with archive.open("rb") as fh:
        for chunk in iter(lambda: fh.read(1 << 20), b""):
            h.update(chunk)
    got = h.hexdigest()
    if got != want:
        raise SystemExit(f"checksum mismatch for {archive.name}: got {got}, want {want}")


def _extract(archive: Path, into: Path) -> None:
    if archive.name.endswith(".tar.gz") or archive.name.endswith(".tgz"):
        with tarfile.open(archive) as tf:
            tf.extractall(into)
    elif archive.name.endswith(".zip"):
        with zipfile.ZipFile(archive) as zf:
            zf.extractall(into)
    else:
        raise SystemExit(f"unknown archive type: {archive.name}")


def _ensure_binary(name: str) -> Path:
    os_, arch = _platform_arch()
    cache = _cache_dir()
    exe = cache / (f"{name}.exe" if os_ == "windows" else name)
    if exe.exists():
        return exe

    ext = "zip" if os_ == "windows" else "tar.gz"
    archive_name = f"{PROJECT_NAME}_{__version__}_{os_}_{arch}.{ext}"
    base = f"https://github.com/{OWNER}/{REPO}/releases/download/v{__version__}"
    url = f"{base}/{archive_name}"
    sums_url = f"{base}/checksums.txt"

    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        archive = tmp / archive_name
        sums = tmp / "checksums.txt"
        print(f"downloading {url}", file=sys.stderr)
        _download(url, archive)
        try:
            _download(sums_url, sums)
            _verify_sha256(archive, sums)
        except SystemExit:
            raise
        except Exception as e:  # noqa: BLE001 — soft-fail checksum download
            print(f"warning: could not fetch checksums.txt ({e}); skipping verification", file=sys.stderr)
        _extract(archive, cache)

    if not exe.exists():
        raise SystemExit(f"extraction did not produce {exe}")
    exe.chmod(exe.stat().st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)
    return exe


def _run(name: str) -> None:
    exe = _ensure_binary(name)
    if sys.platform == "win32":
        import subprocess

        sys.exit(subprocess.call([str(exe), *sys.argv[1:]]))
    else:
        os.execv(str(exe), [str(exe), *sys.argv[1:]])


def main() -> None:
    _run("gospeed")


def main_server() -> None:
    _run("gospeed-server")


if __name__ == "__main__":
    main()
