package main

import (
	"errors"
	"log"
)

type RPCServer struct {
	Config *Config
}

type RPCPayload struct {
	Email    string
	Password string
}

// LogInfor RPC call
func (s *RPCServer) AuthenticateRPC(payload RPCPayload, response *string) error {
	log.Println("Authenticate RPC call: ", payload)
	if payload.Email == "" || payload.Password == "" {
		log.Println("Error: missing email or password")
		return errors.New("missing email or password")
	}

	if s.Config == nil {
		return errors.New("Config is nil")
	}
	if s.Config.Repo == nil {
		return errors.New("Repo is nil")
	}

	user, err := s.Config.Repo.GetByEmail(payload.Email)
	if err != nil {
		log.Println("Error getting email: ", err)
		return err
	}

	valid, err := s.Config.Repo.PasswordMatches(payload.Password, *user)
	if err != nil || !valid {
		log.Println("Error validating password: ", err)
		return err
	}

	// log the request
	err = s.Config.logRequest("authentication", payload.Email)
	if err != nil {
		log.Println("Error logging request: ", err)
		return err
	}

	return nil
}
