package messages

import (
	"encoding/json"
	"errors"
	"github.com/TheWeirdDev/Vodga/shared"
	"github.com/TheWeirdDev/Vodga/shared/consts"
	"net"
	"strconv"
)

type Message struct {
	Command string            `json:"cmd"`
	Args    map[string]string `json:"args"`
}

func SendMessage(msg *Message, c net.Conn) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = c.Write(data)
	return err
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

func GetBytecountMsg() *Message {
	return &Message{Command: consts.MsgGetBytecount}
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

func ConnectMsg(cfgPath string, auth shared.Auth, creds ...string) *Message {
	msg := &Message{Command: consts.MsgConnect}
	if auth == shared.USER_PASS {
		msg.Args = map[string]string{"config": cfgPath, "auth": consts.AuthUserPass,
		"username": creds[0], "password": creds[1]}
	}
	return msg
}

func ErrorMsg(msg string) *Message {
	return &Message{consts.MsgError, map[string]string{"error": msg}}
}

func LogMsg(msg string) *Message {
	return &Message{consts.MsgLog, map[string]string{"log": msg}}
}
