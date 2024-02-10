#!/bin/sh

if [ "$(id -u)" -ne "0" ] ; then
    echo "This script must be executed with root privileges."
    exit 1
fi

ISOLATE_PATH="/usr/local/etc/isolate_bin"
ISOLATE_CONF_PATH="/usr/local/etc/isolate"

groupadd -f kn_sandbox

echo "Cloning git repository"
git clone https://github.com/ioi/isolate.git /tmp/isolate
cd /tmp/isolate

echo "Installing isolate"
make install

mv /usr/local/bin/isolate $ISOLATE_PATH

chgrp kn_sandbox "$ISOLATE_PATH"
chmod 6774 "$ISOLATE_PATH"

echo "Adding current user to kn_sandbox group"
usermod -a -G kn_sandbox "$USER"

echo "Done. You might need to restart your shell for group changes to occur."
