#!/bin/sh
set -eu

bin=${1:?binary required}

tmp=${TMPDIR:-/tmp}/fzf-grid-span-integration.$$
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
mkdir -p "$tmp"

actual=$(printf '@@2@@id1|alpha\n@@1@@id2|beta\n' |
  "$bin" --grid=3 --grid-span-prefix=@@ --delimiter='|' --with-nth=2 --accept-nth=1 --filter=alpha)
[ "$actual" = id1 ]

actual=$(printf '@@2@@alpha\0@@1@@beta\0' |
  "$bin" --grid=3 --grid-span-prefix=@@ --read0 --filter=alpha)
[ "$actual" = alpha ]

actual=$(printf '@@3@@selected\n' |
  "$bin" --grid=3 --grid-span-prefix=@@ --filter=selected)
[ "$actual" = selected ]
