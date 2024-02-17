package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

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
				log.Info("Got message: " + tm.String())
				_, err := a.ResolveTask(tm)
				if err != nil {
					log.Error(err)
				}
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func (a *Agent) ResolveTask(tm serialization.TaskMessage) (int, error) {
	r, err := rpn.NewRPN(tm.Expression)
	if err != nil {
		return 0, err
	}
	tokens := strings.Fields(r.RPNExpression)
	log.Info(tokens)
	for len(tokens) != 1 {
		for i := 2; i < len(tokens); i++ {
			log.Info(i)
			if (tokens[i] == "-" || tokens[i] == "+" || tokens[i] == "*" ||
				tokens[i] == "/") && rpn.IsNumeric(tokens[i-1]) && rpn.IsNumeric(tokens[i-2]) {
				exp := strings.Join(tokens[i-2:i+1], " ")
				a.wg.Add(1)
				go a.calculateOperation(exp, tm.Timeouts[tokens[i]])
			}
		}
		a.wg.Wait()
		s := strings.Join(tokens, " ")
		log.Info(a.results)
		for key, val := range a.results {
			s = strings.Replace(s, key, val, 1)
		}
		tokens = strings.Fields(s)
	}
	res, _ := strconv.Atoi(tokens[0])
	log.Info(tm.Expression + " -> " + tokens[0])
	return res, nil
}

func (a *Agent) calculateOperation(exp string, timeout time.Duration) {
	var res int
	tokens := strings.Fields(exp)
	log.Info(tokens)
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
