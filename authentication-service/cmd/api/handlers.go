package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/tsawler/toolbox"
)

func (app *Config) Authenticate(w http.ResponseWriter, r *http.Request) {
	var tools toolbox.Tools

	var requestPayload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	err := tools.ReadJSON(w, r, &requestPayload)
	if err != nil {
		tools.ErrorJSON(w, errors.New("Error in Json"), http.StatusBadRequest)
		return
	}

	user, err := app.Repo.GetByEmail(requestPayload.Email)
	if err != nil {
		tools.ErrorJSON(w, errors.New("No user with that e-mail"), http.StatusBadRequest)
		return
	}

	valid, err := app.Repo.PasswordMatches(requestPayload.Password, *user)
	if err != nil || !valid {
		tools.ErrorJSON(w, errors.New("Invalid credentials"), http.StatusBadRequest)
		return
	}

	// log the request
	err = app.logRequest("authentication", requestPayload.Email)
	if err != nil {
		tools.ErrorJSON(w, errors.New("Error logging request"))
		return
	}

	payload := toolbox.JSONResponse{
		Error:   false,
		Message: fmt.Sprintf("Logged in as %s", user.Email),
		Data:    user,
	}
	_ = tools.WriteJSON(w, http.StatusAccepted, payload)
}

func (app *Config) logRequest(name, data string) error {
	var entry struct {
		Name string `json:"name"`
		Data string `json:"data"`
	}
	entry.Name = name
	entry.Data = data
	jsonData, _ := json.MarshalIndent(entry, "", "\t")

	request, err := http.NewRequest("POST", "http://logger-service/log", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println(err)
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	_, err = app.Client.Do(request)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
