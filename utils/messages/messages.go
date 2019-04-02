package messages

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TheWeirdDev/Vodga/utils/consts"
)

type Message struct {
	Command    string            `json:"command"`
	Parameters map[string]string `json:"parameters"`
}

func SimpleMsg(cmd string) *Message{
	return &Message{Command:cmd}
}
func UnmarshalMsg(text string) (*Message, error) {
	msg := &Message{}
	err := json.Unmarshal([]byte(text), msg)
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

func ErrorMsg(msg string) *Message {
	return &Message{consts.MsgError, map[string]string{"error": msg}}
}

func LogMsg(msg string) *Message {
	return &Message{consts.MsgLog, map[string]string{"log": msg}}
}