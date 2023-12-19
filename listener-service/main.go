package main

import (
	"listener/event"
	"log"
	"math"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	//connect to rabbitmq
	rabbitConn, err := conntect()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer rabbitConn.Close()
	log.Println("RabbitMQ connection established")

	//start listening for messages

	//create consumer channel
	consumer, err := event.NewConsumer(rabbitConn)
	if err != nil {
		panic(err)
	}

	//watch the queue for messages and consume them
	err = consumer.Listen([]string{"log.INFO", "log.WARNING", "log.ERROR"})
	if err != nil {
		log.Println(err)
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
