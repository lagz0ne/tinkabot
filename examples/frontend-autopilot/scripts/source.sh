#!/bin/sh
set -eu

now=$(date +%s%3N)
author=$(sed -nE 's/.*"authoredBy"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/p' ui/options-site.provenance.json | head -n 1)
author=${author:-unknown}
html=$(awk '
  {
    gsub(/\\/, "\\\\")
    gsub(/"/, "\\\"")
    printf "%s\\n", $0
  }
' ui/options-site.html)
html=${html%\\n}
body="{\"kind\":\"script.effect\",\"effectType\":\"projection\",\"projectionId\":\"src\",\"snapshotRevision\":\"snap-$now\",\"artifactRevision\":\"src.rev.$now\",\"sequence\":$now,\"value\":{\"files\":{\"index.html\":\"$html\"},\"provenance\":{\"ui\":\"options-site.html\",\"author\":\"$author\"}}}"
printf 'Content-Length: %s\r\n\r\n%s' "${#body}" "$body"
