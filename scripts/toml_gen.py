#!/usr/bin/env python
import toml
import json

print("Updating...")

targets = [
    "web/_translations.json",
    "web/assets/_translations.json",
    # "../kilomix/app/util/_translations.json",
    # "../sveltova/src/_translations.json"
]

with open("web/translations.toml", "r") as f:
    vals = toml.load(f)

for target in targets:
    with open(target, "w") as f:
        json.dump(vals, f)
