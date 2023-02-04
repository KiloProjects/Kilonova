package kilonova

import (
	"encoding/json"
	"fmt"

	_ "embed"

	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

type Translation map[string]string
type Translations map[string]Translation

var translations Translations

//go:generate /bin/sh -c "echo $PWD && /usr/bin/python scripts/toml_gen.py"

//go:embed _translations.json
var keys []byte

func TranslationKeyExists(line string) bool {
	_, ok := translations[line]
	return ok
}

func GetText(lang, line string, args ...any) string {
	if _, ok := translations[line]; !ok {
		zap.S().Warnf("Invalid translation key %q", line)
		return "ERR"
	}
	if _, ok := translations[line][lang]; !ok {
		return translations[line][config.Common.DefaultLang]
	}
	return fmt.Sprintf(translations[line][lang], args...)
}

func recurse(prefix string, val map[string]any) {
	for name, val := range val {
		if str, ok := val.(string); ok {
			if _, ok = translations[prefix]; !ok {
				translations[prefix] = make(Translation)
			}
			translations[prefix][name] = str
		} else if deeper, ok := val.(map[string]any); ok {
			recurse(prefix+"."+name, deeper)
		} else {
			zap.S().Fatal("Invalid translation JSON type")
		}
	}
}

func init() {
	translations = make(Translations)
	var elems = make(map[string]map[string]any)
	err := json.Unmarshal(keys, &elems)
	if err != nil {
		zap.S().Fatalf("Error unmarshaling translation keys: %#v", err)
	}
	for name, children := range elems {
		recurse(name, children)
	}
}
