#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
DATA_DIR="$(go tool dasel -f "$SCRIPT_DIR/../config.toml" -s common.data_dir -w yaml)"
LOG_DIR="$(go tool dasel -f "$SCRIPT_DIR/../config.toml" -s common.log_dir -w yaml)"
DSN="$(go tool dasel -f "$SCRIPT_DIR/../config.toml" -s common.db_dsn -w yaml)"

TEMPDIR="$(mktemp -d)"

mkdir "$TEMPDIR/data"
mkdir "$TEMPDIR/logs"
mkdir "$TEMPDIR/instance"

cp -r $DATA_DIR "$TEMPDIR/data"
cp -r $LOG_DIR "$TEMPDIR/logs"
cp -r "$SCRIPT_DIR/.." "$TEMPDIR/instance"

pg_dump "$DSN" >"$TEMPDIR/dump.sql"

tar -C $TEMPDIR --zstd -cf kilonova.tar.zst .

rm -rf $TEMPDIR
