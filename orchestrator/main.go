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
)

var (
	AGENT_INACTIVE    = "inactive"
	AGENT_ACTIVE      = "active"
	AGENT_CALCULATING = "calculating"
	TASK_COMPLETED    = "completed"
	TASK_CALCULATING  = "calculating"
	TASK_ACCEPTED     = "accepted"
)

type Orchestrator struct {
	DB                *sqlx.DB
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
		DB:                db,
		Channel:           ch,
		Router:            mux.NewRouter(),
		LastPingTimestamp: make(map[uuid.UUID]time.Time),
	}
	orchestrator.setupRoutes()

	return orchestrator
}

func (o *Orchestrator) setupRoutes() {
	o.Router.HandleFunc("/agents", o.addAgent).Methods("POST")
	o.Router.HandleFunc("/agents", o.getAllAgents).Methods("GET")
	o.Router.HandleFunc("/agents/{id}/ping", o.agentPing).Methods("POST")
	o.Router.HandleFunc("/expressions", o.addExpression).Methods("POST")
	o.Router.HandleFunc("/expressions", o.getAllExpressions).Methods("GET")
	o.Router.HandleFunc("/timeouts", o.setTimeouts).Methods("POST")
	o.Router.HandleFunc("/timeouts", o.getTimeouts).Methods("GET")
}

func (o *Orchestrator) addAgent(w http.ResponseWriter, r *http.Request) {
	agent := &Agent{
		ID:     uuid.New(),
		Status: AGENT_ACTIVE,
	}

	_, err := o.DB.Exec("INSERT INTO agents (id, status) VALUES ($1, $2)", agent.ID, agent.Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	o.LastPingTimestamp[agent.ID] = time.Now()
	json.NewEncoder(w).Encode(map[string]string{"id": agent.ID.String()})
}

func (o *Orchestrator) getAllAgents(w http.ResponseWriter, r *http.Request) {
	var agents []Agent
	o.DB.Select(&agents, "SELECT * FROM agents")
	err := json.NewEncoder(w).Encode(&agents)
	if err != nil {
		log.Fatal("Error while selecting all agents")
	} else {
		log.Info("Successfully returned all agents slice")
	}
}

func (o *Orchestrator) agentPing(w http.ResponseWriter, r *http.Request) {
	agentIDStr := mux.Vars(r)["id"]
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	o.LastPingTimestamp[agentID] = time.Now()

	_, err = o.DB.Exec("UPDATE agents SET status=$1 WHERE id=$2", AGENT_ACTIVE, agentID)
	if err != nil {
		log.Fatal("Error while updating agents")
	} else {
		log.Info("Successfully updated agents")
	}

	w.WriteHeader(http.StatusOK)
}

func (o *Orchestrator) addExpression(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		Expression string `json:"expression"`
	}

	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Fatal("Error while parsing request body")
		return
	}
	task := Task{
		ID:         uuid.New(),
		Expression: request.Expression,
		Status:     TASK_ACCEPTED,
	}
	_, err := o.DB.Exec(
		"INSERT INTO tasks (id, expression, status) VALUES ($1, $2, $3)",
		task.ID,
		task.Expression,
		task.Status,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Fatal("Error while inserting expression to db")
		return
	}
	// TODO: push message to rabbitmq queue
}

func (o *Orchestrator) getAllExpressions(w http.ResponseWriter, r *http.Request) {
	var tasks []Task
	err := o.DB.Select(&tasks, "SELECT * FROM tasks")
	if err != nil {
		log.Fatal("Error while selecting all tasks")
	} else {
		log.Info("Successfully selected all from tasks")
	}

	err = json.NewEncoder(w).Encode(&tasks)
	if err != nil {
		log.Fatal("Error while selecting all expressions")
	} else {
		log.Info("Successfully returned all tasks")
	}
}

func (o *Orchestrator) startHeartbeatCheck(duration time.Duration) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentTime := time.Now()
			for agentID, lastPingTime := range o.LastPingTimestamp {
				if currentTime.Sub(lastPingTime) > duration {
					log.Info("Agent is inactive: ", agentID.String())
					_, err := o.DB.Exec(
						"UPDATE agents SET status=$1 WHERE id=$2",
						AGENT_INACTIVE,
						agentID,
					)
					if err != nil {
						log.Fatal("Error while updating agents table")
					} else {
						log.Info("Successfully updated agents table")
					}
				}
			}
		}
	}
}

func (o *Orchestrator) setTimeouts(w http.ResponseWriter, r *http.Request) {
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

func (o *Orchestrator) getTimeouts(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"add": o.Timings["add"].String(),
		"sub": o.Timings["sub"].String(),
		"mul": o.Timings["mul"].String(),
		"div": o.Timings["div"].String(),
	})
}

func (o *Orchestrator) startHTTPServer(duration time.Duration) {
	log.Info("Starting HTTP server...")
	go o.startHeartbeatCheck(duration)
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

	orchestrator.startHTTPServer(30 * time.Second)
}
