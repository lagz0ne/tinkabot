#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="${1:-dist/release}"
case "$dist" in
  /*) ;;
  *) dist="$root/$dist" ;;
esac

version="${TB_VERSION:-$(node -p "require('./package.json').version" 2>/dev/null || true)}"
if [[ -z "$version" || "$version" == "undefined" ]]; then
  echo "release-package: package.json version is required" >&2
  exit 1
fi

tag="v${version#v}"
target="$(cd "$root/substrate/go" && go env GOHOSTOS)-$(cd "$root/substrate/go" && go env GOHOSTARCH)"
name="tinkabot-$tag-$target"
pkg="$dist/$name"
archive="$dist/$name.tar.gz"
commit="$(git -C "$root" rev-parse --short HEAD 2>/dev/null || echo unknown)"
if [[ "$commit" != "unknown" ]] && [[ -n "$(git -C "$root" status --porcelain 2>/dev/null)" ]]; then
  commit="$commit-dirty"
fi
built_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

rm -rf "$pkg" "$archive" "$archive.sha256"
mkdir -p "$dist"

TB_VERSION="${tag#v}" TB_COMMIT="$commit" TB_BUILT_AT="$built_at" bash "$root/scripts/package-tinkabot.sh" "$pkg"

mkdir -p "$pkg/examples" "$pkg/docs/manual" "$pkg/release"
cp "$root/README.md" "$pkg/README.md"
cp "$root/LICENSE" "$pkg/LICENSE"
cp "$root/examples/README.md" "$pkg/examples/README.md"
cp -R "$root/examples/clock" "$pkg/examples/clock"
cp -R "$root/examples/builder" "$pkg/examples/builder"
rm -rf "$pkg/examples/builder/node_modules"
cp "$root/docs/manual/v1.md" "$pkg/docs/manual/v1.md"
cp "$root/release/v1.json" "$pkg/release/v1.json"

cat > "$pkg/release.json" <<JSON
{
  "name": "tinkabot",
  "version": "${tag#v}",
  "target": "$target",
  "gitCommit": "$commit",
  "builtAt": "$built_at",
  "binary": "tinkabot",
  "bundledBwrap": "libexec/tinkabot/bwrap",
  "bundledNats": "libexec/tinkabot/nats",
  "examples": [
    "examples/clock",
    "examples/builder"
  ]
}
JSON

(cd "$dist" && tar -czf "$archive" "$name")
(cd "$dist" && sha256sum "$name.tar.gz" > "$name.tar.gz.sha256")

echo "release package $archive"
echo "checksum $archive.sha256"
