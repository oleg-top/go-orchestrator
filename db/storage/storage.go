package storage

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Статусы выражений и агентов
var (
	StatusAgentInactive   = "inactive"
	StatusAgentActive     = "active"
	StatusTaskCompleted   = "completed"
	StatusTaskCalculating = "calculating"
	StatusTaskAccepted    = "accepted"
	StatusTaskInvalid     = "invalid"
	StatusTaskRepublished = "republished"
)

// Структура хранилища
type Storage struct {
	db *sqlx.DB
}

// Структура агента, которая хранится в бд
type Agent struct {
	ID         uuid.UUID `db:"id"`
	Status     string    `db:"status"`
	LastOnline string    `db:"last_online"`
}

// Структура задачи, которая хранится в бд
type Task struct {
	ID         uuid.UUID `db:"id"`
	Expression string    `db:"expression"`
	Status     string    `db:"status"`
	Result     string    `db:"result"`
	AgentID    uuid.UUID `db:"agent_id"`
}

// Записывает задачу в бд
func (s *Storage) AddTask(expression string) (uuid.UUID, error) {
	task := &Task{
		ID:         uuid.New(),
		Expression: expression,
		Status:     StatusTaskAccepted,
		Result:     "",
		AgentID:    uuid.Nil,
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

// Обновляет статус задачи в бд
func (s *Storage) UpdateTaskStatus(id uuid.UUID, status string) error {
	_, err := s.db.Exec("UPDATE tasks SET status=$1 WHERE id=$2", status, id)
	if err != nil {
		return err
	}
	return nil
}

// Обновляет результат задачи в бд
func (s *Storage) UpdateTaskResult(id uuid.UUID, res string) error {
	_, err := s.db.Exec("UPDATE tasks SET result=$1 WHERE id=$2", res, id)
	if err != nil {
		return err
	}
	return nil
}

// Обновляет у задачи айди агента, который ее выполняет
func (s *Storage) UpdateTaskAgentID(taskID uuid.UUID, agentID uuid.UUID) error {
	_, err := s.db.Exec("UPDATE tasks SET agent_id=$1 WHERE id=$2", agentID, taskID)
	if err != nil {
		return err
	}
	return nil
}

// Возвращает задачу по ее айди
func (s *Storage) GetTaskById(id uuid.UUID) ([]Task, error) {
	var tasks []Task
	err := s.db.Select(&tasks, "SELECT * FROM tasks WHERE id=$1", id)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

// Возвращает все задачи из бд
func (s *Storage) GetAllTasks() ([]Task, error) {
	var tasks []Task
	err := s.db.Select(&tasks, "SELECT * FROM tasks")
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

// Добавляет агента в бд
func (s *Storage) AddAgent() (uuid.UUID, error) {
	agent := &Agent{
		ID:         uuid.New(),
		Status:     StatusAgentActive,
		LastOnline: time.Now().Format(time.RFC1123Z),
	}

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

// Возвращает агента по его айди
func (s *Storage) GetAgentById(id uuid.UUID) ([]Agent, error) {
	var agents []Agent
	err := s.db.Select(&agents, "SELECT * FROM agents WHERE id=$1", id)
	if err != nil {
		return nil, err
	}
	return agents, nil
}

// Возвращает всех агентов из бд
func (s *Storage) GetAllAgents() ([]Agent, error) {
	var agents []Agent
	err := s.db.Select(&agents, "SELECT * FROM agents")
	if err != nil {
		return nil, err
	}
	return agents, nil
}

// Обновляет статус агента по его айди
func (s *Storage) UpdateAgentStatus(id uuid.UUID, status string) error {
	_, err := s.db.Exec("UPDATE agents SET status=$1 WHERE id=$2", status, id)
	if err != nil {
		return err
	}
	return nil
}

// Обновляет последнее время в сети агента
func (s *Storage) UpdateAgentLastOnline(id uuid.UUID, lastOnline time.Time) error {
	lastOnlineString := lastOnline.Format(time.RFC1123Z)
	_, err := s.db.Exec("UPDATE agents SET last_online=$1 WHERE id=$2", lastOnlineString, id)
	if err != nil {
		return err
	}
	return nil
}

// Возвращает новый экземпляр хранилища
func NewStorage(db *sqlx.DB) *Storage {
	return &Storage{
		db: db,
	}
}
