package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/tsawler/toolbox"
)

func (app *Config) SendMail(w http.ResponseWriter, r *http.Request) {
	var tools toolbox.Tools

	type mailMessage struct {
		From    string `json:"from"`
		To      string `json:"to"`
		Subject string `json:"subject"`
		Message string `json:"message"`
	}

	var requestPayload mailMessage

	err := tools.ReadJSON(w, r, &requestPayload)
	if err != nil {
		log.Println("Error reading JSON: ", err)
		tools.ErrorJSON(w, errors.New("Error in JSON"))
		return
	}

	message := Message{
		From:    requestPayload.From,
		To:      requestPayload.To,
		Subject: requestPayload.Subject,
		Data:    requestPayload.Message,
	}

	err = app.Mailer.SendSMTPMessage(message)
	if err != nil {
		log.Println("Error sending email: ", err)
		tools.ErrorJSON(w, err)
		return
	}
	payload := toolbox.JSONResponse{
		Error:   false,
		Message: "Message sent to " + requestPayload.To,
	}

	tools.WriteJSON(w, http.StatusAccepted, payload)

}
