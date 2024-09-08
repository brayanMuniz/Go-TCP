package main

import (
	// "bufio"
	"fmt"
	"log"
	"net"
	"os"
)

const (
	HOSTPORT = "localhost:8080"
	TYPE     = "tcp"
)

type Server struct {
	listenPort     string
	ln             net.Listener
	quitChannel    chan struct{}
	messageChannel chan []byte
}

func NewServer() *Server {
	return &Server{
		listenPort:     HOSTPORT,
		quitChannel:    make(chan struct{}),
		messageChannel: make(chan []byte, 10),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen(TYPE, HOSTPORT)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	s.ln = ln

	go s.acceptLoop()

	// This prevents the server from closing, unless it is explicitly called
	<-s.quitChannel
	close(s.messageChannel) // notify the clients that the message channels are closed.

	// NOTE: these next 2 lines never execute, unless we explicitly close the server
	defer ln.Close() // close the server when Start function completes
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			continue
		}

		// create a go routine to handle the read
		// NOTE: think of them as lightweight concurrent routines
		go s.readLoop(conn)
	}

}

func (s *Server) readLoop(conn net.Conn) {
	defer conn.Close()
	buffer := make([]byte, 2048)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("read error:", err)
			continue
		}
		s.messageChannel <- buffer[:n]
	}
}

func main() {
	server := NewServer()
	go func() {
		for msg := range server.messageChannel {
			fmt.Println("received message: ", string(msg))

		}

	}()

	log.Fatal(server.Start())
}

// continue video at: https://youtu.be/qJQrrscB1-4?si=qCL1DOGhvm0stPDH&t=804
// Add a message structure
// be able to map the who is currently connected.
// https://www.golinuxcloud.com/golang-tcp-server-client/
