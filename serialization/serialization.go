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
