#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out="${1:-dist/tinkabot}"
case "$out" in
  /*) ;;
  *) out="$root/$out" ;;
esac

if [[ -n "${TB_BWRAP:-}" ]]; then
  bwrap="$TB_BWRAP"
else
  bwrap="$(command -v bwrap || true)"
fi

if [[ -z "$bwrap" || ! -f "$bwrap" || ! -x "$bwrap" ]]; then
  echo "package-tinkabot: executable bwrap not found; install bubblewrap or set TB_BWRAP" >&2
  exit 1
fi

version="${TB_VERSION:-$(node -p "require('./package.json').version || 'dev'" 2>/dev/null || echo dev)}"
commit="${TB_COMMIT:-$(git -C "$root" rev-parse --short HEAD 2>/dev/null || echo unknown)}"
if [[ -z "${TB_COMMIT:-}" ]] && [[ "$commit" != "unknown" ]] && [[ -n "$(git -C "$root" status --porcelain 2>/dev/null)" ]]; then
  commit="$commit-dirty"
fi
built_at="${TB_BUILT_AT:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
ldflags="-X main.version=$version -X main.commit=$commit -X main.builtAt=$built_at"

rm -rf "$out"
mkdir -p "$out/libexec/tinkabot"

(cd "$root/substrate/go" && go build -ldflags "$ldflags" -o "$out/tinkabot" ./cmd/tinkabot)
(cd "$root/substrate/go" && go build -ldflags "$ldflags" -o "$out/tinkalet" ./cmd/tinkalet)
(cd "$root/tools/natscli" && go build -o "$out/libexec/tinkabot/nats" github.com/nats-io/natscli/nats)
cp "$bwrap" "$out/libexec/tinkabot/bwrap"
chmod 0755 "$out/tinkabot" "$out/tinkalet" "$out/libexec/tinkabot/bwrap" "$out/libexec/tinkabot/nats"

echo "packaged $out"
echo "version $version"
echo "binary $out/tinkalet"
echo "bundled bwrap $out/libexec/tinkabot/bwrap"
echo "bundled nats $out/libexec/tinkabot/nats"
