package main

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

	"github.com/oleg-top/go-orchestrator/serialization"
)

type Agent struct {
	ID      uuid.UUID
	Channel *amqp.Channel
}

func NewAgent(ch *amqp.Channel) *Agent {
	return &Agent{Channel: ch}
}

func (a *Agent) Registrate() error {
	type Response struct {
		ID string `json:"id"`
	}
	req, err := http.NewRequest("POST", "http://localhost:8080/agents", nil)
	if err != nil {
		log.Error("Error while registrating new agent: " + err.Error())
		return err
	}
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Error("Error while registrating new agent: " + err.Error())
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error("Error while registrating new agent: " + err.Error())
		return err
	}
	var id Response
	err = json.Unmarshal(body, &id)
	if err != nil {
		log.Error("Error while registrating new agent: " + err.Error())
		return err
	}
	validID, err := uuid.Parse(id.ID)
	if err != nil {
		log.Error("Error while registrating new agent: " + err.Error())
		return err
	}
	a.ID = validID
	log.Info("Successfully registrated new agent: " + a.ID.String())
	return nil
}

func (a *Agent) HandleMessages() {
	q, err := a.Channel.QueueDeclare("tasks_queue", false, false, false, false, nil)
	msgs, err := a.Channel.Consume(
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

func (a *Agent) ResolveTask(tm serialization.TaskMessage) (int, error) {
	return 0, nil
}

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

	agent := NewAgent(ch)
	agent.Registrate()
	agent.HandleMessages()
}
