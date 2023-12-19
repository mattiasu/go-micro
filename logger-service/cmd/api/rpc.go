package main

import (
	"context"
	"log"
	"log-service/data"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RPCServer struct{}

type RPCPayload struct {
	Name string
	Data string
}

// LogInfor RPC call
func (s *RPCServer) LogInfo(payload RPCPayload, response *string) error {
	collection := client.Database("logs").Collection("logs")
	_, err := collection.InsertOne(context.TODO(), data.LogEntry{
		ID:        primitive.NewObjectID().Hex(),
		Name:      payload.Name,
		Data:      payload.Data,
		CreatedAt: time.Now(),
	})
	if err != nil {
		log.Println("Error inserting log: ", err)
		return err
	}
	*response = "Processed payload via RPC: " + payload.Name
	return nil
}
