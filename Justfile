
default:
	just --list

katex:
	yarn --cwd ./web/assets install
	cp web/assets/node_modules/katex/dist/katex.min.js sudoapi/mdrenderer/knkatex/katex.min.js && \
		echo "Copied katex.min.js to target directory" || \
		echo "Failed to copy katex.min.js to target directory"

export MAXMIND_PATH := x"~/.local/share/GeoIP/GeoLite2-City.mmdb"
maxmind:
	mkdir -p $HOME/.local/share/GeoIP

	curl -sSL "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb" -o '${MAXMIND_PATH}'

	@echo "Downloaded GeoLite2-City database."

	tmp=$(mktemp) \
	echo '$tmp' && \
	jq ".\"integrations.maxmind.db_path\" = \"${MAXMIND_PATH}\"" flags.json > '$tmp' && mv '$tmp' flags.json


	echo "Updated configuration file with correct path."