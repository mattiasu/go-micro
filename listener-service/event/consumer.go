package event

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn     *amqp.Connection
	queuName string
}

func NewConsumer(conn *amqp.Connection) (Consumer, error) {
	consumer := Consumer{
		conn: conn,
	}
	err := consumer.setup()
	if err != nil {
		return Consumer{}, err
	}
	return consumer, nil
}

func (consumer *Consumer) setup() error {
	channel, err := consumer.conn.Channel()
	if err != nil {
		return err
	}
	return declareExchange(channel)
}

type Payload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (consumer *Consumer) Listen(topics []string) error {
	channel, err := consumer.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	queue, err := declareRandomQueue(channel)
	if err != nil {
		return err
	}

	for _, topic := range topics {
		err = bindQueueToTopic(channel, queue.Name, topic)
		if err != nil {
			return err
		}
	}

	msgs, err := channel.Consume(
		queue.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	forever := make(chan bool)
	go func() {
		for d := range msgs {
			var payload Payload
			err := json.Unmarshal(d.Body, &payload)
			if err != nil {
				log.Println(err)
				continue
			}
			go handlePayload(payload)

			log.Printf("Received message with routing key %s", d.RoutingKey)
			log.Printf("Received message with payload %+v", payload)
		}
	}()
	log.Printf("Listening for messages with routing key %s", topics)
	<-forever
	return nil
}

func handlePayload(payload Payload) {
	err := logEvent(payload)
	if err != nil {
		log.Println(err)
	}
}

func logEvent(entry Payload) error {
	jsonData, _ := json.MarshalIndent(entry, "", "\t")

	request, err := http.NewRequest("POST", "http://logger-service/log", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println("Error from logger service: ", err)
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusOK {
		return err
	}
	return nil
}
