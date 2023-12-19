package main

import (
	"broker/event"
	"broker/logs"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
	"time"

	"github.com/tsawler/toolbox"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
	Log    LogPayload  `json:"log,omitempty"`
	Mail   MailPayload `json:"mail,omitempty"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type MailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	var tools toolbox.Tools

	payload := toolbox.JSONResponse{
		Error:   false,
		Message: "Hit the broker endpoint",
	}

	_ = tools.WriteJSON(w, http.StatusAccepted, payload)
}

func (app *Config) HandleSubmission(w http.ResponseWriter, r *http.Request) {
	var tools toolbox.Tools
	var requestPayload RequestPayload

	err := tools.ReadJSON(w, r, &requestPayload)
	if err != nil {
		tools.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	log.Println("Handle Submission, Request payload: ", requestPayload)

	switch requestPayload.Action {
	case "auth":
		app.authenticateViaRPC(w, requestPayload.Auth)
	case "log":
		app.logEventViaRPC(w, requestPayload.Log)
	case "mail":
		app.sendMail(w, requestPayload.Mail)
	default:
		tools.ErrorJSON(w, errors.New("Unknown action"), http.StatusBadRequest)
	}
}

func (app *Config) authenticate(w http.ResponseWriter, a AuthPayload) {
	var tools toolbox.Tools
	jsonData, _ := json.MarshalIndent(a, "", "\t")

	request, err := http.NewRequest("POST", "http://authentication-service/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		tools.ErrorJSON(w, err)
		return
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		tools.ErrorJSON(w, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		tools.ErrorJSON(w, errors.New("Invalid credentials"))
		return
	} else if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusOK {
		tools.ErrorJSON(w, errors.New(fmt.Sprintf("Error from authentication service: %d", response.StatusCode)))
		return
	}

	var jsonFromService toolbox.JSONResponse

	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		tools.ErrorJSON(w, err)
		return
	}

	if jsonFromService.Error {
		tools.ErrorJSON(w, err, http.StatusUnauthorized)
		return
	}

	var payload toolbox.JSONResponse
	payload.Error = false
	payload.Message = "Authenticated"
	payload.Data = jsonFromService.Data

	_ = tools.WriteJSON(w, http.StatusAccepted, payload)
}

func (app *Config) logItem(w http.ResponseWriter, entry LogPayload) {
	var tools toolbox.Tools
	jsonData, _ := json.MarshalIndent(entry, "", "\t")

	request, err := http.NewRequest("POST", "http://logger-service/log", bytes.NewBuffer(jsonData))
	if err != nil {
		tools.ErrorJSON(w, err)
		return
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println("Error from logger service: ", err)
		tools.ErrorJSON(w, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusOK {
		tools.ErrorJSON(w, errors.New(fmt.Sprintf("Error from logging service: %d", response.StatusCode)))
		return
	}

	var payload toolbox.JSONResponse
	payload.Error = false
	payload.Message = "Logged"

	_ = tools.WriteJSON(w, http.StatusAccepted, payload)
}

func (app *Config) sendMail(w http.ResponseWriter, m MailPayload) {
	var tools toolbox.Tools
	jsonData, _ := json.MarshalIndent(m, "", "\t")

	request, err := http.NewRequest("POST", "http://mail-service/send", bytes.NewBuffer(jsonData))
	if err != nil {
		tools.ErrorJSON(w, err)
		return
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println("Error from mail service: ", err)
		tools.ErrorJSON(w, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusOK {
		tools.ErrorJSON(w, errors.New(fmt.Sprintf("Error from mail service: %d", response.StatusCode)))
		return
	}

	var payload toolbox.JSONResponse
	payload.Error = false
	payload.Message = "Mail sent to " + m.To

	_ = tools.WriteJSON(w, http.StatusAccepted, payload)
}

func (app *Config) logEventViaRabbit(w http.ResponseWriter, l LogPayload) {
	var tools toolbox.Tools

	err := app.pushToQueue(l.Name, l.Data)
	if err != nil {
		tools.ErrorJSON(w, err)
		log.Println(err)
	}
	var payload toolbox.JSONResponse
	payload.Error = false
	payload.Message = "Logged via RabbitMQ"

	_ = tools.WriteJSON(w, http.StatusAccepted, payload)
}

func (app *Config) pushToQueue(name string, msg string) error {
	emitter, err := event.NewEventEmitter(app.Rabbit)
	if err != nil {
		return err
	}
	payload := LogPayload{
		Name: name,
		Data: msg,
	}

	j, _ := json.MarshalIndent(&payload, "", "\t")
	err = emitter.Push(string(j), "log.INFO")
	if err != nil {
		return err
	}
	return nil
}

type RPCLogPayload struct {
	Name string
	Data string
}

type RPCAuthPayload struct {
	Email    string
	Password string
}

func (app *Config) logEventViaRPC(w http.ResponseWriter, l LogPayload) {
	var tools toolbox.Tools
	client, err := rpc.Dial("tcp", "logger-service:5001")
	if err != nil {
		tools.ErrorJSON(w, err)
		log.Println(err)
	}

	rpcPayload := RPCLogPayload{
		Name: l.Name,
		Data: l.Data,
	}
	var response string

	err = client.Call("RPCServer.LogInfo", rpcPayload, &response)
	if err != nil {
		tools.ErrorJSON(w, err)
		log.Println(err)
	}

	var payload toolbox.JSONResponse
	payload.Error = false
	payload.Message = response

	_ = tools.WriteJSON(w, http.StatusAccepted, payload)
}

func (app *Config) authenticateViaRPC(w http.ResponseWriter, a AuthPayload) {
	var tools toolbox.Tools

	//log the request
	log.Println("Authenticate via RPC: ", a)

	client, err := rpc.Dial("tcp", "authentication-service:5001")
	if err != nil {
		tools.ErrorJSON(w, err)
		log.Println("Dial: ", err)
	}

	rpcPayload := RPCAuthPayload{
		Email:    a.Email,
		Password: a.Password,
	}
	var response string

	err = client.Call("RPCServer.AuthenticateRPC", rpcPayload, &response)
	if err != nil {
		tools.ErrorJSON(w, err)
		log.Println("Call: ", err)
	}

	var payload toolbox.JSONResponse
	payload.Error = false
	payload.Message = response

	_ = tools.WriteJSON(w, http.StatusAccepted, payload)
}

func (app *Config) logViaGRPC(w http.ResponseWriter, r *http.Request) {
	var tools toolbox.Tools

	var requestPayload RequestPayload
	err := tools.ReadJSON(w, r, &requestPayload)
	if err != nil {
		tools.ErrorJSON(w, err)
		return
	}

	conn, err := grpc.Dial("logger-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		tools.ErrorJSON(w, err)
	}
	defer conn.Close()

	c := logs.NewLogServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = c.WriteLog(ctx, &logs.LogRequest{
		LogEntry: &logs.Log{
			Name: requestPayload.Log.Name,
			Data: requestPayload.Log.Data,
		},
	})

	if err != nil {
		tools.ErrorJSON(w, err)
		log.Println(err)
	}

	var payload toolbox.JSONResponse
	payload.Error = false
	payload.Message = "logled via gRPC"

	_ = tools.WriteJSON(w, http.StatusAccepted, payload)
}
