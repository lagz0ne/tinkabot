#!/bin/sh
# Builder app source: emits the bundle.builder.src projection -- a tiny Vite app
# source map -- as a framed JSON effect on stdout. The build filter watches this
# projection and rebuilds the app each time the source changes; never sees NATS.
secs=$(date +%s)
html='<!doctype html><html><head><meta charset=utf-8><title>builder</title></head><body><script type=module src=/src/main.ts></script></body></html>'
main='const ts = '"$secs"'; const hue = ts % 360; document.body.style.background = `hsl(${hue}, 70%, 65%)`; document.body.style.font = `2rem/1.4 system-ui, sans-serif`; document.body.style.margin = `3rem`; const h = document.createElement(`h1`); h.textContent = `built from source emitted at ${ts}`; document.body.appendChild(h);'
v="{\"files\":{\"index.html\":\"$html\",\"src/main.ts\":\"$main\"}}"
b="{\"kind\":\"script.effect\",\"effectType\":\"projection\",\"projectionId\":\"bundle.builder.src\",\"snapshotRevision\":\"snap-$secs\",\"artifactRevision\":\"src.rev.1\",\"sequence\":$secs,\"value\":$v}"
printf 'Content-Length: %s\r\n\r\n%s' "${#b}" "$b"
