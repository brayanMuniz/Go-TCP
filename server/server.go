package main

import (
	// "bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

const (
	HOSTPORT = "localhost:8080"
	TYPE     = "tcp"

	// Error codes
	ERR_USERNAME_TAKEN           = 0
	ERR_USERNAME_TOO_LONG        = 1
	ERR_USERNAME_CONTAINS_SPACES = 2
	ERR_UNKNOWN_USER_PRIVATE_MSG = 3
	ERR_UNKNOWN_MESSAGE_FORMAT   = 4
)

type Message struct {
	from    string
	payload []byte
	conn    net.Conn
}

type Server struct {
	listenPort     string
	ln             net.Listener
	quitChannel    chan struct{}
	messageChannel chan Message
	clients        map[net.Addr]net.Conn // key value pairs
	userNames      map[string]net.Conn   // username (key) to the client connection (value)
}

func NewServer() *Server {
	return &Server{
		listenPort:     HOSTPORT,
		quitChannel:    make(chan struct{}),
		messageChannel: make(chan Message, 10),
		clients:        make(map[net.Addr]net.Conn),
		userNames:      make(map[string]net.Conn),
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

		fmt.Println("New connection to the server", conn.RemoteAddr())

		// Add the new connection to the clients map
		s.addClient(conn)

		// create a go routine to read the message from the client.
		go func(c net.Conn) {
			// NOTE: this closes the connection when the go routine ends.
			defer s.removeClient(c)
			s.readLoop(c)
		}(conn)

	}

}

func (s *Server) addClient(conn net.Conn) {
	addr := conn.RemoteAddr()
	s.clients[addr] = conn // Add the client connection to the map
	fmt.Printf("Client %s connected.\n", addr)
}

// NOTE: usually you would just return an err type, but for the sake of the assignment, this will do
func (s *Server) addUserNameToClient(msg Message) int {
	// Split the payload into words
	words := strings.Fields(string(msg.payload))

	// Check if there are at least two words
	if len(words) < 2 {
		return ERR_UNKNOWN_MESSAGE_FORMAT // Return an error if the format is incorrect
	}

	// Find the position of the first space after the first word to slice the remaining part
	messageText := string(msg.payload)
	firstSpaceIndex := strings.Index(messageText, " ")

	// The username is everything after the first word and the space
	userName := strings.TrimSpace(messageText[firstSpaceIndex+1:])
	fmt.Println(userName)

	// Check if the username is too long
	if len(userName) > 20 {
		return ERR_USERNAME_TOO_LONG
	}

	// Check if the username contains spaces
	if strings.Contains(userName, " ") {
		return ERR_USERNAME_CONTAINS_SPACES
	}

	// Check if the username is already taken
	if _, taken := s.userNames[userName]; taken {
		return ERR_USERNAME_TAKEN
	}

	// Add the username to the map, using the connection from the message
	s.userNames[userName] = msg.conn
	fmt.Printf("Added %s to %s.\n", userName, msg.from)

	return -1
}

func (s *Server) removeClient(conn net.Conn) {
	addr := conn.RemoteAddr()
	delete(s.clients, addr) // Remove the client from the map
	fmt.Printf("Client %s disconnected.\n", addr)
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

		s.messageChannel <- Message{
			from:    conn.RemoteAddr().String(),
			payload: buffer[:n],
			conn:    conn,
		}
	}
}

func (s *Server) handleMessage(msg Message) {
	// Convert message payload to a string
	messageText := string(msg.payload)

	// Split the message into words to extract the first word as a command
	words := strings.Fields(messageText)
	if len(words) == 0 {
		return // No command found, so skip processing
	}

	// Extract the first word (command)
	command := words[0]

	switch command {

	case "REG":
		fmt.Printf("Handling username for %s\n", msg.from)
		errCode := s.addUserNameToClient(msg)
		if errCode != -1 {
			var errMsg string
			switch errCode {
			case ERR_USERNAME_TAKEN:
				errMsg = "Username is already taken."
			case ERR_USERNAME_TOO_LONG:
				errMsg = "Username is too long."
			case ERR_USERNAME_CONTAINS_SPACES:
				errMsg = "Username contains spaces."
			case ERR_UNKNOWN_MESSAGE_FORMAT:
				errMsg = "Unknown message format."
			default:
				errMsg = "An unknown error occurred."
			}

			// Send the error message back to the client
			msg.conn.Write([]byte(fmt.Sprintf("Error: %s\n", errMsg)))
		} else {
			msg.conn.Write([]byte("Username registered successfully.\n"))
		}
	case "MESG":
		fmt.Printf("Broadcasting message from %s\n", msg.from)
	case "PMSG":
		fmt.Printf("Personal Message")
	case "EXIT":
		fmt.Printf("EXIT")
	default:
		// Handle unknown commands
		fmt.Printf("Unknown command received from %s: %s\n", msg.from, messageText)
	}
}

func main() {
	server := NewServer()
	go func() {
		for msg := range server.messageChannel {
			fmt.Printf("Received Message from %s: %s \n", msg.from, msg.payload)
			server.handleMessage(msg)
		}
	}()

	log.Fatal(server.Start())
}
