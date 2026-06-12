#!/bin/sh
# Clock app backend: emits the bundle.clock projection and the clock page as
# framed JSON effects on stdout. The process never sees NATS — the platform
# decides what materializes.
now=$(date -u +%Y-%m-%dT%H:%M:%SZ)
secs=$(date +%s)
page="<!doctype html><html><head><meta charset=utf-8><title>clock</title></head><body style=font-family:monospace;margin:2rem><h1>tinkabot clock</h1><p>backend state from /projections/bundle.clock (auto-refresh 2s)</p><pre id=s>loading...</pre><script>async function r(){try{const x=await fetch('/projections/bundle.clock');document.getElementById('s').textContent=JSON.stringify(await x.json(),null,2)}catch(e){document.getElementById('s').textContent=String(e)}}r();setInterval(r,2000)</script></body></html>"
b1="{\"kind\":\"script.effect\",\"effectType\":\"projection\",\"projectionId\":\"bundle.clock\",\"snapshotRevision\":\"snap-$secs\",\"artifactRevision\":\"clock.rev.1\",\"sequence\":$secs,\"value\":{\"page\":\"clock\",\"renderedAt\":\"$now\",\"unix\":$secs}}"
b2="{\"kind\":\"script.effect\",\"effectType\":\"artifact\",\"artifactName\":\"bundle/clock/index.html\",\"artifactRevision\":\"clock.rev.1\",\"mediaType\":\"text/html\",\"body\":\"$page\"}"
printf 'Content-Length: %s\r\n\r\n%s' "${#b1}" "$b1"
printf 'Content-Length: %s\r\n\r\n%s' "${#b2}" "$b2"
