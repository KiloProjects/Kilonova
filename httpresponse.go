package kilonova

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

func StatusData(w http.ResponseWriter, status string, retData any, statusCode int) {

	if err, ok := retData.(error); ok {
		retData = err.Error()

		var sErr *statusError
		if errors.As(err, &sErr) {
			statusCode = sErr.Code
		}
	}
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(struct {
		Status string `json:"status"`
		Data   any    `json:"data"`
	}{
		Status: status,
		Data:   retData,
	})
	if err != nil {
		slog.ErrorContext(context.Background(), "Couldn't send return data", slog.Any("err", err))
	}
}
