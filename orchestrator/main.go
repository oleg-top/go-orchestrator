package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

	"github.com/oleg-top/go-orchestrator/db/storage"
	"github.com/oleg-top/go-orchestrator/serialization"
)

// Структура оркестратора
type Orchestrator struct {
	Storage  *storage.Storage
	Channel  *amqp.Channel
	Router   *mux.Router
	Timeouts map[string]time.Duration
}

// Функция создания нового экземпляра оркестратора
func NewOrchestrator(db *sqlx.DB, ch *amqp.Channel) *Orchestrator {
	orchestrator := &Orchestrator{
		Storage: storage.NewStorage(db),
		Channel: ch,
		Router:  mux.NewRouter(),
	}
	orchestrator.SetupRoutes()

	return orchestrator
}

// Настройка всех эндпоинтов
func (o *Orchestrator) SetupRoutes() {
	o.Router.HandleFunc("/agents", o.AddAgent).Methods("POST")
	o.Router.HandleFunc("/agents", o.GetAllAgents).Methods("GET")
	o.Router.HandleFunc("/agents/{id}/ping", o.AgentPing).Methods("POST")
	o.Router.HandleFunc("/expressions", o.AddExpression).Methods("POST")
	o.Router.HandleFunc("/expressions", o.GetAllExpressions).Methods("GET")
	o.Router.HandleFunc("/expressions/{id}", o.GetExpressionById).Methods("GET")
	o.Router.HandleFunc("/timeouts", o.SetTimeouts).Methods("POST")
	o.Router.HandleFunc("/timeouts", o.GetTimeouts).Methods("GET")
}

