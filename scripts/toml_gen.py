#!/usr/bin/env python
import toml
import json

print(
    "DEPRECATED: If you're still running this, it means that something is not ok. It will be removed soon"
)

targets = [
    "./_translations.json",
    "web/assets/_translations.json",
    # "../kilomix/app/util/_translations.json",
    # "../sveltova/src/_translations.json"
]

with open("./translations.toml", "r") as f:
    vals = toml.load(f)

for target in targets:
    with open(target, "w") as f:
        json.dump(vals, f)
