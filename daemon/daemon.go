package daemon

import (
	"errors"
	"flag"
	"fmt"
	"github.com/TheWeirdDev/Vodga/utils/consts"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"syscall"
)

var InstanceExists = errors.New("another vodga instance is running.\n" +
	"Kill the process or remove the socket: " + consts.UnixSocket)

type CommandFunc func(net.Conn) error

var (
	quit     chan struct{}
	ln       net.Listener
	commands = map[string]CommandFunc{
		consts.MsgStop: stopServer,
	}
)

// Check for all the requirements for starting the server
func InitDaemon() error {
	quit = make(chan struct{})

	// Daemon needs root privileges to run
	if err := checkUser(); err != nil {
		return err
	}

	// Only one instance may run at the same time
	err := checkExistingInstance()
	// Check all the command line arguments to decide what to do next
	if err := checkArgs(err); err != nil {
		return err
	}

	log.Println("Starting vodga daemon")
	ln, err = net.Listen("unix", consts.UnixSocket)

	if err != nil {
		return err
	}

	// Make the socket accessible to users other than root
	if err := os.Chmod(consts.UnixSocket, os.FileMode(0777)); err != nil {
		return err
	}
	return nil
}

func StartServer() {

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
	}(ln, &quit)

loop:
	for {
		// Waits for new client connection
		conn, err := ln.Accept()
		if err != nil {
			select {

			// If we write something to this channel,
			// it means server stopped for a known reason
			case <-quit:
				log.Println("Server stopped")

				// Otherwise the error is unknown and needs to be handled
			default:
				log.Printf("Accept error: %+v", err)
			}
			break loop
		}
		go daemonServer(conn)
	}
}

func daemonServer(c net.Conn) {
	for {
		buf := make([]byte, 1024)
		nr, err := c.Read(buf)

		// Returns EOF when disconnected
		if err == io.EOF {
			log.Println("Client disconnected")
			break
		} else if err != nil {
			log.Fatalf("Server error: %v", err)
		}

		cmd := string(buf[0:nr])
		log.Println("Server got:", cmd)
		if err := processCommand(cmd, c); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}

func stopServer(c net.Conn) error {
	_, err := c.Write([]byte(consts.MsgKilled))
	if err != nil {
		return err
	}
	close(quit)
	return ln.Close()
}

func processCommand(cmd string, c net.Conn) error {
	fn, ok := commands[cmd]
	if !ok {
		log.Printf("Got unknown command: %v\n", cmd)
		_, err := c.Write([]byte("Unknown command"))
		if err != nil {
			return err
		}
		return nil
	}
	return fn(c)
}

func checkArgs(err error) error {

	command := flag.String("command", "start", "start or stop the daemon")
	//numbPtr := flag.Int("numb", 42, "an int")
	//boolPtr := flag.Bool("fork", false, "a bool")

	//var svar string
	//flag.StringVar(&svar, "svar", "bar", "a string var")

	flag.Parse()

	shouldExit := true
	switch *command {

	case "restart":
		{
			shouldExit = false
		}
		fallthrough
	case "stop":
		{
			if err == InstanceExists {
				if err := stopExistingServer(); err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else {
				log.Println("No existing instance found")
			}
			// Shouldn't exit if restart requested
			if shouldExit {
				os.Exit(0)
			}
		}

	case "start":
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown command \"%s\"", *command)
	}
	return nil
	//fmt.Println("tail:", flag.Args())
}

func stopExistingServer() error {
	c, err := net.Dial("unix", consts.UnixSocket)
	if err != nil {
		return err
	}
	defer c.Close()

	_, err = c.Write([]byte(consts.MsgStop))
	if err != nil {
		return err
	}

	buf := make([]byte, 1024)
	n, err := c.Read(buf[:])
	if err != nil {
		return err
	}

	if string(buf[0:n]) == consts.MsgKilled {
		log.Println("Server stopped")
		return nil
	}
	return errors.New("can't stop the server")
}

func checkUser() error {
	theUser, err := user.Current()
	if err != nil {
		return errors.New("failed to get user details")
	}
	if theUser.Uid != "0" {
		return errors.New("the Vodga daemon needs root privileges to run")
	}
	return err
}

func checkExistingInstance() error {
	if _, err := os.Stat(consts.UnixSocket); err == nil {
		return InstanceExists
	} else if os.IsNotExist(err) {
		return nil
	} else {
		return err
	}
}
