package serialization

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Интерфейс сообщения для RabbitMQ
type Message interface {
	String() string
}

// Структура сообщения, хранящего в себе выражение
type TaskMessage struct {
	ID         uuid.UUID                `json:"id"`
	Expression string                   `json:"expression"`
	Timeouts   map[string]time.Duration `json:"timings"`
}

// Возвращает строковое представление сообщения
func (tm TaskMessage) String() string {
	return fmt.Sprintf(
		"Expression: %s; ID: %s; Timings: %v",
		tm.Expression,
		tm.ID.String(),
		tm.Timeouts,
	)
}

// Структура сообщения, хранящего в себе результат выражения
type ResultMessage struct {
	ID     uuid.UUID `json:"id"`
	Result string    `json:"result"`
	Status string    `json:"status"`
}

// Возвращает строковое представление сообщения
func (rm ResultMessage) String() string {
	return fmt.Sprintf("ID: %s; Result: %s; Status: %s", rm.ID, rm.Result, rm.Status)
}

// Структура сообщения, хранящего в себе айди выражения и айди агента, на котором вычисляется выражение
type CalculatingMessage struct {
	AgentID uuid.UUID `json:"agent_id"`
	TaskID  uuid.UUID `json:"task_id"`
}

// Возвращает строковое представление сообщения
func (cm CalculatingMessage) String() string {
	return fmt.Sprintf("AgentID: %s; TaskID: %s", cm.AgentID.String(), cm.TaskID.String())
}

// Переводит сообщение в байты
func Serialize[T Message](msg T) ([]byte, error) {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	err := encoder.Encode(msg)
	return b.Bytes(), err
}

// Переводит байты в сообщение
func Deserialize[T Message](b []byte) (T, error) {
	var msg T
	buf := bytes.NewBuffer(b)
	decoder := json.NewDecoder(buf)
	err := decoder.Decode(&msg)
	return msg, err
}
