package storage

import (
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
)

type Storage struct {
	db *sqlx.DB
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

func (s *Storage) AddTask(expression string) (uuid.UUID, error) {
	task := &Task{
		ID:         uuid.New(),
		Expression: expression,
		Status:     StatusTaskAccepted,
	}
	_, err := s.db.Exec(
		"INSERT INTO tasks (id, expression, status) VALUES ($1, $2, $3)",
		task.ID,
		task.Expression,
		task.Status,
	)
	if err != nil {
		return uuid.Nil, err
	}
	return task.ID, nil
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
		ID:     uuid.New(),
		Status: StatusAgentActive,
	}

	_, err := s.db.Exec("INSERT INTO agents (id, status) VALUES ($1, $2)", agent.ID, agent.Status)
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

func (s *Storage) UpdateAgent(id uuid.UUID, status string) error {
	_, err := s.db.Exec("UPDATE agents SET status=$1 WHERE id=$2", status, id)
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
