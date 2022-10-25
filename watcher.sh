#!/bin/sh
tmux \; new-session ./runkn.sh \; split-window -h "while true; do inotifywait -q -e attrib web/translations.toml; ./scripts/toml_gen.py; done" \; setw remain-on-exit on \; split-window -v yarn --cwd ./web/assets watchJS \; split-window -v yarn --cwd ./web/assets watchCSS \; select-layout tiled \; attach
