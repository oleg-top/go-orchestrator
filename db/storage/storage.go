package storage

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	StatusAgentInactive    = "inactive"
	StatusAgentActive      = "active"
	StatusAgentCalculating = "calculating"
	StatusTaskCompleted    = "completed"
	StatusTaskCalculating  = "calculating"
	StatusTaskAccepted     = "accepted"
	StatusTaskInvalid      = "invalid"
)

type Storage struct {
	db *sqlx.DB
}

type Agent struct {
	ID         uuid.UUID `db:"id"`
	Status     string    `db:"status"`
	LastOnline string    `db:"last_online"`
}

type Task struct {
	ID         uuid.UUID `db:"id"`
	Expression string    `db:"expression"`
	Status     string    `db:"status"`
	Result     string    `db:"result"`
}

func (s *Storage) AddTask(expression string) (uuid.UUID, error) {
	task := &Task{
		ID:         uuid.New(),
		Expression: expression,
		Status:     StatusTaskAccepted,
		Result:     "",
	}
	_, err := s.db.Exec(
		"INSERT INTO tasks (id, expression, status, result) VALUES ($1, $2, $3, $4)",
		task.ID,
		task.Expression,
		task.Status,
		task.Result,
	)
	if err != nil {
		return uuid.Nil, err
	}
	return task.ID, nil
}

func (s *Storage) UpdateTask(id uuid.UUID, res, status string) error {
	_, err := s.db.Exec("UPDATE tasks SET status=$1, result=$2 WHERE id=$3", status, res, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) GetAllTasks() ([]Task, error) {
	var tasks []Task
	err := s.db.Select(&tasks, "SELECT * FROM tasks")
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *Storage) AddAgent() (uuid.UUID, error) {
	agent := &Agent{
		ID:         uuid.New(),
		Status:     StatusAgentActive,
		LastOnline: time.Now().Format(time.RFC1123Z),
	}
	// time format is yyyy-MM-dd HH:mm:ss

	_, err := s.db.Exec(
		"INSERT INTO agents (id, status, last_online) VALUES ($1, $2, $3)",
		agent.ID,
		agent.Status,
		agent.LastOnline,
	)
	if err != nil {
		return uuid.Nil, err
	}
	return agent.ID, nil
}

func (s *Storage) GetAllAgents() ([]Agent, error) {
	var agents []Agent
	err := s.db.Select(&agents, "SELECT * FROM agents")
	if err != nil {
		return nil, err
	}
	return agents, nil
}

func (s *Storage) UpdateAgentStatus(id uuid.UUID, status string) error {
	_, err := s.db.Exec("UPDATE agents SET status=$1 WHERE id=$2", status, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) UpdateAgentLastOnline(id uuid.UUID, lastOnline time.Time) error {
	lastOnlineString := lastOnline.Format(time.RFC1123Z)
	_, err := s.db.Exec("UPDATE agents SET last_online=$1 WHERE id=$2", lastOnlineString, id)
	if err != nil {
		return err
	}
	return nil
}

func NewStorage(db *sqlx.DB) *Storage {
	return &Storage{
		db: db,
	}
}
