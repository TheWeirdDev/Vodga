package daemon

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"syscall"
)

const (
	unixSocket = "/tmp/vodgad.sock"
	msgKilled  = "KILLED"
)

var InstanceExists = errors.New("another vodga instance is running.\n" +
	"Kill the process or remove the socket: " + unixSocket)
var (
	quit chan bool
	ln   net.Listener
)

// Check for all the requirements for starting the server
func InitDaemon() error {
	quit = make(chan bool, 1)

	// Daemon needs root privileges to run
	if err := checkUser(); err != nil {
		return err
	}

	// Only one instance may run at the same time
	err := checkExistingInstance()
	// Check all the command line arguments to decide what to do next
	checkArgs(err)

	log.Println("Starting vodga daemon")
	ln, err = net.Listen("unix", unixSocket)

	if err != nil {
		return fmt.Errorf("listen error: %v", err)
	}

	// Make the socket accessible to users other than root
	if err := os.Chmod(unixSocket, os.FileMode(0777)); err != nil {
		return fmt.Errorf("socket permission error: %v", err)
	}
	return nil
}

func StartServer() {

	// Listen for OS signals and close the socket before exiting
	go func(ln net.Listener, kill *chan bool) {
		sigchan := make(chan os.Signal, 5)
		signal.Notify(sigchan, os.Interrupt, syscall.SIGPIPE, syscall.SIGKILL,
			syscall.SIGTERM, syscall.SIGQUIT)
		// Waits for signals
		err := <-sigchan
		log.Printf("Server Killed by: %v", err)
		*kill <- true
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
				{
					log.Println("Server stopped")
					break loop
				}
			// Otherwise the error is unknown and needs to be handled
			default:
				log.Printf("Accept error: %+v", err)
				break loop
			}
		}
		go daemonServer(conn)
	}
}

func daemonServer(c net.Conn) {
	for {
		buf := make([]byte, 1024)
		nr, err := c.Read(buf)
		if err != nil {
			return
		}

		cmd := string(buf[0:nr])
		log.Println("Server got:", cmd)
		processCommand(cmd, c)
	}
}

func processCommand(cmd string, c net.Conn) {
	if cmd == "STOP_SERVER" {
		_, err := c.Write([]byte(msgKilled))
		if err != nil {
			log.Fatal("Writing client error: ", err)
		}
		quit <- true
		ln.Close()
	}
}

func checkArgs(err error) {

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
					log.Fatalf("Error: %v", err)
				}
			} else if err != nil {
				log.Fatalf("Error : %v", err)
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
			log.Fatalf("Error: %v", err)
		}

	default:
		log.Printf("Unknown command: %s", *command)
		os.Exit(0)
	}

	//fmt.Println("tail:", flag.Args())
}

func stopExistingServer() error {
	c, err := net.Dial("unix", unixSocket)
	if err != nil {
		log.Fatal("Dial error", err)
	}
	defer c.Close()

	const msg = "STOP_SERVER"
	_, err = c.Write([]byte(msg))
	if err != nil {
		log.Fatal("Write error:", err)
	}
	log.Println("Client sent:", msg)

	buf := make([]byte, 1024)
	n, err := c.Read(buf[:])
	if err != nil {
		return err
	}

	if string(buf[0:n]) == msgKilled {
		log.Println("Server stopped")
		return nil
	}
	return fmt.Errorf("can't stop the server")
}

func checkUser() error {
	theUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get user details")
	}
	if theUser.Uid != "0" {
		return fmt.Errorf("the Vodga daemon needs root privileges to run")
	}
	return err
}

func checkExistingInstance() error {
	if _, err := os.Stat(unixSocket); err == nil {
		return InstanceExists
	} else if os.IsNotExist(err) {
		return nil
	} else {
		return err
	}
}
