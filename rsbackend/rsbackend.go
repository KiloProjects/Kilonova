package rsbackend

import "net/http"

type BackendInteractor struct {
	cl *http.Client
}

func New() *BackendInteractor {
	cl := &http.Client{}
	return &BackendInteractor{cl}
}
