#!/bin/sh
# Clock present filter: long-lived process fed one JSON line per state change
# by the platform on stdin. Never sees NATS — derives bundle.clock.view from
# bundle.clock.state and emits framed effects on stdout.
while IFS= read -r line; do
  rendered=$(printf '%s' "$line" | sed -n 's/.*"renderedAt":"\([^"]*\)".*/\1/p')
  unix=$(printf '%s' "$line" | sed -n 's/.*"unix":\([0-9]*\).*/\1/p')
  [ -z "$rendered" ] && continue
  [ -z "$unix" ] && continue
  b="{\"kind\":\"script.effect\",\"effectType\":\"projection\",\"projectionId\":\"bundle.clock.view\",\"snapshotRevision\":\"snap-v-$unix\",\"artifactRevision\":\"clock.rev.1\",\"sequence\":$unix,\"value\":{\"display\":\"clock at $rendered\",\"tick\":$unix}}"
  printf 'Content-Length: %s\r\n\r\n%s' "${#b}" "$b"
done
