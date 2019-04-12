package messages

import (
	"encoding/json"
	"errors"
	"github.com/TheWeirdDev/Vodga/utils/consts"
	"strconv"
)

type Message struct {
	Command string            `json:"cmd"`
	Args    map[string]string `json:"args"`
}

func SimpleMsg(cmd string) *Message {
	return &Message{Command: cmd}
}

func BytecountMsg(in, out, tin, tout int) *Message {
	bytecount := map[string]string{
		"in":   strconv.Itoa(in),
		"out":  strconv.Itoa(out),
		"tin":  strconv.Itoa(tin),
		"tout": strconv.Itoa(tout),
	}

	return &Message{Command: consts.MsgByteCount,
		Args: bytecount}
}

func StateMsg(state string) *Message {
	return &Message{Command: consts.MsgStateChanged,
		Args: map[string]string{"state": state}}
}

func UnmarshalMsg(text string) (*Message, error) {
	msg := &Message{}
	err := json.Unmarshal([]byte(text), msg)
	if msg.Command == "" {
		return nil, errors.New("command is a empty")
	}
	return msg, err
}

func ErrorMsg(msg string) *Message {
	return &Message{consts.MsgError, map[string]string{"error": msg}}
}

func LogMsg(msg string) *Message {
	return &Message{consts.MsgLog, map[string]string{"log": msg}}
}
