package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/common"
)

func getFormInt(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	sid := r.FormValue(name)
	if sid == "" {
		errorData(w, fmt.Sprintf("Missing param %s", name), http.StatusBadRequest)
		return 0, false
	}
	id, err := strconv.ParseInt(sid, 10, 32)
	if err != nil {
		errorData(w, fmt.Sprintf("Param `%s` not int", name), http.StatusBadRequest)
		return 0, false
	}
	return int(id), true
}

func returnData(w http.ResponseWriter, retData interface{}) {
	statusData(w, "success", retData, 200)
}

func statusData(w http.ResponseWriter, status string, retData interface{}, statusCode int) {
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(common.RetData{
		Status: status,
		Data:   retData,
	})
	if err != nil {
		if err != nil {
			log.Printf("[ERROR] Couldn't send return data: %v", err)
		}
	}
}

func errorData(w http.ResponseWriter, retData interface{}, errCode int) {
	statusData(w, "error", retData, errCode)
}

func getContextValue(r *http.Request, name string) interface{} {
	return r.Context().Value(common.KNContextType(name))
}
