package daemon

import (
	"os"
	"os/exec"
)

type Auth int

const (
	NO_AUTH Auth = iota
	USER_PASS
)

type credentials struct {
	auth       Auth
	username   string
	password   string
}

type Openvpn struct {
	config    string
	creds     credentials
	process   *exec.Cmd
	connected bool
	state     string
	bytesIn   int64
	bytesOut  int64
}

func (o *Openvpn) closeConnection() error {
	return o.process.Process.Signal(os.Interrupt)
}

func (o *Openvpn) isRunning() bool {
	return o.process != nil
}