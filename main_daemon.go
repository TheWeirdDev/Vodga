package main

import (
	"github.com/TheWeirdDev/Vodga/daemon"
	"log"
)

func main() {

	if err := daemon.InitDaemon(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Starts and waits for server to stop
	daemon.StartServer()
}
