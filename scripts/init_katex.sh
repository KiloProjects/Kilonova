#!/bin/sh
# Use the currently installed 
#

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
yarn --cwd "$SCRIPT_DIR/../web/assets" install
cp "$SCRIPT_DIR/../web/assets/node_modules/katex/dist/katex.min.js" "$SCRIPT_DIR/../sudoapi/mdrenderer/knkatex/katex.min.js"

if [ $? -eq 0 ]; then
	echo "Copied katex.min.js to target directory"
else
	echo "Failed to copy katex.min.js to target directory"
fi
