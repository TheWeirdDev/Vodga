package daemon

import (
	"bufio"
	"fmt"
	"github.com/TheWeirdDev/Vodga/utils"
	"github.com/TheWeirdDev/Vodga/utils/consts"
	"io"
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
	for {
		buf := make([]byte, 1024)
		nr, err := c.Read(buf)

		// Returns EOF when disconnected
		if err == io.EOF {
			log.Printf("Client #%d disconnected", id)
			break
		} else if err != nil {
			log.Fatalf("Server error: %v", err)
		}

		cmd := strings.TrimSpace(string(buf[0:nr]))
		log.Printf("Server got(#%d): %s\n", id, cmd)
		if err := d.processCommand(cmd, c); err != nil {
			log.Printf("Error: %v", err)
		}
	}
}

func (d *Daemon) stopServer(c net.Conn) error {
	_, err := c.Write([]byte(consts.MsgKilled))
	if err != nil {
		return err
	}
	close(d.quit)
	return d.ln.Close()
}

func (d *Daemon) startOpenVPN(config string, c net.Conn) {
	cmd := exec.Command("openvpn", "--config", config,
		"--management", consts.MgmtSocket, "unix", "--management-query-passwords")

	// create a pipe for the output of the script
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		_, err := c.Write([]byte(fmt.Sprintf("%s %s", consts.MsgError, "Can't open StdoutPipe for OpenVPN")))
		if err != nil {
			log.Println("Error: can't write to connection")
		}
		return
	}

	scanner := bufio.NewScanner(cmdReader)
	//TODO: scanner.Split()

	go func() {
		for scanner.Scan() {
			c.Write([]byte("LOG " + scanner.Text() + "\n"))
		}
	}()

	err = cmd.Start()
	if err != nil {
		_, err := c.Write([]byte(fmt.Sprintf("%s %s", consts.MsgError, "Can't start OpenVPN")))
		if err != nil {
			log.Println("Error: can't write to connection")
		}
		return
	}
	go d.connectMgmt()
	d.openvpn = cmd

	err = cmd.Wait()
	if err != nil {
		d.openvpn = nil
		log.Println("OpenVPN closed unexpectedly")
		_, err := c.Write([]byte(consts.MsgDisconnected))
		if err != nil {
			log.Println("Error: can't write to connection")
		}
		return
	}

	d.openvpn = nil
	log.Println("OpenVPN Closed")
	_, err = c.Write([]byte(consts.MsgDisconnected))
	if err != nil {
		log.Println("Error: can't write to connection")
	}

}

func (d *Daemon) processCommand(cmd string, c net.Conn) error {
	d.mux.Lock()
	defer d.mux.Unlock()

	fields := strings.Fields(cmd)
	if len(fields) < 1 {
		return nil
	}
	cmd = fields[0]
	switch cmd {
	case consts.MsgStop:
		return d.stopServer(c)

	case consts.MsgConnect:
		if err := utils.EnsureEnoughArguments(fields, 1); err != nil {
			_, err := c.Write([]byte(fmt.Sprintf("%s %s", consts.MsgError, err.Error())))
			return err
		}
		if d.openvpn == nil {
			go d.startOpenVPN(fields[1], c)
		} else {
			_, err := c.Write([]byte(fmt.Sprintf("%s %s", consts.MsgError, "OpenVPN is already running")))
			return err
		}
		return nil

	case consts.MsgDisconnect:
		if err := utils.EnsureEnoughArguments(fields, 0); err != nil {
			_, err := c.Write([]byte(fmt.Sprintf("%s %s", consts.MsgError, err.Error())))
			return err
		}
		if d.openvpn != nil {
			return d.openvpn.Process.Signal(os.Interrupt)
		} else {
			_, err := c.Write([]byte(fmt.Sprintf("%s %s", consts.MsgError, "OpenVPN is not running")))
			return err
		}

	default:
		log.Printf("Unknown command: %v\n", cmd)
		_, err := c.Write([]byte(consts.UnknownCmd))
		if err != nil {
			return err
		}
		return nil
	}
}

func (d *Daemon) connectMgmt() {
	time.Sleep(time.Second)
	c, err := net.Dial("unix", consts.MgmtSocket)
	if err != nil {
		log.Println("Can't connect to management socket")
		return
	}
	log.Println("Connected to management socket")
	defer c.Close()
	//_, err = c.Write([]byte("bytecount 1\n"))
	_, err = c.Write([]byte("state on\n"))
	if err != nil {
		log.Println("ERROR")
		return
	}
	buf := make([]byte, 1024)
	for {
		n, err := c.Read(buf[:])
		if err != nil {
			return
		}
		log.Println("$$$ GOT: ", string(buf[0:n]))
	}

}
