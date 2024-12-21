package kilonova

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

func StatusData(w http.ResponseWriter, status string, retData any, statusCode int) {

	if err, ok := retData.(error); ok {
		retData = err.Error()

		var err2 *statusError
		if errors.As(err, &err2) {
			statusCode = err2.Code
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
		if strings.Contains(err.Error(), "broken pipe") {
			return
		}
		slog.ErrorContext(context.Background(), "Couldn't send return data", slog.Any("err", err))
	}
}
