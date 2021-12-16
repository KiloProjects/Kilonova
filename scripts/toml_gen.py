#!/usr/bin/env python
import toml
import json

with open('web/translations.toml', 'r') as f:
    vals = toml.load(f)
with open('web/_translations.json', 'w') as f:
    json.dump(vals, f)
with open('web/assets/_translations.json', 'w') as f:
    json.dump(vals, f)
