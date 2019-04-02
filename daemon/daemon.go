package daemon

import (
	"bufio"
	"encoding/json"
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
	openvpn *exec.Cmd
}

func NewDaemon() *Daemon {
	instance := &Daemon{}
	instance.quit = make(chan struct{})

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
	if err := scanner.Err(); err != nil{
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
	_, err = c.Write([]byte(string(data) + "\n"))
	if err != nil {
		log.Println("Error: can't write to connection")
	}
}

func (d *Daemon) stopServer(c net.Conn) {
	close(d.quit)
	if err := d.ln.Close(); err != nil {
		log.Printf("Server return an error: %v\n", err)
	}
	d.sendMessage(messages.SimpleMsg(consts.MsgKilled), c)
}

func (d *Daemon) startOpenVPN(msg *messages.Message, c net.Conn) {
	config, ok := msg.Parameters["config"]
	if !ok {
		d.sendMessage(messages.ErrorMsg("Config is needed to start openvpn"), c)
		log.Println("Error: no config was given")
		return
	}
	cmd := exec.Command("openvpn", "--config", config,
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
	d.openvpn = cmd

	err = cmd.Wait()
	if err != nil {
		d.openvpn = nil
		log.Println("OpenVPN closed unexpectedly")
		d.sendMessage(messages.SimpleMsg(consts.MsgDisconnected), c)
		return
	}

	d.openvpn = nil
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
		if err := msg.EnsureEnoughArguments(2); err != nil {
			d.sendMessage(messages.ErrorMsg(err.Error()), c)
			return
		}
		if d.openvpn == nil {
			go d.startOpenVPN(msg, c)
		} else {
			d.sendMessage(messages.ErrorMsg("OpenVPN is already running"), c)
			return
		}
		return

	case consts.MsgDisconnect:
		if err := msg.EnsureEnoughArguments(0); err != nil {
			d.sendMessage(messages.ErrorMsg(err.Error()), c)
			return
		}
		if d.openvpn != nil {
			if err := d.openvpn.Process.Signal(os.Interrupt); err != nil {
				log.Println("Can't close openvpn")
				d.sendMessage(messages.ErrorMsg("Can't close openvpn"), c)
			}
		} else {
			d.sendMessage(messages.ErrorMsg("OpenVPN is not running"), c)
		}

	default:
		log.Printf("Unknown command: %v\n", msg.Command)
		d.sendMessage(messages.ErrorMsg(consts.UnknownCmd), c)
	}
}

func (d *Daemon) connectToMgmt() {
	time.Sleep(time.Second)
	c, err := net.Dial("unix", consts.MgmtSocket)
	if err != nil {
		log.Println("Can't connect to management socket")
		return
	}
	log.Println("Connected to management socket")
	defer c.Close()
	//"bytecount 1\n"
	if _, err := c.Write([]byte("state on\n")); err!= nil{
		log.Fatalf("Error: can't write to openvpn management\n")
	}

	scanner := bufio.NewScanner(c)
	for scanner.Scan() {
		log.Println("$$$ GOT: ", scanner.Text())
	}
	if err := scanner.Err(); err != nil{
		log.Printf("Management error: %v\n", err)
	}
}
