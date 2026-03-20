#!/usr/bin/env bash

set -euo pipefail

# Publish platform-specific packages first
for dir in dist/npm/*/; do
  npm publish "$dir" --access public
done

# Publish root packages
cp README.md scripts/gospeed/README.md
cp README.md scripts/gospeed-server/README.md
(cd scripts/gospeed && npm publish --access public)
(cd scripts/gospeed-server && npm publish --access public)
rm scripts/gospeed/README.md scripts/gospeed-server/README.md