package main

import (
	"context"
	"fmt"
	"log"
	"log-service/data"
	"log-service/logs"
	"net"

	"google.golang.org/grpc"
)

type LogServer struct {
	logs.UnimplementedLogServiceServer
	Model data.Models
}

func (l *LogServer) WriteLog(ctx context.Context, req *logs.LogRequest) (*logs.LogResponse, error) {
	input := req.GetLogEntry()

	logEntry := data.LogEntry{
		Name: input.GetName(),
		Data: input.GetData(),
	}

	err := l.Model.LogEntry.Insert(logEntry)
	if err != nil {
		res := &logs.LogResponse{Result: "Failed"}
		return res, err
	}
	res := &logs.LogResponse{Result: "Success"}
	return res, nil
}

func (app *Config) gRPCListen() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen for gRPC: %v", err)
	}

	srv := grpc.NewServer()
	logs.RegisterLogServiceServer(srv, &LogServer{Model: app.Models})

	log.Printf("Starting gRPC server on port %v", grpcPort)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("failed to serve gRPC: %v", err)
	}
}
