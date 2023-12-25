#!/usr/bin/env bash

# Before running this, remember to install:
# ubuntu/debian: libcap-dev libsystemd-dev
# fedora/rhel: libcap-devel systemd-devel

VERSION_LIMIT=5.19
CURRENT_VERSION=$(uname -r | cut -d"." -f1,2)
if (( $(echo "$CURRENT_VERSION <= $VERSION_LIMIT" |bc -l) )); then
    echo "Kernel version must be at least $VERSION_LIMIT, please upgrade your kernel or use legacy installer instead (current kernel version: $CURRENT_VERSION)"
    exit 1
fi


if [ "$(id -u)" -ne "0" ] ; then
    echo "This script must be executed with root privileges."
    exit 1
fi

if [ ! -f "/sys/fs/cgroup/cgroup.controllers" ]; then
    echo "CGroup v2 doesn't seem to be enabled on this system, please use the legacy installer instead"
    exit 1
fi

echo "Cloning git repository"
git clone https://github.com/ioi/isolate.git /tmp/isolate
cd /tmp/isolate
git checkout cg2

echo "Installing isolate"
make install

echo "Setting up systemd service"
cp ./systemd/isolate.{service,slice} /etc/systemd/system

echo "Enabling systemd service"
systemctl daemon-reload
systemctl enable --now isolate # Start keeper daemon and auto-start on boot


echo "Done."
