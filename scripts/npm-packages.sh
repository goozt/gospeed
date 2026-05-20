#!/usr/bin/env bash

set -euo pipefail

VERSION="$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo '0.0.0')"
if [ "${1:-}" = "-y" ]; then
  VERSION="${2:-$VERSION}"
elif [ -n "${1:-}" ]; then
  VERSION="$1"
else
  echo "Latest tag found: $VERSION" 
  read -rp "Press Enter to confirm or input a different version: " input_version
  VERSION="${input_version:-$VERSION}"
fi

DIST_DIR="dist"
OUT_DIR="dist/npm"

# Go OS → npm os
declare -A OS_MAP=(
  [linux]=linux
  [darwin]=darwin
  [windows]=win32
  [freebsd]=freebsd
)

# Go arch → npm cpu
declare -A CPU_MAP=(
  [amd64]=x64
  [arm64]=arm64
  [arm]=arm
  [386]=ia32
)

rm -rf "$OUT_DIR"

for dir in "$DIST_DIR"/*/; do
  # skip non-build dirs (e.g. dist/npm/)
  dirname="$(basename "$dir")"

  # Pattern: {binary}_{os}_{arch}_{variant}
  # binary can contain hyphens (gospeed-server), so we parse from the right
  # Strip variant (last _segment): v1, v8.0, sse2, 6, etc.
  without_variant="${dirname%_*}"

  # Extract arch (now last segment)
  arch="${without_variant##*_}"

  # Strip arch to get binary_os
  binary_os="${without_variant%_*}"

  # Extract os (last segment of binary_os)
  os="${binary_os##*_}"

  # Extract binary name (everything before _os)
  binary="${binary_os%_*}"

  # Validate: skip if os or arch not in our maps
  [[ -z "${OS_MAP[$os]+x}" ]] && continue
  [[ -z "${CPU_MAP[$arch]+x}" ]] && continue

  npm_os="${OS_MAP[$os]}"
  npm_cpu="${CPU_MAP[$arch]}"
  pkg_name="@goozt/${binary}-${npm_os}-${npm_cpu}"
  pkg_dir="$OUT_DIR/${binary}-${npm_os}-${npm_cpu}"

  mkdir -p "$pkg_dir"

  # Find the binary file in the dir (handle .exe for windows)
  for f in "$dir"*; do
    [ -f "$f" ] && cp "$f" "$pkg_dir/"
  done

  # Determine bin entries
  bin_name="$binary"
  bin_file="$binary"
  if [[ "$os" == "windows" ]]; then
    bin_file="${binary}.exe"
  fi

  cat > "$pkg_dir/package.json" <<EOF
{
  "name": "${pkg_name}",
  "version": "${VERSION}",
  "os": ["${npm_os}"],
  "cpu": ["${npm_cpu}"],
  "bin": {
    "${bin_name}": "${bin_file}"
  },
  "license": "MIT",
  "author": "Nikhil John <maintainer@goozt.org>"
}
EOF

  echo "Created $pkg_dir"
done

# Create main package.json for client and server
for pkg_dir in "packaging/npm"/*/; do
  [ -d "$pkg_dir" ] || continue
  pkg_name="$(basename "$pkg_dir")"
  npm_pkg_name="@goozt/${pkg_name}"
  case "$pkg_name" in (*server*) pkg_type="server";; *) pkg_type="client" ;; esac
  cat > "$pkg_dir/package.json" <<EOF
{
  "name": "${npm_pkg_name}",
  "version": "${VERSION}",
  "description": "A fast, zero-dependency network speed testing tool written in Go. Client-server architecture for accurate network performance measurement.",
  "keywords": [
    "${pkg_name}",
    "${pkg_type}",
    "cli",
    "performance",
    "network",
    "speed",
    "benchmark",
    "optimization",
    "speed-test"
  ],
  "homepage": "https://gospeed.goozt.org",
  "bugs": {
    "url": "https://github.com/goozt/gospeed/issues"
  },
  "repository": {
    "type": "git",
    "url": "git+https://github.com/goozt/gospeed.git"
  },
  "license": "MIT",
  "author": "Nikhil John <maintainer@goozt.org>",
  "type": "commonjs",
  "publishConfig": {
    "access": "public"
  },
  "optionalDependencies": {
    "${npm_pkg_name}-darwin-arm64": "${VERSION}",
    "${npm_pkg_name}-darwin-x64": "${VERSION}",
    "${npm_pkg_name}-freebsd-arm64": "${VERSION}",
    "${npm_pkg_name}-freebsd-x64": "${VERSION}",
    "${npm_pkg_name}-linux-arm": "${VERSION}",
    "${npm_pkg_name}-linux-arm64": "${VERSION}",
    "${npm_pkg_name}-linux-ia32": "${VERSION}",
    "${npm_pkg_name}-linux-x64": "${VERSION}",
    "${npm_pkg_name}-win32-arm64": "${VERSION}",
    "${npm_pkg_name}-win32-ia32": "${VERSION}",
    "${npm_pkg_name}-win32-x64": "${VERSION}"
  },
  "bin": {
    "${pkg_name}": "./index.js"
  }
}
EOF
  echo "Created $pkg_dir""package.json"
done

echo "Done. Generated npm packages in $OUT_DIR/"
