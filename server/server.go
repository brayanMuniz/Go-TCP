package main

import (
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
	Command string // First word of the payload
	Content string // Rest of the message (e.g., username or message content)

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

func (s *Server) parseMessage(msg *Message) {
	// Split the payload into words
	words := strings.Fields(string(msg.payload))

	// Check if there are at least two words
	if len(words) < 2 {
		msg.Command = ""
		msg.Content = ""
		return
	}

	// Set the first word as the command
	msg.Command = words[0]

	// The rest of the payload (everything after the first word) is the content
	firstSpaceIndex := strings.Index(string(msg.payload), " ")
	msg.Content = strings.TrimSpace(string(msg.payload)[firstSpaceIndex+1:])
}

// Function to extract the message without the command
func (msg *Message) getMessageWithoutCommand() string {
	// Convert the payload to a string
	messageText := string(msg.payload)

	// Find the position of the first space after the command (first word)
	firstSpaceIndex := strings.Index(messageText, " ")

	// Return everything after the first space (the actual message)
	if firstSpaceIndex != -1 && len(messageText) > firstSpaceIndex+1 {
		return strings.TrimSpace(messageText[firstSpaceIndex+1:])
	}
	return ""
}

func (s *Server) addClient(conn net.Conn) {
	addr := conn.RemoteAddr()
	s.clients[addr] = conn // Add the client connection to the map
	fmt.Printf("Client %s connected.\n", addr)
}

// NOTE: usually you would just return an err type, but for the sake of the assignment, this will do
func (s *Server) addUserNameToClient(msg *Message) int {
	userName := msg.Content

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

	return -1 // Success
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

func (s *Server) handleMessage(msg *Message) {
	// Parse the message into command and username
	s.parseMessage(msg)

	switch msg.Command {
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
			var userList []string
			for username := range s.userNames {
				userList = append(userList, username)
			}
			numberOfUsers := len(s.userNames)

			// Format the message as a string (could also use JSON if needed)
			userListMessage := fmt.Sprintf("%d %v\n", numberOfUsers, userList)

			// Send the message to the current user
			msg.conn.Write([]byte(userListMessage))

			// Send message to all other users that username has joined chat.
			newUserMessage := fmt.Sprintf("%s has joined the chat", msg.Content)
			for _, conn := range s.clients {
				if conn != msg.conn {
					conn.Write([]byte(newUserMessage))
				}
			}

		}

	case "MESG":
		// Find the username from the map using the connection
		var username string
		for name, conn := range s.userNames {
			if conn == msg.conn {
				username = name
				break
			}
		}

		if username != "" {
			newUserMessage := fmt.Sprintf("%s: %s\n", username, msg.getMessageWithoutCommand())
			for _, conn := range s.clients {
				if conn != msg.conn {
					conn.Write([]byte(newUserMessage))
				}
			}
		} else {
			msg.conn.Write([]byte("Error: Username not found.\n"))
		}

	case "PMSG":
		fmt.Printf("Personal Message")
	case "EXIT":
		fmt.Printf("EXIT")
	default:
		// Handle unknown commands
		fmt.Printf("Unknown command received from %s: %s\n", msg.from, msg.from)
	}
}

func main() {
	server := NewServer()
	go func() {
		for msg := range server.messageChannel {
			fmt.Printf("Received Message from %s: %s \n", msg.from, msg.payload)
			server.handleMessage(&msg)
		}
	}()

	log.Fatal(server.Start())
}
