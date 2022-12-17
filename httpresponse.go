package kilonova

import (
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

func StatusData(w http.ResponseWriter, status string, retData any, statusCode int) {
	if err, ok := retData.(*StatusError); ok {
		// zap.S().Warn("*StatusError passed to sudoapi.statusData. This might not be intended")
		err.WriteError(w)
		return
	}
	if err, ok := retData.(error); ok {
		retData = err.Error()
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
		zap.S().WithOptions(zap.AddCallerSkip(1)).Errorf("Couldn't send return data: %v", err)
	}
}
