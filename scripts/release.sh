#!/usr/bin/env bash

set -euo pipefail

RELEASE_FILE="RELEASE_VERSION"

if [[ ! -f "$RELEASE_FILE" ]]; then
  echo "No $RELEASE_FILE file found. Run 'task bump VERSION=x.y.z' first."
  exit 1
fi

VERSION="$(cat "$RELEASE_FILE" | tr -d '[:space:]')"

if [[ -z "$VERSION" ]]; then
  echo "$RELEASE_FILE is empty."
  exit 1
fi

TAG="v${VERSION}"

# Check that the git tag exists
if ! git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "Git tag $TAG does not exist. Run 'task bump VERSION=$VERSION' first."
  exit 1
fi

# Check that this version hasn't already been released on GitHub
if gh release view "$TAG" >/dev/null 2>&1; then
  echo "GitHub release $TAG already exists. Skipping."
  rm -f "$RELEASE_FILE"
  exit 0
fi

echo "Releasing $TAG..."

# GoReleaser handles: GitHub release, Homebrew tap, Scoop bucket,
# Snap, Docker Hub, deb/rpm packages
goreleaser release --clean

# npm packages
echo "Publishing npm packages..."
bash scripts/npm-packages.sh "$VERSION"
bash scripts/npm-publish.sh

# Clean up
rm -f "$RELEASE_FILE"
echo "Release $TAG complete."
