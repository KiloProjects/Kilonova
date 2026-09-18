package web

import (
	"encoding/json"
	"io/fs"
	"log/slog"
)

// viteChunk is the subset of a Vite manifest entry that we care about.
type viteChunk struct {
	File string `json:"file"`
}

// viteAssets maps a source path, as spelled in web/assets (for example "app.ts"
// or "tailwind.css"), to the content-hashed file Vite emitted for it. One build
// is run per bundle, so there is one manifest per bundle to merge.
type viteAssets map[string]viteChunk

var assets = loadAssets()

func loadAssets() viteAssets {
	all := viteAssets{}
	manifests, err := fs.Glob(embedded, "static/manifest.*.json")
	if err != nil {
		slog.Error("Could not look for Vite manifests", slog.Any("err", err))
	}
	for _, name := range manifests {
		data, err := fs.ReadFile(embedded, name)
		if err != nil {
			slog.Error("Could not read Vite manifest", slog.String("name", name), slog.Any("err", err))
			continue
		}
		if err := json.Unmarshal(data, &all); err != nil {
			slog.Error("Could not parse Vite manifest", slog.String("name", name), slog.Any("err", err))
		}
	}
	if len(all) == 0 {
		slog.Error("No Vite manifest was embedded, assets will 404. Run `pnpm -C web/assets build` and rebuild.")
	}
	return all
}

// Asset returns the public URL of a file built by Vite, addressed by its source
// path. Everything it points at lives under /static/misc and is content-hashed,
// so it can be cached forever.
func (a viteAssets) Asset(src string) string {
	chunk, ok := a[src]
	if !ok {
		slog.Warn("Asset missing from the Vite manifest", slog.String("src", src))
		return "/static/" + src
	}
	return "/static/" + chunk.File
}
