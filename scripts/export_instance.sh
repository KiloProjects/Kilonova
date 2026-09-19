#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
# Same contract as the binary: .env in the repo root, overridden by the real environment.
if [ -f "$SCRIPT_DIR/../.env" ]; then
	set -a; source "$SCRIPT_DIR/../.env"; set +a
fi
# Logs live under $KN_DATA_DIR/logs, so copying the data dir covers them.
DATA_DIR="${KN_DATA_DIR:?KN_DATA_DIR must be set}"
DSN="${KN_DB_DSN:-}"

TEMPDIR="$(mktemp -d)"

mkdir "$TEMPDIR/data"
mkdir "$TEMPDIR/instance"

cp -r "$DATA_DIR" "$TEMPDIR/data"

cp -r "$SCRIPT_DIR/.." "$TEMPDIR/instance"

pg_dump "$DSN" >"$TEMPDIR/dump.sql"

tar -C "$TEMPDIR" --zstd -cf kilonova.tar.zst .

rm -rf "$TEMPDIR"
