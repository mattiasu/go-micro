package main

import (
	"log"
	"log-service/data"
	"net/http"

	"github.com/tsawler/toolbox"
)

type JSONPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) WriteLog(w http.ResponseWriter, r *http.Request) {
	var tools toolbox.Tools

	var requestPayload JSONPayload
	_ = tools.ReadJSON(w, r, &requestPayload)

	event := data.LogEntry{
		Name: requestPayload.Name,
		Data: requestPayload.Data,
	}

	err := app.Models.LogEntry.Insert(event)
	if err != nil {
		log.Println("Error inserting log entry: ", err)
		tools.ErrorJSON(w, err)
		return
	}

	resp := toolbox.JSONResponse{
		Error:   false,
		Message: "Log entry written",
	}

	_ = tools.WriteJSON(w, http.StatusAccepted, resp)

}
