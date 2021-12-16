package web

import (
	_ "embed"
	"encoding/json"
)

//go:generate python toml_gen.py | tee _translations.json assets/_translations.json >/dev/null

type Translation map[string]string
type Translations map[string]Translation

//go:embed _translations.json
var keys []byte

var translations Translations

func init() {
	err := json.Unmarshal(keys, &translations)
	if err != nil {
		panic(err)
	}
}
