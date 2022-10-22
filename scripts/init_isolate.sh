#!/bin/sh

if [ "$(id -u)" -ne "0" ] ; then
    echo "This script must be executed with root privileges."
    exit 1
fi

ISOLATE_PATH="/usr/local/etc/isolate_bin"
ISOLATE_CONF_PATH="/usr/local/etc/isolate"

groupadd -f kn_sandbox

echo "Downloading isolate binary"
curl -sSL 'https://github.com/KiloProjects/isolate/releases/latest/download/isolate' -o "$ISOLATE_PATH"

chmod ug+x "$ISOLATE_PATH"
chgrp kn_sandbox "$ISOLATE_PATH"
chmod +s "$ISOLATE_PATH"

echo "Downloading isolate config"
curl -sSL 'https://github.com/KiloProjects/isolate/releases/latest/download/default.cf' -o "$ISOLATE_CONF_PATH"

echo "Done. Don't forget to add your user account to the `kn_sandbox` group"
