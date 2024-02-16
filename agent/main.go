package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

	"github.com/oleg-top/go-orchestrator/serialization"
)

func main() {
	conn, err := amqp.Dial("amqp://defaultuser:defaultpass@localhost:5672/")
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ: ", err)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("Failed to open a channel")
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("tasks_queue", false, false, false, false, nil)
	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to consume message")
		return
	}

	var forever chan struct{}

	go func() {
		for d := range msgs {
			tm, err := serialization.Deserialize[serialization.TaskMessage](d.Body)
			if err != nil {
				log.Error(err)
			} else {
				log.Info(tm.String())
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
