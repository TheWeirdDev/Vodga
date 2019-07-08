package messages

import (
	"encoding/json"
	"errors"
	"github.com/TheWeirdDev/Vodga/shared/auth"
	"github.com/TheWeirdDev/Vodga/shared/consts"
	"github.com/TheWeirdDev/Vodga/shared/utils"
	"log"
	"net"
)

type Message struct {
	Command string            `json:"cmd"`
	Args    map[string]string `json:"args"`
}

func SendMessage(msg *Message, c net.Conn) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Fatalf("Can't marshal message")
	}
	data = append(data, '\n')
	_, err = c.Write(data)
	if err != nil {
		log.Fatalf("Can't send message")
	}
}

func SimpleMsg(cmd string) *Message {
	return &Message{Command: cmd}
}

func BytecountMsg(in, out, tin, tout uint64) *Message {
	i, o, ti, to := utils.BytecountToString(in, out, tin, tout)
	bytecount := map[string]string{
		"in": i, "out": o, "tin": ti, "tout": to,
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

func ConnectMsg(cfgPath string, authMethod auth.Auth, creds ...string) *Message {
	msg := &Message{Command: consts.MsgConnect}
	if authMethod == auth.USER_PASS {
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
