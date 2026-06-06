package httpresponse

import (
	"encoding/json"
	"net/http"
	"time"
)

type SetResponse struct {
	Status     string      `json:"status"`
	Data       interface{} `json:"data,omitempty"`
	AccessTime string      `json:"accessTime"`
}

func Response(w http.ResponseWriter, data interface{}, code int) {
	setResponse := SetResponse{
		Status:     http.StatusText(code),
		AccessTime: time.Now().Format("02-01-2006 15:04:05"),
		Data:       data}
	response, _ := json.Marshal(setResponse)
	w.Header().Set("Content-Type", "Application/json")
	w.WriteHeader(code)
	_, err := w.Write(response)
	if err != nil {
		return
	}
}
