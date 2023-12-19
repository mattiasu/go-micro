package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const mainPort = "8080"

type Config struct {
	Rabbit *amqp.Connection
}

func main() {
	//connect to rabbitmq
	rabbitConn, err := conntect()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer rabbitConn.Close()
	log.Println("RabbitMQ connection established")

	app := Config{
		Rabbit: rabbitConn,
	}

	log.Printf("Starting front end service on port %s\n", mainPort)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", mainPort),
		Handler: app.routes(),
	}

	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func conntect() (*amqp.Connection, error) {
	//connect to rabbitmq
	var counts int64
	var backoff = 1 * time.Second
	var connection *amqp.Connection

	for {
		var err error
		connection, err = amqp.Dial("amqp://rabbitmq:password@rabbitmq/")
		if err == nil {
			return connection, nil
		}

		if counts > 5 {
			return nil, err
		}
		counts++
		log.Println("RabbitMQ not available, waiting...")
		time.Sleep(backoff)
		backoff = time.Duration(math.Pow(float64(backoff), 2)) * time.Second
		continue
	}
}
