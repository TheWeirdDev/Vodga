package daemon

import (
	"github.com/TheWeirdDev/Vodga/shared"
	"os"
	"os/exec"
)


type Openvpn struct {
	config    string
	creds     shared.Credentials
	process   *exec.Cmd
	connected bool
	state     string
	bytesIn   uint64
	bytesOut  uint64
	totalIn   uint64
	totalOut  uint64
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