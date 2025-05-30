#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
pip install mkdocs mkdocs-material

cd "$SCRIPT_DIR/.."
mkdocs build --clean --site--dir /var/www/html/kndocs