#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="${1:-$(mktemp -d /tmp/tinkabot-tinkalet-package.XXXXXX)}"
case "$dist" in
  /*) ;;
  *) dist="$root/$dist" ;;
esac

mkdir -p "$dist"
bash "$root/scripts/release-package.sh" "$dist" >/dev/null
archive="$(find "$dist" -maxdepth 1 -name 'tinkabot-v*.tar.gz' | sort | tail -n 1)"
if [[ -z "$archive" ]]; then
  echo "smoke-tinkalet-package: release archive missing" >&2
  exit 1
fi

tar -xzf "$archive" -C "$dist"
pkg="${archive%.tar.gz}"
for file in tinkabot tinkalet libexec/tinkabot/bwrap libexec/tinkabot/nats; do
  if [[ ! -x "$pkg/$file" ]]; then
    echo "smoke-tinkalet-package: $file is not executable" >&2
    exit 1
  fi
done
grep -q '"tinkalet"' "$pkg/release.json"

"$pkg/tinkabot" --version >/dev/null
"$pkg/tinkalet" --help | grep -q 'usage: tinkalet'
"$pkg/tinkalet" --version | grep -q '^tinkalet '

store="$(mktemp -d /tmp/tinkabot-store.XXXXXX)"
cfg="$(mktemp -d /tmp/tinkalet-config.XXXXXX)"
data="$(mktemp -d /tmp/tinkalet-data.XXXXXX)"
home="$(mktemp -d /tmp/tinkalet-home.XXXXXX)"
log="$dist/tinkabot.log"
err="$dist/tinkabot.err"
pid=""

cleanup() {
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    kill -TERM "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT

(cd "$pkg" && PATH="/usr/bin:/bin" ./tinkabot --store "$store" --shell 127.0.0.1:0 --bundle examples/clock >"$log" 2>"$err") &
pid=$!

for _ in {1..200}; do
  if grep -q '^shell  http://127\.0\.0\.1:' "$log"; then
    break
  fi
  if ! kill -0 "$pid" 2>/dev/null; then
    echo "smoke-tinkalet-package: tinkabot exited before posture" >&2
    sed -n '1,120p' "$log" >&2 || true
    sed -n '1,120p' "$err" >&2 || true
    exit 1
  fi
  sleep 0.1
done

shell_url="$(awk '$1 == "shell" { print $2; exit }' "$log")"
if [[ -z "$shell_url" ]]; then
  echo "smoke-tinkalet-package: shell URL missing" >&2
  sed -n '1,120p' "$log" >&2 || true
  sed -n '1,120p' "$err" >&2 || true
  exit 1
fi

run_tinkalet() {
  env -i \
    HOME="$home" \
    TINKALET_CONFIG_DIR="$cfg" \
    TINKALET_DATA_DIR="$data" \
    PATH="/nonexistent" \
    "$pkg/tinkalet" "$@"
}

mv "$pkg/libexec/tinkabot/nats" "$pkg/libexec/tinkabot/nats.disabled"

got="$(run_tinkalet profile import local --store "$store" --name local)"
[[ "$got" == "profile local imported" ]]
got="$(run_tinkalet profile use local)"
[[ "$got" == "profile local selected" ]]

before="$dist/projection-before.json"
after="$dist/projection-after.json"
for _ in {1..150}; do
  if curl -fsS "$shell_url/projections/bundle.clock.state" >"$before"; then
    break
  fi
  sleep 0.1
done
if [[ ! -s "$before" ]]; then
  echo "smoke-tinkalet-package: clock projection missing before trigger" >&2
  exit 1
fi

sleep 2
got="$(run_tinkalet trigger bundle.clock.tick --request-id smoke-tinkalet-package-1)"
[[ "$got" == "profile local accepted bundle.clock.tick" ]]

for _ in {1..150}; do
  if curl -fsS "$shell_url/projections/bundle.clock.state" >"$after" && ! cmp -s "$before" "$after"; then
    echo "package root $pkg"
    echo "shell $shell_url"
    echo "tinkalet trigger accepted and projection changed"
    echo "packaged nats sidecar was removed before the Tinkalet commands"
    echo "gate:tinkalet-package passed"
    exit 0
  fi
  sleep 0.1
done

echo "smoke-tinkalet-package: projection did not change after Tinkalet trigger" >&2
exit 1
