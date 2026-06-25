#!/bin/sh
# Clock present filter: long-lived process fed one JSON line per state change
# by the platform on stdin. Never sees NATS — derives bundle.clock.view from
# bundle.clock.state and emits framed effects on stdout.
prev_ms=""
while IFS= read -r line; do
  received_ms=$(date +%s%3N)
  parsed=$(printf '%s\n' "$line" | awk '
    {
      rendered = unix = ms = seq = ""
      if (match($0, /"renderedAt":"[^"]+"/)) {
        rendered = substr($0, RSTART + 14, RLENGTH - 15)
      }
      if (match($0, /"unix":[0-9]+/)) {
        unix = substr($0, RSTART + 7, RLENGTH - 7)
      }
      if (match($0, /"ms":[0-9]+/)) {
        ms = substr($0, RSTART + 5, RLENGTH - 5)
      }
      if (match($0, /"seq":[0-9]+/)) {
        seq = substr($0, RSTART + 6, RLENGTH - 6)
      }
      printf "%s\n%s\n%s\n%s\n", rendered, unix, ms, seq
    }')
  rendered=$(printf '%s\n' "$parsed" | sed -n '1p')
  unix=$(printf '%s\n' "$parsed" | sed -n '2p')
  ms=$(printf '%s\n' "$parsed" | sed -n '3p')
  seq=$(printf '%s\n' "$parsed" | sed -n '4p')
  [ -z "$rendered" ] && continue
  [ -z "$unix" ] && continue
  [ -z "$ms" ] && ms=$((unix * 1000))
  [ -z "$seq" ] && seq="$ms"
  latency_ms=$((received_ms - ms))
  [ "$latency_ms" -lt 0 ] && latency_ms=0
  interval_json=null
  if [ -n "$prev_ms" ]; then
    interval_json=$((ms - prev_ms))
  fi
  prev_ms="$ms"
  b="{\"kind\":\"script.effect\",\"effectType\":\"projection\",\"projectionId\":\"view\",\"snapshotRevision\":\"snap-v-$seq\",\"artifactRevision\":\"clock.rev.1\",\"sequence\":$seq,\"value\":{\"display\":\"clock at $rendered\",\"tick\":$seq,\"unix\":$unix,\"ms\":$ms,\"seq\":$seq,\"sourceIntervalMs\":$interval_json,\"filterReceivedMs\":$received_ms,\"filterLatencyMs\":$latency_ms}}"
  printf 'Content-Length: %s\r\n\r\n%s' "${#b}" "$b"
done
