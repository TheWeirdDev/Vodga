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
	bytesIn   int
	bytesOut  int
	totalIn   int
	totalOut  int
}

func (o *Openvpn) closeConnection() error {
	if o.process != nil {
		return o.process.Process.Signal(os.Interrupt)
	}
	return nil
}

func (o *Openvpn) isRunning() bool {
	return o.process != nil
}