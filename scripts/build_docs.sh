#!/bin/bash


SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
DOCS_DIR="/var/www/html/kndocs"

if ! command -v zensical >/dev/null 2>&1
then
    echo "installing zensical"
	pip install zensical
else
	echo "zensical installation found"
fi


cd "$SCRIPT_DIR/.." || exit
zensical build --clean

rm -rf "$DOCS_DIR"
mv ./docs_site "$DOCS_DIR"