// Регистрация агента
func (o *Orchestrator) AddAgent(w http.ResponseWriter, r *http.Request) {
	id, err := o.Storage.AddAgent()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info("Adding agent: " + id.String())
	// o.LastPingTimestamp[id] = time.Now()
	err = o.Storage.UpdateAgentLastOnline(id, time.Now())
	if err != nil {
		log.Error("Error while updating agent's last online: " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	if err != nil {
		log.Error("Error while encoding json: " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Возвращение списка всех агентов
func (o *Orchestrator) GetAllAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := o.Storage.GetAllAgents()
	if err != nil {
		log.Error("Error while selecting all agents: " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(&agents)
	if err != nil {
		log.Error("Error while encoding json: " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		log.Info("Successfully returned all agents slice")
	}
}

// Обработка пингов агентов
func (o *Orchestrator) AgentPing(w http.ResponseWriter, r *http.Request) {
	agentIDStr := mux.Vars(r)["id"]
	log.Info("Got ping from: " + agentIDStr)
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		log.Error("Error while parsing uuid: " + err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// o.LastPingTimestamp[agentID] = time.Now()

	err = o.Storage.UpdateAgentStatus(agentID, storage.StatusAgentActive)
	if err != nil {
		log.Error("Error while updating agents: " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = o.Storage.UpdateAgentLastOnline(agentID, time.Now())
	if err != nil {
		log.Error("Error while updating agents: " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		log.Info("Successfully updated agent: " + agentIDStr)
	}

	w.WriteHeader(http.StatusOK)
}

// Добавление выражения
func (o *Orchestrator) AddExpression(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		Expression string `json:"expression"`
	}

	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Error("Error while parsing request body: " + err.Error())
		return
	}
	taskID, err := o.Storage.AddTask(request.Expression)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error("Error while inserting expression to db: " + err.Error())
		return
	}
	err = json.NewEncoder(w).Encode(map[string]string{"id": taskID.String()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error("Error while encoding json: " + err.Error())
		return
	}
	q, err := o.Channel.QueueDeclare("tasks_queue", false, false, false, false, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error("Error while encoding json: " + err.Error())
		return
	}
	tm := serialization.TaskMessage{
		ID:         taskID,
		Expression: request.Expression,
		Timeouts:   o.Timeouts,
	}
	serialized, err := serialization.Serialize[serialization.TaskMessage](tm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error("Error while serializing task message: " + err.Error())
		return
	}
	err = o.Channel.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{ContentType: "application/json", Body: serialized},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error("Error while publishing task message: " + err.Error())
		return
	}
	log.Info("Successfully published task message")
}

// Получение выражения по ID
func (o *Orchestrator) GetExpressionById(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	validID, err := uuid.Parse(id)
	if err != nil {
		log.Error("Error while parsing id: " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tasks, err := o.Storage.GetTaskById(validID)
	if err != nil {
		log.Error("Error while getting expression by id: " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(&tasks)
	if err != nil {
		log.Error("Error while encoding task: " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Получение всех выражений
func (o *Orchestrator) GetAllExpressions(w http.ResponseWriter, r *http.Request) {
	tasks, err := o.Storage.GetAllTasks()
	if err != nil {
		log.Error("Error while selecting all tasks: " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		log.Info("Successfully selected all from tasks")
	}

	err = json.NewEncoder(w).Encode(&tasks)
	if err != nil {
		log.Error("Error while selecting all expressions: " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		log.Info("Successfully returned all tasks")
	}
}

// Горутина, которая мониторит состояние всех серверов
func (o *Orchestrator) StartHeartbeatCheck(duration time.Duration) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	var agents []storage.Agent
	var err error

	for {
		select {
		case <-ticker.C:
			agents, err = o.Storage.GetAllAgents()
			if err != nil {
				log.Error("Error while getting all agents for heartbeat: " + err.Error())
			}
			currentTime := time.Now()
			for _, agent := range agents {
				lastPingTime, err := time.Parse(time.RFC1123Z, agent.LastOnline)
				if err != nil {
					log.Error("Error while parsing agent's lastOnline: " + err.Error())
				}
				log.Info(
					"ID: " + agent.ID.String() + "; sub: " + currentTime.Sub(lastPingTime).
						String(),
				)
				if currentTime.Sub(lastPingTime) > duration {
					log.Info("Agent is inactive: ", agent.ID.String())
					err := o.Storage.UpdateAgentStatus(agent.ID, storage.StatusAgentInactive)
					if err != nil {
						log.Error("Error while updating agents table: " + err.Error())
					} else {
						log.Info("Successfully updated agents table")
					}
				}
			}
		}
	}
}

// Горутина, которая проверяет, работают ли агенты, на которых считаются выражения
func (o *Orchestrator) StartTaskStatusCheck(duration time.Duration) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	var tasks []storage.Task
	var agents []storage.Agent
	var agent storage.Agent
	var err error

	for {
		select {
		case <-ticker.C:
			tasks, err = o.Storage.GetAllTasks()
			if err != nil {
				log.Error("Error while getting all tasks for status check: " + err.Error())
			}
			for _, task := range tasks {
				if task.Status == storage.StatusTaskCalculating {
					log.Info("Found calculating task")
					agents, err = o.Storage.GetAgentById(task.AgentID)
					if err != nil {
						log.Error("Error while getting agent by id: " + err.Error())
					}
					agent = agents[0]
					if agent.Status == storage.StatusAgentInactive {
						log.Info("Republishing task: " + task.ID.String())
						q, err := o.Channel.QueueDeclare(
							"tasks_queue",
							false,
							false,
							false,
							false,
							nil,
						)
						if err != nil {
							log.Error("Error while encoding json: " + err.Error())
						}
						tm := serialization.TaskMessage{
							ID:         task.ID,
							Expression: task.Expression,
							Timeouts:   o.Timeouts,
						}
						serialized, err := serialization.Serialize[serialization.TaskMessage](tm)
						if err != nil {
							log.Error("Error while serializing task message: " + err.Error())
						}
						err = o.Channel.Publish(
							"",
							q.Name,
							false,
							false,
							amqp.Publishing{ContentType: "application/json", Body: serialized},
						)
						if err != nil {
							log.Error("Error while publishing task message: " + err.Error())
						}
						err = o.Storage.UpdateTaskStatus(task.ID, storage.StatusTaskRepublished)
						if err != nil {
							log.Error("Error while updating task status: " + err.Error())
						}
						log.Info("Successfully republished task: " + task.ID.String())
					}
				}
			}
		}
	}
}

// Горутина, которая записывает в бд выражению агента, который его считает
func (o *Orchestrator) HandleCalculatingStatuses() {
	status_queue, _ := o.Channel.QueueDeclare("status_queue", false, false, false, false, nil)
	msgs, err := o.Channel.Consume(
		status_queue.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Error("Failed to consume message: " + err.Error())
	}

	var forever chan struct{}

	go func() {
		for d := range msgs {
			cm, err := serialization.Deserialize[serialization.CalculatingMessage](d.Body)
			if err != nil {
				log.Error("Error while deserializing cm: " + err.Error())
			} else {
				err = o.Storage.UpdateTaskAgentID(cm.TaskID, cm.AgentID)
				if err != nil {
					log.Error("Error while updating task: " + err.Error())
				}
				err = o.Storage.UpdateTaskStatus(cm.TaskID, storage.StatusTaskCalculating)
				if err != nil {
					log.Error("Error while updating task: " + err.Error())
				}
			}
		}
	}()

	<-forever
}

// Горутина, которая принимает все результаты выражений
func (o *Orchestrator) HandleResults() {
	result_queue, _ := o.Channel.QueueDeclare("result_queue", false, false, false, false, nil)
	msgs, err := o.Channel.Consume(
		result_queue.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Error("Failed to consume message: " + err.Error())
	}

	var forever chan struct{}

	go func() {
		for d := range msgs {
			rm, err := serialization.Deserialize[serialization.ResultMessage](d.Body)
			if err != nil {
				log.Error(err)
			} else {
				log.Info("Got message: " + rm.String())
				err := o.Storage.UpdateTaskStatus(rm.ID, rm.Status)
				if err != nil {
					log.Error("Error while updating task: " + err.Error())
				} else {
					log.Info("Successfully updated task: " + rm.ID.String())
				}
				err = o.Storage.UpdateTaskResult(rm.ID, rm.Result)
				if err != nil {
					log.Error("Error while updating task: " + err.Error())
				} else {
					log.Info("Successfully updated task: " + rm.ID.String())
				}
			}
		}
	}()

	<-forever
}

// Настраивает время выполнения каждой операции
func (o *Orchestrator) SetTimeouts(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		Add int `json:"add"`
		Sub int `json:"sub"`
		Mul int `json:"mul"`
		Div int `json:"div"`
	}
	var request Request

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	o.Timeouts["add"] = time.Millisecond * time.Duration(request.Add)
	o.Timeouts["sub"] = time.Millisecond * time.Duration(request.Sub)
	o.Timeouts["mul"] = time.Millisecond * time.Duration(request.Mul)
	o.Timeouts["div"] = time.Millisecond * time.Duration(request.Div)

	w.WriteHeader(http.StatusOK)
}

// Получает все времени выполнения операций
func (o *Orchestrator) GetTimeouts(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"add": o.Timeouts["add"].String(),
		"sub": o.Timeouts["sub"].String(),
		"mul": o.Timeouts["mul"].String(),
		"div": o.Timeouts["div"].String(),
	})
}

// Запускает оркестратор
func (o *Orchestrator) StartHTTPServer(heartbeatDuration, statusCheckDuration time.Duration) {
	log.Info("Starting HTTP server...")
	go o.StartHeartbeatCheck(heartbeatDuration)
	go o.HandleResults()
	go o.HandleCalculatingStatuses()
	go o.StartTaskStatusCheck(statusCheckDuration)
	http.ListenAndServe(":8080", o.Router)
}

var schema = `
CREATE TABLE IF NOT EXISTS tasks (
	id VARCHAR(128) PRIMARY KEY,
	expression VARCHAR(128),
  result VARCHAR(128),
	status VARCHAR(128),
  agent_id VARCHAR(128)
);

CREATE TABLE IF NOT EXISTS agents (
	id VARCHAR(128) PRIMARY KEY,
	status VARCHAR(128),
  last_online VARCHAR(128)
);
`

// Настраивает коннекты и запускает все
func main() {
	db, err := sqlx.Connect("sqlite3", "db/database.db")
	if err != nil {
		log.Fatal("Failed to connect to sqlite3: ", err)
		return
	}
	defer db.Close()

	db.MustExec(schema)

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

	orchestrator := NewOrchestrator(db, ch)
	orchestrator.Timeouts = map[string]time.Duration{
		"add": 30000 * time.Millisecond,
		"sub": 2000 * time.Millisecond,
		"mul": 1000 * time.Millisecond,
		"div": 5000 * time.Millisecond,
	}
	if err != nil {
		log.Fatal(err)
		return
	}

	orchestrator.StartHTTPServer(30*time.Second, 3*time.Second)
}
