package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/TheWeirdDev/Vodga/daemon"
	"github.com/TheWeirdDev/Vodga/utils/consts"
	"github.com/TheWeirdDev/Vodga/utils/messages"
	"log"
	"net"
	"os"
	"os/user"
)

var InstanceExists = errors.New("another vodga instance is running.\n" +
	"Kill the process or remove the socket: " + consts.UnixSocket)

func main() {
	// Daemon needs root privileges to run
	if err := checkUser(); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}

	// Only one instance may run at the same time
	err := checkExistingInstance()
	// Check all the command line arguments to decide what to do next
	if err := checkArgs(err); err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Println("Vodga daemon is running")

	server := daemon.NewDaemon()
	// Starts and waits for server to stop
	server.StartServer()
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

	data, err := json.Marshal(&messages.Message{Command:consts.MsgStop})
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = c.Write(data)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(c)
	if scanner.Scan() {
		s := scanner.Text()
		msg := &messages.Message{}
		err = json.Unmarshal([]byte(s), msg)
		if msg.Command == "" {
			return errors.New("returned command is a empty")
		}
		if msg.Command == consts.MsgKilled {
			log.Println("Server stopped")
			return nil
		}
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
