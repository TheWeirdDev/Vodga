package daemon

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/TheWeirdDev/Vodga/shared"
	"github.com/TheWeirdDev/Vodga/shared/consts"
	"github.com/TheWeirdDev/Vodga/shared/messages"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Daemon struct {
	quit    chan struct{}
	ln      net.Listener
	conns   []net.Conn
	mtx     sync.Mutex
	openvpn Openvpn
}

func NewDaemon() *Daemon {
	instance := &Daemon{}
	instance.quit = make(chan struct{})
	instance.openvpn = Openvpn{connected: false, bytesIn: 0, bytesOut: 0}

	ln, err := net.Listen("unix", consts.UnixSocket)
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
	instance.ln = ln

	// Make the socket accessible to users other than root
	if err := os.Chmod(consts.UnixSocket, os.FileMode(0777)); err != nil {
		log.Fatalf("Failed to set socket permissions: %v", err)
	}
	return instance
}

func (d *Daemon) StartServer() {

	// Listen for OS signals and close the socket before exiting
	go func(ln net.Listener, kill *chan struct{}) {
		sigchan := make(chan os.Signal, 5)
		signal.Notify(sigchan, os.Interrupt, syscall.SIGPIPE, syscall.SIGKILL,
			syscall.SIGTERM, syscall.SIGQUIT)
		// Waits for signals
		err := <-sigchan
		log.Printf("Server Killed by: '%v', closing openvpn\n", err)
		d.openvpn.closeConnection()
		time.Sleep(1500 * time.Millisecond)
		close(*kill)
		ln.Close()
	}(d.ln, &d.quit)

loop:
	for {
		// Waits for new client connection
		conn, err := d.ln.Accept()
		if err != nil {
			select {

			// If we write something to this channel,
			// it means server stopped for a known reason
			case <-d.quit:
				log.Println("Server stopped")
				// Otherwise the error is unknown and needs to be handled
			default:
				log.Printf("Accept error: %+v", err)
			}
			break loop
		}

		d.conns = append(d.conns, conn)
		go d.daemonServer(conn, len(d.conns)-1)
	}
}

func (d *Daemon) daemonServer(c net.Conn, id int) {
	defer func(c *net.Conn) {
		(*c).Close()
		d.conns[id] = nil
	}(&c)

	log.Printf("Client #%d connected", id)
	scanner := bufio.NewScanner(c)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if len(text) == 0 {
			continue
		}

		log.Printf("Client %d: %s\n", id, text)
		msg, err := messages.UnmarshalMsg(text)
		if err != nil {
			log.Printf("Got invalid message: %v", err)
			messages.SendMessage(messages.SimpleMsg(consts.UnknownCmd), c)
			continue
		}
		d.processMessage(msg, c)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Server error: %v\n", err)
	}
	log.Printf("Client #%d disconnected\n", id)
}


func (d *Daemon) stopServer(c net.Conn) {
	close(d.quit)
	// Send the information before closing the socket
	d.broadcastMessage(messages.SimpleMsg(consts.MsgKilled))
	if err := d.ln.Close(); err != nil {
		log.Fatalf("Server returned an error: %v\n", err)
	}
}

func (d *Daemon) startOpenVPN(c net.Conn) {
	cmd := exec.Command("openvpn", "--config", d.openvpn.config,
		"--management", consts.MgmtSocket, "unix", "--management-query-passwords",
		"--management-hold")

	// Kill other openvpn instances before starting this one
	d.killOpenvpn()

	// Reset openvpn state after it's closed
	defer d.resetOpenvpn()

	// create a pipe for the output of the script
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		messages.SendMessage(messages.ErrorMsg("Can't open StdoutPipe for OpenVPN"), c)
		return
	}

	scanner := bufio.NewScanner(cmdReader)
	//TODO: scanner.Split()

	go func() {
		for scanner.Scan() {
			d.broadcastMessage(messages.LogMsg(scanner.Text()))
		}
	}()

	err = cmd.Start()
	if err != nil {
		messages.SendMessage(messages.ErrorMsg("Can't start OpenVPN"), c)
		return
	}
	go d.connectToMgmt()
	d.openvpn.process = cmd

	err = cmd.Wait()
	if err != nil {
		log.Println("OpenVPN closed unexpectedly")
	} else {
		log.Println("OpenVPN Closed")
	}

	d.broadcastMessage(messages.SimpleMsg(consts.MsgDisconnected))
}

func (d *Daemon) processMessage(msg *messages.Message, c net.Conn) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	switch msg.Command {
	case consts.MsgStop:
		d.stopServer(c)

	case consts.MsgConnect:
		if err := d.prepareOpenvpn(msg, c); err != nil {
			log.Printf("Error: %v\n", err)
			return
		}
		go d.startOpenVPN(c)

	case consts.MsgDisconnect:
		if !d.openvpn.isRunning() {
			messages.SendMessage(messages.ErrorMsg("OpenVPN is not running"), c)
			return
		}
		if err := d.openvpn.closeConnection(); err != nil {
			log.Println("Can't close openvpn")
			messages.SendMessage(messages.ErrorMsg("Can't close openvpn"), c)
		}

	case consts.MsgKillOpenvpn:
		d.killOpenvpn()

	case consts.MsgGetBytecount:
		messages.SendMessage(messages.BytecountMsg(d.openvpn.bytesIn, d.openvpn.bytesOut,
			d.openvpn.totalIn, d.openvpn.bytesOut), c)

	default:
		log.Printf("Unknown command: %v\n", msg.Command)
		messages.SendMessage(messages.ErrorMsg(consts.UnknownCmd), c)
	}
}

