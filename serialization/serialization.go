package serialization

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Message interface {
	String() string
}

type TaskMessage struct {
	ID         uuid.UUID                `json:"id"`
	Expression string                   `json:"expression"`
	Timeouts   map[string]time.Duration `json:"timings"`
}

func (tm TaskMessage) String() string {
	return fmt.Sprintf(
		"Expression: %s; ID: %s; Timings: %v",
		tm.Expression,
		tm.ID.String(),
		tm.Timeouts,
	)
}

type ResultMessage struct {
	ID     uuid.UUID `json:"id"`
	Result string    `json:"result"`
	Status string    `json:"status"`
}

func (rm ResultMessage) String() string {
	return fmt.Sprintf("ID: %s; Result: %s; Status: %s", rm.ID, rm.Result, rm.Status)
}

func Serialize[T Message](msg T) ([]byte, error) {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	err := encoder.Encode(msg)
	return b.Bytes(), err
}

func Deserialize[T Message](b []byte) (T, error) {
	var msg T
	buf := bytes.NewBuffer(b)
	decoder := json.NewDecoder(buf)
	err := decoder.Decode(&msg)
	return msg, err
}
