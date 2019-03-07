GOBASE=$(shell pwd)
GOPATH=$(GOBASE)/vendor:$(GOBASE):/home/alireza/code/golang # You can remove or change the path after last colon.
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)


compile-daemon:
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go build -o daemon.out main_daemon.go
