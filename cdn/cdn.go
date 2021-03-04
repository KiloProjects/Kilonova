package cdn

import (
	"errors"
	"io/fs"
	"log"
	"net/http"
	"path"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/logic"
)

type CDN struct {
	kn    *logic.Kilonova
	userv kilonova.UserService
	cdn   kilonova.CDNStore
}

func (s *CDN) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fpath := r.URL.RequestURI()
		if r.Method != "GET" {
			http.Error(w, http.StatusText(405), 405)
			return
		}
		file, mod, err := s.cdn.GetFile(fpath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				http.Error(w, http.StatusText(404), 404)
				return
			}
			log.Println("Could not get file in CDN:", err)
			http.Error(w, http.StatusText(500), 500)
			return
		}
		w.Header().Add("cache-control", "public, max-age=86400, immutable")
		http.ServeContent(w, r, path.Base(fpath), mod, file)
	})
}

func New(kn *logic.Kilonova, db kilonova.TypeServicer) *CDN {
	return &CDN{kn: kn, userv: db.UserService(), cdn: kn.DM}
}
