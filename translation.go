package kilonova

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	_ "embed"

	"github.com/KiloProjects/kilonova/internal/config"
)

type Translation map[string]string
type Translations map[string]Translation

var translations Translations

//go:generate go run ./scripts/toml_gen --target ./_translations.json --target ./web/assets/_translations.json

//go:embed _translations.json
var keys []byte

func TranslationKeyExists(line string) bool {
	_, ok := translations[line]
	return ok
}

func GetText(lang, line string, args ...any) string {
	if _, ok := translations[line]; !ok {
		slog.WarnContext(context.TODO(), "Invalid translation key", slog.Any("key", line))
		return line
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
			slog.ErrorContext(context.Background(), "Invalid translation JSON type")
			os.Exit(1)
		}
	}
}

func init() {
	translations = make(Translations)
	var elems = make(map[string]map[string]any)
	err := json.Unmarshal(keys, &elems)
	if err != nil {
		slog.ErrorContext(context.Background(), "Error unmarshaling translation keys", slog.Any("err", err))
		os.Exit(1)
	}
	for name, children := range elems {
		recurse(name, children)
	}
}
