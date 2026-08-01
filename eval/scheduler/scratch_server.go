package scheduler

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/afero"
)

// ScratchHandler serves the grader's scratch dir as an HTTP data plane over the
// same TLS+token channel as the RPC control plane, replacing the old SFTP
// subsystem. It speaks PUT/GET/DELETE on /scratch/{id}, where {id} is a
// platform-minted UUID naming a file directly in the grader's scratch fs (the
// same fs the BoxManager reads). Parsing {id} as a UUID is the path-traversal
// guard — no slashes, no "..", nothing but a flat file name.
func ScratchHandler(fsys afero.Fs, reg *ClientRegistry) (string, http.Handler) {
	h := &scratchServer{fs: fsys}
	return "/scratch/", reg.authMiddleware(http.StripPrefix("/scratch/", h))
}

type scratchServer struct{ fs afero.Fs }

func (s *scratchServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path
	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, "bad scratch id", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		s.put(w, r, id)
	case http.MethodGet:
		s.get(w, id)
	case http.MethodDelete:
		s.remove(w, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *scratchServer) put(w http.ResponseWriter, r *http.Request, id string) {
	f, err := s.fs.OpenFile(id, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, cerr := io.Copy(f, r.Body)
	// Close before responding so the bytes are on disk before the platform's
	// SaveFile returns and any RunBox3 references the id.
	if err := f.Close(); err != nil && cerr == nil {
		cerr = err
	}
	if cerr != nil {
		s.fs.Remove(id)
		http.Error(w, cerr.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *scratchServer) get(w http.ResponseWriter, id string) {
	f, err := s.fs.Open(id)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, f)
}

func (s *scratchServer) remove(w http.ResponseWriter, id string) {
	if err := s.fs.RemoveAll(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
