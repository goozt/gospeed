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

echo "Using version: $VERSION"
exit 0;

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

echo "Done. Generated npm packages in $OUT_DIR/"
