# gospeed (PyPI wrapper)

Thin Python launcher for the [`gospeed`](https://github.com/goozt/gospeed) Go binaries.

The wheel itself contains no binary. On first invocation each command
downloads the matching prebuilt archive from GitHub Releases (using the
wheel's version) and caches it under `~/.cache/gospeed/<version>/`.
Subsequent runs exec the cached binary directly.

Install (Fury PyPI):

```sh
pipx install --index-url https://pypi.fury.io/nikhiljohn10/ gospeed
# or
pip install --index-url https://pypi.fury.io/nikhiljohn10/ gospeed
```

The wheel exposes two console scripts: `gospeed` (client) and
`gospeed-server` (server).

```sh
gospeed --version
gospeed-server --version
```
