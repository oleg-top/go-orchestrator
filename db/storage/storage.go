package storage

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	AGENT_INACTIVE    = "inactive"
	AGENT_ACTIVE      = "active"
	AGENT_CALCULATING = "calculating"
	TASK_COMPLETED    = "completed"
	TASK_CALCULATING  = "calculating"
	TASK_ACCEPTED     = "accepted"
)

type Storage struct {
	DB *sqlx.DB
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

func (s *Storage) addTask(expression, status string) (uuid.UUID, error) {
	task := &Task{
		ID:         uuid.New(),
		Expression: expression,
		Status:     TASK_ACCEPTED,
	}
	_, err := s.DB.Exec(
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

func (s *Storage) getAllTasks() ([]Task, error) {
	var tasks []Task
	err := s.DB.Select(&tasks, "SELECT * FROM tasks")
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *Storage) addAgent() (uuid.UUID, error) {
	agent := &Agent{
		ID:     uuid.New(),
		Status: AGENT_ACTIVE,
	}

	_, err := s.DB.Exec("INSERT INTO agents (id, status) VALUES ($1, $2)", agent.ID, agent.Status)
	if err != nil {
		return uuid.Nil, err
	}
	return agent.ID, nil
}

func (s *Storage) getAllAgents() ([]Agent, error) {
	var agents []Agent
	err := s.DB.Select(&agents, "SELECT * FROM agents")
	if err != nil {
		return nil, err
	}
	return agents, nil
}

func (s *Storage) updateAgents(id uuid.UUID, status string) error {
	_, err := s.DB.Exec("UPDATE agents SET status=$1 WHERE id=$2", status, id)
	if err != nil {
		return err
	}
	return nil
}
