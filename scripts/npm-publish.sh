#!/usr/bin/env bash

set -euo pipefail

if grep -q "registry.npmjs.org/:_authToken=" ~/.npmrc; then
  echo "Existing npm auth token found. Skipping login..."
else
  echo "No auth token found. Initiating npm login for @goozt scope..."
  npm login --scope=@goozt
fi

NPM_TOKEN=$(grep '_authToken' ~/.npmrc | cut -d'=' -f2)

if [ -z "$NPM_TOKEN" ]; then
  echo "Error: NPM token not found. Please run 'npm login' first."
  exit 1
fi

# Publish platform-specific packages first
for dir in dist/npm/*/; do
  [ -f "${dir}package.json" ] || continue
  name=$(node -p "require('./${dir}package.json').name")
  version=$(node -p "require('./${dir}package.json').version")
  if npm view "${name}@${version}" version >/dev/null 2>&1; then
    echo "Skipping ${name}@${version} (already published)"
    continue
  fi
  npm publish "$dir" --access public
done

# Publish root packages
for pkg in gospeed gospeed-server; do
  dir="packaging/npm/${pkg}"
  cp README.md "${dir}/README.md"
  name=$(node -p "require('./${dir}/package.json').name")
  version=$(node -p "require('./${dir}/package.json').version")
  if npm view "${name}@${version}" version >/dev/null 2>&1; then
    echo "Skipping ${name}@${version} (already published)"
    rm "${dir}/README.md"
    continue
  fi
  (cd "$dir" && npm publish --access public)
  rm "${dir}/README.md"
done