func (d *Daemon) writeToMgmt(text string, c net.Conn) {
	text += "\n"
	if _, err := c.Write([]byte(text)); err != nil {
		log.Println("Error: can't write to openvpn management")
	}
}

func (d *Daemon) connectToMgmt() {
	tries := 10
	var c net.Conn

	// Try connecting to mgmt socket until it's connected
	// Maximum tries: 10 times
	for {
		var err error
		c, err = net.Dial("unix", consts.MgmtSocket)
		if err != nil {
			time.Sleep(300 * time.Millisecond)
			if tries == 0 {
				log.Fatalf("Can't connect to management socket\n")
			}
			tries--
		} else {
			log.Println("Connected to mgmt socket")
			break
		}
	}
	defer c.Close()

	// Starts the show
	d.writeToMgmt("hold release", c)
	d.writeToMgmt("state on", c)
	d.writeToMgmt("bytecount 1", c)

	scanner := bufio.NewScanner(c)
	for scanner.Scan() {
		txt := scanner.Text()
		//TODO: Remove all debug lines after it's done
		log.Println("MGMT: ", txt)
		d.processMgmtCommand(txt, c)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Management error: %v\n", err)
	}
}

func (d *Daemon) killOpenvpn() {
	cmd := exec.Command("killall", "-e", "-9", "openvpn")
	if err := cmd.Run(); err != nil {
		log.Printf("Error while killing openvpn: %v\n", err)
	}
}

func (d *Daemon) resetOpenvpn() {
	d.openvpn.config = ""
	d.openvpn.bytesOut = 0
	d.openvpn.bytesIn = 0
	d.openvpn.connected = false
	d.openvpn.state = ""
	d.openvpn.creds = shared.Credentials{}
}

func (d *Daemon) broadcastMessage(msg *messages.Message) {
	for _, conn := range d.conns {
		if conn != nil {
			messages.SendMessage(msg, conn)
		}
	}
}

func (d *Daemon) prepareOpenvpn(msg *messages.Message, c net.Conn) error {
	if !d.openvpn.connected {
		config, ok := msg.Args["config"]
		if !ok {
			messages.SendMessage(messages.ErrorMsg("Config is needed to start openvpn"), c)
			return errors.New("no config was given")
		}
		auth, ok := msg.Args["auth"]
		if !ok {
			messages.SendMessage(messages.ErrorMsg("Auth method is needed to start openvpn"), c)
			return errors.New("no auth method was given")
		}
		d.openvpn.config = config
		switch auth {
		case consts.AuthNoAuth:
			d.openvpn.creds = shared.Credentials{Auth: shared.NO_AUTH}
		case consts.AuthUserPass:
			username, ok := msg.Args["username"]
			if !ok {
				messages.SendMessage(messages.ErrorMsg("Username is needed to start openvpn"), c)
				return errors.New("no config was given")
			}
			password, ok := msg.Args["password"]
			if !ok {
				messages.SendMessage(messages.ErrorMsg("Password is needed to start openvpn"), c)
				return errors.New("no config was given")
			}
			d.openvpn.creds = shared.Credentials{Auth: shared.USER_PASS, Username: username, Password: password}
		default:
			d.resetOpenvpn()
			messages.SendMessage(messages.ErrorMsg("Unknown auth type"), c)
			return errors.New("unknown auth type")
		}
	} else {
		messages.SendMessage(messages.ErrorMsg("OpenVPN is already running"), c)
	}
	return nil
}

func (d *Daemon) processMgmtCommand(cmd string, c net.Conn) {
	if len(cmd) < 1 {
		return
	} else if cmd[0] != '>' {
		if strings.HasPrefix(cmd, "ERROR:") {
			colonIndex := strings.IndexRune(cmd, ':')
			errstr := cmd[colonIndex+1:]
			log.Println("Mgmt error: ", cmd[colonIndex+1:])
			d.broadcastMessage(messages.ErrorMsg(errstr))
			return
		}
	}

	fieldsFunc := func(r rune) bool {
		return r == ','
	}
	colonIndex := strings.IndexRune(cmd, ':')

	switch cmd[1:colonIndex] {
	case "PASSWORD":
		const errstr = "Verification Failed"
		if strings.Contains(cmd, errstr) {
			log.Println("Invalid credentials")
			d.broadcastMessage(messages.ErrorMsg(consts.MsgAuthFailed))
			return
		}
		userpass := fmt.Sprintf(`username "Auth" %s
								 password "Auth" %s`,
			d.openvpn.creds.Username, d.openvpn.creds.Password)

		d.writeToMgmt(userpass, c)

	case "STATE":
		state := cmd[colonIndex+1:]
		states := strings.FieldsFunc(state, fieldsFunc)
		if len(state) < 2 {
			return
		}
		state = states[1]
		d.broadcastMessage(messages.StateMsg(state))

	case "BYTECOUNT":
		data := cmd[colonIndex+1:]
		inout := strings.FieldsFunc(data, fieldsFunc)
		if len(inout) < 2 {
			return
		}
		in, _ := strconv.Atoi(inout[0])
		out, _ := strconv.Atoi(inout[1])
		d.openvpn.totalIn += in - d.openvpn.bytesIn
		d.openvpn.totalOut += out - d.openvpn.bytesOut
		d.openvpn.bytesIn = in
		d.openvpn.bytesOut = out
	}

}
