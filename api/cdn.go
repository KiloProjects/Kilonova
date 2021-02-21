package api

import (
	"errors"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (s *API) deleteCDNObject(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fpath := path.Clean(r.FormValue("path"))
	if fpath == "" {
		errorData(w, "No path specified", 400)
		return
	}
	if !util.User(r).Admin || r.FormValue("path") == "." {
		fpath = path.Join(strconv.Itoa(util.User(r).ID), fpath)
	}
	if err := s.manager.DeleteObject(fpath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			errorData(w, "File doesn't exist", 400)
			return
		}
		if errors.Is(err, kilonova.ErrNotEmpty) {
			errorData(w, "Folder not empty", 400)
			return
		}
		log.Println("Unknown error while deleting object:", err)
		errorData(w, err, 500)
	}
	returnData(w, "Deleted")
}

func (s *API) saveCDNFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(100 << 20); err != nil { // 100MB
		errorData(w, "Invalid request", 400)
		return
	}
	fpath := path.Clean(r.FormValue("path"))
	if fpath == "" {
		errorData(w, "No path specified", 400)
		return
	}
	f, fh, err := r.FormFile("file")
	if err != nil {
		errorData(w, "No file sent", 400)
		return
	}
	fpath = path.Join(fpath, fh.Filename)
	if !util.User(r).Admin || r.FormValue("path") == "." {
		fpath = path.Join(strconv.Itoa(util.User(r).ID), fpath)
	}
	defer f.Close()
	if err := s.manager.SaveFile(fpath, f); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "File saved")

}

func (s *API) createCDNDir(w http.ResponseWriter, r *http.Request) {
	dpath := path.Clean(r.FormValue("path"))
	if !util.User(r).Admin || r.FormValue("path") == "." {
		dpath = path.Join(strconv.Itoa(util.User(r).ID), dpath)
	}
	if err := s.manager.CreateDir(dpath); err != nil {
		errorData(w, err, 400)
		return
	}
	returnData(w, dpath)
}

func (s *API) readCDNDirectory(w http.ResponseWriter, r *http.Request) {
	dpath := path.Clean(r.FormValue("path"))
	if !util.User(r).Admin || r.FormValue("path") == "." {
		dpath = path.Join(strconv.Itoa(util.User(r).ID), dpath)
	}
	if err := s.manager.CreateDir(dpath); err != nil {
		errorData(w, err, 400)
		return
	}
	dir, err := s.manager.ReadDir(dpath)
	if err != nil {
		if errors.Is(err, kilonova.ErrNotDirectory) {
			errorData(w, err, 400)
			return
		}
		log.Println("Could not read CDN dir:", err)
		errorData(w, err, 500)
		return
	}
	var out struct {
		CanReadFirst bool                   `json:"can_read_first"`
		Paths        []string               `json:"path"`
		Dirs         []kilonova.CDNDirEntry `json:"dirs"`
	}
	out.CanReadFirst = util.User(r).Admin
	out.Paths = strings.Split(dpath, "/")
	if out.Paths[0] == "." {
		out.Paths = out.Paths[1:]
	}
	out.Dirs = dir
	returnData(w, out)
}
