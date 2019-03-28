package daemon

import (
	"github.com/TheWeirdDev/Vodga/utils/consts"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

type Daemon struct {
	quit  chan struct{}
	ln    net.Listener
	conns []net.Conn
	mux   sync.Mutex
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
			log.Fatalf("Server error: %v", err)
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

	default:
		log.Printf("Unknown command: %v\n", cmd)
		_, err := c.Write([]byte(consts.UnknownCmd))
		if err != nil {
			return err
		}
		return nil
	}
}
