package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func signalListen(stop chan bool) {
	s := make(chan os.Signal)
	signal.Notify(s, syscall.SIGTERM)
	signal.Notify(s, syscall.SIGINT)
	<-s
	stop <- true
}

func main() {
	addr := "10.69.152.61:8080"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println(err)
	}

	h := &Handler{
		name: "test_server",
	}
	s := http.Server{
		Addr:    addr,
		Handler: h,
	}
	go s.Serve(l)
	stopCh := make(chan bool)
	go signalListen(stopCh)
	<-stopCh
}

type Handler struct {
	name string
}

func NewHandler(name string) *Handler {
	return &Handler{
		name: name,
	}
}

func (h *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fmt.Println(req.Host)
	rw.Write([]byte("testtesttest"))
	return
}
