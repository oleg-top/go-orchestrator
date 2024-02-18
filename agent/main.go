package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

	"github.com/oleg-top/go-orchestrator/db/storage"
	"github.com/oleg-top/go-orchestrator/rpn"
	"github.com/oleg-top/go-orchestrator/serialization"
)

type Agent struct {
	ID      uuid.UUID
	Channel *amqp.Channel
	results map[string]string
	mu      sync.Mutex
	wg      sync.WaitGroup
}

func NewAgent(ch *amqp.Channel) *Agent {
	return &Agent{Channel: ch, results: make(map[string]string)}
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
	go a.SendHeartbeat(15 * time.Second)
	return nil
}

func (a *Agent) HandleMessages() {
	tasks_queue, _ := a.Channel.QueueDeclare("tasks_queue", false, false, false, false, nil)
	result_queue, _ := a.Channel.QueueDeclare("result_queue", false, false, false, false, nil)
	msgs, err := a.Channel.Consume(
		tasks_queue.Name,
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
			a.publishCalculatingStatus(tm.ID)
			if err != nil {
				log.Error(err)
			} else {
				log.Info("Got message: " + tm.String())
				res, err := a.ResolveTask(tm)
				var status string
				if err != nil {
					status = storage.StatusTaskInvalid
					log.Error(err)
				} else {
					status = storage.StatusTaskCompleted
				}
				rm := serialization.ResultMessage{
					ID:     tm.ID,
					Result: res,
					Status: status,
				}
				serialized, err := serialization.Serialize[serialization.ResultMessage](rm)
				if err != nil {
					log.Error("Error while serializing task message")
				}
				err = a.Channel.Publish(
					"",
					result_queue.Name,
					false,
					false,
					amqp.Publishing{ContentType: "application/json", Body: serialized},
				)
				if err != nil {
					log.Error("Error while publishing task message")
				} else {
					log.Info("Published result message: " + tm.ID.String())
				}
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func (a *Agent) ResolveTask(tm serialization.TaskMessage) (string, error) {
	r, err := rpn.NewRPN(tm.Expression)
	if err != nil {
		return "", err
	}
	tokens := strings.Fields(r.RPNExpression)
	for len(tokens) != 1 {
		for i := 2; i < len(tokens); i++ {
			if (tokens[i] == "-" || tokens[i] == "+" || tokens[i] == "*" ||
				tokens[i] == "/") && rpn.IsNumeric(tokens[i-1]) && rpn.IsNumeric(tokens[i-2]) {
				exp := strings.Join(tokens[i-2:i+1], " ")
				a.wg.Add(1)
				var timeout time.Duration
				switch tokens[i] {
				case "+":
					timeout = tm.Timeouts["add"]
				case "-":
					timeout = tm.Timeouts["sub"]
				case "*":
					timeout = tm.Timeouts["mul"]
				case "/":
					timeout = tm.Timeouts["div"]
				}
				go a.calculateOperation(exp, timeout)
			}
		}
		a.wg.Wait()
		s := strings.Join(tokens, " ")
		for key, val := range a.results {
			s = strings.Replace(s, key, val, 1)
		}
		tokens = strings.Fields(s)
	}
	log.Info(tm.Expression + " -> " + tokens[0])
	return tokens[0], nil
}

func (a *Agent) publishCalculatingStatus(taskID uuid.UUID) error {
	status_queue, _ := a.Channel.QueueDeclare("status_queue", false, false, false, false, nil)
	cm := serialization.CalculatingMessage{
		AgentID: a.ID,
		TaskID:  taskID,
	}
	serialized, err := serialization.Serialize[serialization.CalculatingMessage](cm)
	if err != nil {
		log.Error("Error while serializing task message: " + err.Error())
	}
	err = a.Channel.Publish(
		"",
		status_queue.Name,
		false,
		false,
		amqp.Publishing{ContentType: "application/json", Body: serialized},
	)
	if err != nil {
		log.Error("Error while publishing status message: " + err.Error())
	}
	return nil
}

func (a *Agent) calculateOperation(exp string, timeout time.Duration) {
	var res int
	tokens := strings.Fields(exp)
	first, _ := strconv.Atoi(tokens[0])
	second, _ := strconv.Atoi(tokens[1])
	operation := tokens[2]
	switch operation {
	case "+":
		res = first + second
	case "-":
		res = first - second
	case "*":
		res = first * second
	case "/":
		res = first / second
	}
	time.Sleep(timeout)
	a.mu.Lock()
	a.results[exp] = strconv.Itoa(res)
	a.mu.Unlock()
	log.Info("goroutine: " + exp + "; result: " + strconv.Itoa(res))
	a.wg.Done()
}

func (a *Agent) SendHeartbeat(duration time.Duration) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			req, err := http.NewRequest(
				"POST",
				fmt.Sprintf("http://localhost:8080/agents/%s/ping", a.ID.String()),
				nil,
			)
			if err != nil {
				log.Error("Error while creating hearbeat ping request: " + err.Error())
			}
			client := &http.Client{}
			_, err = client.Do(req)
			if err != nil {
				log.Error("Error while sending hearbeat ping: " + err.Error())
			} else {
				log.Info("Successfully sent hearbeat ping: " + a.ID.String())
			}
		}
	}
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
