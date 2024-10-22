#!/usr/bin/env bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
OUT_PATH="$HOME/.local/share/GeoIP/GeoLite2-City.mmdb"

mkdir -p $HOME/.local/share/GeoIP
curl -sSL "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb" -o "$OUT_PATH"

echo "Downloaded GeoLite2-City database."

tmp=$(mktemp)
echo $tmp
jq ".\"integrations.maxmind.db_path\" = \"$OUT_PATH\"" flags.json > "$tmp" && mv "$tmp" "flags.json"


echo "Updated configuration file with correct path."