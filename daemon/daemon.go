package daemon

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TheWeirdDev/Vodga/utils/consts"
	"github.com/TheWeirdDev/Vodga/utils/messages"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Daemon struct {
	quit    chan struct{}
	ln      net.Listener
	conns   []net.Conn
	mux     sync.Mutex
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
		log.Printf("Server Killed by: %v", err)
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

		log.Printf("Server got(#%d): %s\n", id, text)
		msg, err := messages.UnmarshalMsg(text)
		if err != nil {
			log.Printf("Got invalid command: %v", err)
			d.sendMessage(messages.SimpleMsg(consts.UnknownCmd), c)
			continue
		}
		d.processMessage(msg, c)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Server error: %v\n", err)
	}
	log.Printf("Client #%d disconnected\n", id)
}

func (d *Daemon) sendMessage(msg *messages.Message, c net.Conn) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}
	data = append(data, '\n')
	_, err = c.Write(data)
	if err != nil {
		log.Println("Error: can't write to connection")
	}
}

func (d *Daemon) stopServer(c net.Conn) {
	close(d.quit)
	if err := d.ln.Close(); err != nil {
		log.Fatalf("Server returned an error: %v\n", err)
	}
	d.sendMessage(messages.SimpleMsg(consts.MsgKilled), c)
}

func (d *Daemon) startOpenVPN(c net.Conn) {
	cmd := exec.Command("openvpn", "--config", d.openvpn.config,
		"--management", consts.MgmtSocket, "unix", "--management-query-passwords")

	// create a pipe for the output of the script
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		d.sendMessage(messages.ErrorMsg("Can't open StdoutPipe for OpenVPN"), c)
		return
	}

	scanner := bufio.NewScanner(cmdReader)
	//TODO: scanner.Split()

	go func() {
		for scanner.Scan() {
			d.sendMessage(messages.LogMsg(scanner.Text()), c)
		}
	}()

	err = cmd.Start()
	if err != nil {
		d.sendMessage(messages.ErrorMsg("Can't start OpenVPN"), c)
		return
	}
	go d.connectToMgmt()
	d.openvpn.process = cmd

	err = cmd.Wait()
	if err != nil {
		d.openvpn.process = nil
		log.Println("OpenVPN closed unexpectedly")
		d.sendMessage(messages.SimpleMsg(consts.MsgDisconnected), c)
		return
	}

	d.openvpn.process = nil
	log.Println("OpenVPN Closed")
	d.sendMessage(messages.SimpleMsg(consts.MsgDisconnected), c)

}

func (d *Daemon) processMessage(msg *messages.Message, c net.Conn) {
	d.mux.Lock()
	defer d.mux.Unlock()

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
		if err := msg.EnsureEnoughArguments(0); err != nil {
			d.sendMessage(messages.ErrorMsg(err.Error()), c)
			return
		}
		if !d.openvpn.isRunning() {
			d.sendMessage(messages.ErrorMsg("OpenVPN is not running"), c)
			return
		}
		if err := d.openvpn.closeConnection(); err != nil {
			log.Println("Can't close openvpn")
			d.sendMessage(messages.ErrorMsg("Can't close openvpn"), c)
		}

	default:
		log.Printf("Unknown command: %v\n", msg.Command)
		d.sendMessage(messages.ErrorMsg(consts.UnknownCmd), c)
	}
}

func (d *Daemon) connectToMgmt() {
	tries := 10
	var c net.Conn

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

	log.Println("Connected to management socket")
	defer c.Close()
	//"bytecount 1\n"
	if _, err := c.Write([]byte("state on\n")); err != nil {
		log.Fatalf("Error: can't write to openvpn management\n")
	}

	scanner := bufio.NewScanner(c)
	for scanner.Scan() {
		txt := scanner.Text()
		//TODO: Remove all debug lines after it's done
		log.Println("$$$ GOT: ", txt)
		d.processMgmtCommand(txt, c)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Management error: %v\n", err)
	}
}

func (d *Daemon) prepareOpenvpn(msg *messages.Message, c net.Conn) error {
	if !d.openvpn.connected {
		config, ok := msg.Parameters["config"]
		if !ok {
			d.sendMessage(messages.ErrorMsg("Config is needed to start openvpn"), c)
			return errors.New("no config was given")
		}
		auth, ok := msg.Parameters["auth"]
		if !ok {
			d.sendMessage(messages.ErrorMsg("Auth method is needed to start openvpn"), c)
			return errors.New("no auth method was given")
		}
		switch auth {
		case consts.AuthNoAuth:
			d.openvpn.config = config
			d.openvpn.creds = credentials{auth: NO_AUTH}
		case consts.AuthUserPass:
			username, ok := msg.Parameters["username"]
			if !ok {
				d.sendMessage(messages.ErrorMsg("Username is needed to start openvpn"), c)
				return errors.New("no config was given")
			}
			password, ok := msg.Parameters["password"]
			if !ok {
				d.sendMessage(messages.ErrorMsg("Password is needed to start openvpn"), c)
				return errors.New("no config was given")
			}
			d.openvpn.config = config
			d.openvpn.creds = credentials{auth: USER_PASS, username: username, password: password}
		default:
			d.sendMessage(messages.ErrorMsg("Unknown auth type"), c)
			return errors.New("unknown auth type")
		}
	} else {
		d.sendMessage(messages.ErrorMsg("OpenVPN is already running"), c)
	}
	return nil
}

func (d *Daemon) processMgmtCommand(cmd string, c net.Conn) {
	authTemplate := `username "Auth" %s
					 password "Auth" %s
					 `
	if len(cmd) < 1 && cmd[0] != '>' {
		return
	}
	colonIndex := strings.IndexRune(cmd, ':')

	switch cmd[1:colonIndex] {
	case "PASSWORD":
		errstr := "Verification Failed"
		if strings.Contains(cmd, errstr){
			log.Println("Invalid credentials")
			//d.sendMessage(messages.ErrorMsg(errstr, ))
			return
		}
		userpass := fmt.Sprintf(authTemplate,
			d.openvpn.creds.username, d.openvpn.creds.password)

		if _, err := c.Write([]byte(userpass)); err != nil {
			log.Fatalf("Error: can't write to openvpn management\n")
		}
	}

}
