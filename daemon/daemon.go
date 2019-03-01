package main

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
	UnixSocket = "/tmp/vodgad.sock"
	MsgKilled  = "KILLED"
)

var InstanceExists = errors.New("another vodga instance is running.\n" +
	"Kill the process or remove the socket: " + UnixSocket)
var (
	quit chan bool
	ln   net.Listener
)

func main() {
	quit = make(chan bool, 1)

	if err := checkUser(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	err := checkExistingInstance()
	checkArgs(err)

	log.Println("Starting vodga daemon")
	ln, err = net.Listen("unix", UnixSocket)

	if err != nil {
		log.Fatalf("listen error: %v", err)
	}

	if err := os.Chmod(UnixSocket, os.FileMode(0777)); err != nil {
		log.Fatalf("Socket permission error: %v", err)
	}

	go func(ln net.Listener, kill *chan bool) {
		sigchan := make(chan os.Signal, 5)
		signal.Notify(sigchan, os.Interrupt, syscall.SIGPIPE, syscall.SIGKILL,
			syscall.SIGTERM, syscall.SIGQUIT)
		err := <-sigchan
		log.Printf("Server Killed by: %v", err)
		*kill <- true
		ln.Close()
	}(ln, &quit)

loop:
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-quit:
				{
					log.Println("Server stopped")
					break loop
				}
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
		_, err := c.Write([]byte(MsgKilled))
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
			} else {
				log.Println("No existing instance found")
			}
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

	//fmt.Println("command:", *wordPtr)
	//fmt.Println("numb:", *numbPtr)
	//fmt.Println("fork:", *boolPtr)
	//fmt.Println("svar:", svar)
	//fmt.Println("tail:", flag.Args())
}

func stopExistingServer() error {
	c, err := net.Dial("unix", UnixSocket)
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

	if string(buf[0:n]) == MsgKilled {
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
	if _, err := os.Stat(UnixSocket); err == nil {
		return InstanceExists
	} else if os.IsNotExist(err) {
		return nil
	} else {
		return err
	}
}
