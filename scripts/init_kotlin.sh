#!/usr/bin/env bash

# Inspired by https://github.com/goncalossilva/rpm-kotlin/blob/main/kotlin.spec

if [ "$(id -u)" -ne "0" ] ; then
    echo "This script must be executed with root privileges."
    exit 1
fi

set -e

KOTLIN_VERSION="1.9.24"

rm -rf /tmp/kotlin
mkdir /tmp/kotlin
cd /tmp/kotlin
echo "Downloading archive"
wget "https://github.com/JetBrains/kotlin/releases/download/v$KOTLIN_VERSION/kotlin-compiler-$KOTLIN_VERSION.zip" -O ./kotlin.zip

unzip kotlin.zip && cd kotlinc
sed -i "s|\(DIR *= *\).*|\1/usr/bin|" bin/*
sed -i "s|\(KOTLIN_HOME *= *\).*|\1/usr/share/kotlin|" bin/*

install -m 0755 bin/kotlin /usr/bin/
install -m 0755 bin/kotlin-dce-js /usr/bin/
install -m 0755 bin/kotlinc /usr/bin/
install -m 0755 bin/kotlinc-js /usr/bin/
install -m 0755 bin/kotlinc-jvm /usr/bin/

mkdir -p /usr/share/kotlin/
install -m 0644 build.txt /usr/share/kotlin/
mkdir -p /usr/share/kotlin/lib/
install -m 0644 lib/* /usr/share/kotlin/lib/

mkdir -p /usr/share/kotlin

