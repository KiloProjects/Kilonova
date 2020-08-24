package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/KiloProjects/Kilonova/common"
)

func returnData(w http.ResponseWriter, status string, returnData interface{}) {
	statusData(w, status, returnData, 200)
}

func statusData(w http.ResponseWriter, status string, returnData interface{}, statusCode int) {
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(common.RetData{
		Status: status,
		Data:   returnData,
	})
	if err != nil {
		if err != nil {
			log.Printf("[ERROR] Couldn't send return data: %v", err)
		}
	}
}

func errorData(w http.ResponseWriter, returnData interface{}, errCode int) {
	statusData(w, "error", returnData, errCode)
}

func getContextValue(r *http.Request, name string) interface{} {
	return r.Context().Value(common.KNContextType(name))
}
