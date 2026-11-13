
default:
	just --list

export MAXMIND_PATH := x"~/.local/share/GeoIP/GeoLite2-City.mmdb"
maxmind:
	mkdir -p $HOME/.local/share/GeoIP

	curl -sSL "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb" -o ${MAXMIND_PATH}"

	@echo "Downloaded GeoLite2-City database."

	@echo "Set KN_MAXMIND_DB=${MAXMIND_PATH} in your .env."

uv:
	# TODO use data dir path dynamically
	curl -LsSf https://astral.sh/uv/install.sh | env UV_UNMANAGED_INSTALL="/home/alexv/src/kninfo/data/uv/bin" UV_NO_MODIFY_PATH=1 sh