package messages

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Message struct {
	Command string                 `json:"command"`
	Parameters map[string]string   `json:"parameters"`
}

func UnmarshalMsg(cmd string) (*Message, error) {
	msg := &Message{}
	err := json.Unmarshal([]byte(cmd), msg)
	if msg.Command == "" {
		return nil, errors.New("command is a empty")
	}
	return msg, err
}

func (msg *Message) EnsureEnoughArguments(count int) error {
	c := len(msg.Parameters)
	if c != count {
		return fmt.Errorf("command %q takes %d argument(s) but %d were given", msg.Command, count, c)
	}
	return nil
}