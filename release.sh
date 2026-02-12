#!/bin/bash
set -euo pipefail

usage() {
  echo "Usage: ./release.sh [patch|minor|major]"
  exit 1
}

[[ $# -ne 1 ]] && usage

bump="$1"
[[ "$bump" != "patch" && "$bump" != "minor" && "$bump" != "major" ]] && usage

# Get latest tag, default to v0.0.0
latest=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
version="${latest#v}"

IFS='.' read -r major minor patch <<< "$version"

case "$bump" in
  major) major=$((major + 1)); minor=0; patch=0 ;;
  minor) minor=$((minor + 1)); patch=0 ;;
  patch) patch=$((patch + 1)) ;;
esac

next="v${major}.${minor}.${patch}"

echo "Current: $latest"
echo "Next:    $next"
echo ""
read -rp "Release $next? [y/N] " confirm
[[ "$confirm" != "y" && "$confirm" != "Y" ]] && echo "Aborted." && exit 0

git tag "$next"
git push origin "$next"

echo ""
echo "Tagged and pushed $next. Release workflow running."
echo "https://github.com/arturopuente/var/actions"
