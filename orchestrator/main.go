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
)

type Orchestrator struct {
	Storage           *storage.Storage
	Channel           *amqp.Channel
	Router            *mux.Router
	Timings           map[string]time.Duration
	LastPingTimestamp map[uuid.UUID]time.Time
}

type Agent struct {
	ID     uuid.UUID `db:"id"`
	Status string    `db:"status"`
}

type Task struct {
	ID         uuid.UUID `db:"id"`
	Expression string    `db:"expression"`
	Status     string    `db:"status"`
}

type Expression struct {
	expression string
}

func NewOrchestrator(db *sqlx.DB, ch *amqp.Channel) *Orchestrator {
	orchestrator := &Orchestrator{
		Storage:           storage.NewStorage(db),
		Channel:           ch,
		Router:            mux.NewRouter(),
		LastPingTimestamp: make(map[uuid.UUID]time.Time),
	}
	orchestrator.SetupRoutes()

	return orchestrator
}

func (o *Orchestrator) SetupRoutes() {
	o.Router.HandleFunc("/agents", o.AddAgent).Methods("POST")
	o.Router.HandleFunc("/agents", o.GetAllAgents).Methods("GET")
	o.Router.HandleFunc("/agents/{id}/ping", o.AgentPing).Methods("POST")
	o.Router.HandleFunc("/expressions", o.AddExpression).Methods("POST")
	o.Router.HandleFunc("/expressions", o.GetAllExpressions).Methods("GET")
	o.Router.HandleFunc("/timeouts", o.SetTimeouts).Methods("POST")
	o.Router.HandleFunc("/timeouts", o.GetTimeouts).Methods("GET")
}

func (o *Orchestrator) AddAgent(w http.ResponseWriter, r *http.Request) {
	id, err := o.Storage.AddAgent()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	o.LastPingTimestamp[id] = time.Now()
	json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
}

func (o *Orchestrator) GetAllAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := o.Storage.GetAllAgents()
	if err != nil {
		log.Error("Error while selecting all agents")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(&agents)
	if err != nil {
		log.Error("Error while encoding json")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		log.Info("Successfully returned all agents slice")
	}
}

func (o *Orchestrator) AgentPing(w http.ResponseWriter, r *http.Request) {
	agentIDStr := mux.Vars(r)["id"]
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	o.LastPingTimestamp[agentID] = time.Now()

	err = o.Storage.UpdateAgent(agentID, storage.StatusAgentActive)
	if err != nil {
		log.Error("Error while updating agents")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		log.Info("Successfully updated agents")
	}

	w.WriteHeader(http.StatusOK)
}

func (o *Orchestrator) AddExpression(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		Expression string `json:"expression"`
	}

	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Error("Error while parsing request body")
		return
	}
	_, err := o.Storage.AddTask(request.Expression)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Error("Error while inserting expression to db")
		return
	}
	// TODO: push message to rabbitmq queue
}

func (o *Orchestrator) GetAllExpressions(w http.ResponseWriter, r *http.Request) {
	tasks, err := o.Storage.GetAllTasks()
	if err != nil {
		log.Error("Error while selecting all tasks")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		log.Info("Successfully selected all from tasks")
	}

	err = json.NewEncoder(w).Encode(&tasks)
	if err != nil {
		log.Error("Error while selecting all expressions")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		log.Info("Successfully returned all tasks")
	}
}

func (o *Orchestrator) StartHeartbeatCheck(duration time.Duration) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentTime := time.Now()
			for agentID, lastPingTime := range o.LastPingTimestamp {
				if currentTime.Sub(lastPingTime) > duration {
					log.Info("Agent is inactive: ", agentID.String())
					err := o.Storage.UpdateAgent(agentID, storage.StatusAgentInactive)
					if err != nil {
						log.Error("Error while updating agents table")
					} else {
						log.Info("Successfully updated agents table")
					}
				}
			}
		}
	}
}

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

	o.Timings["add"] = time.Millisecond * time.Duration(request.Add)
	o.Timings["sub"] = time.Millisecond * time.Duration(request.Sub)
	o.Timings["mul"] = time.Millisecond * time.Duration(request.Mul)
	o.Timings["div"] = time.Millisecond * time.Duration(request.Div)

	w.WriteHeader(http.StatusOK)
}

func (o *Orchestrator) GetTimeouts(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"add": o.Timings["add"].String(),
		"sub": o.Timings["sub"].String(),
		"mul": o.Timings["mul"].String(),
		"div": o.Timings["div"].String(),
	})
}

func (o *Orchestrator) StartHTTPServer(duration time.Duration) {
	log.Info("Starting HTTP server...")
	go o.StartHeartbeatCheck(duration)
	http.ListenAndServe(":8080", o.Router)
}

var schema = `
CREATE TABLE IF NOT EXISTS tasks (
	id VARCHAR(128) PRIMARY KEY,
	expression VARCHAR(128),
	status VARCHAR(128)
);

CREATE TABLE IF NOT EXISTS agents (
	id VARCHAR(128) PRIMARY KEY,
	status VARCHAR(128)
);
`

func main() {
	db, err := sqlx.Connect("sqlite3", "../db/database.db")
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

	if err != nil {
		log.Fatal(err)
		return
	}

	orchestrator.StartHTTPServer(30 * time.Second)
}
