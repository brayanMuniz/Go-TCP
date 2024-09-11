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
			if err.Error() == "EOF" {
				// Client closed the connection, handle it
				fmt.Println("Client closed the connection.")
				break
			}
			fmt.Println("Read error:", err)
			break
		}

		// Process the message from the client
		s.messageChannel <- Message{
			from:    conn.RemoteAddr().String(),
			payload: buffer[:n],
			conn:    conn,
		}
	}
}

func getUserNameFromConnection(s *Server, c net.Conn) string {
	var username string
	for name, conn := range s.userNames {
		if conn == c {
			username = name
			break
		}
	}
	return username
}

func getConnFromUserName(s *Server, username string) (net.Conn, bool) {
	conn, exists := s.userNames[username]
	return conn, exists
}

func getFirstWord(m string) string {
	// Convert the payload to a string and split it by spaces
	words := strings.Fields(string(m))
	if len(words) > 0 {
		return words[0] // Return the first word (username)
	}
	return "" // Return an empty string if no words found
}

func removeFirstWord(m string) string {
	// Find the position of the first space
	firstSpaceIndex := strings.Index(m, " ")

	// If there is no space, return an empty string (meaning no other words)
	if firstSpaceIndex == -1 {
		return ""
	}

	// Return the part of the string after the first word (everything after the first space)
	return strings.TrimSpace(m[firstSpaceIndex+1:])
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
				errMsg = "0"
			case ERR_USERNAME_TOO_LONG:
				errMsg = "1"
			case ERR_USERNAME_CONTAINS_SPACES:
				errMsg = "2"
			case ERR_UNKNOWN_MESSAGE_FORMAT:
				errMsg = "4"
			default:
				errMsg = "4"
			}

			// Send the error message back to the client
			msg.conn.Write([]byte(fmt.Sprintf("ERR: %s\n", errMsg)))
		} else {
			var userList []string
			for username := range s.userNames {
				userList = append(userList, username)
			}
			numberOfUsers := len(s.userNames)

			// Format the message as a string
			userListMessage := fmt.Sprintf("%d %v\n", numberOfUsers, userList)

			// Send the message to the current user
			msg.conn.Write([]byte(userListMessage))

			// Send message to all other users that username has joined chat.
			newUserMessage := fmt.Sprintf("%s has joined the chat\n", msg.Content)
			for _, conn := range s.clients {
				if conn != msg.conn {
					conn.Write([]byte(newUserMessage))
				}
			}

		}

	case "MESG":
		username := getUserNameFromConnection(s, msg.conn)

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
		// x -> y

		// get x username based on their connection
		x_username := getUserNameFromConnection(s, msg.conn)
		if x_username == "" {
			msg.conn.Write([]byte("Error: Username not found.\n"))
			return
		}
		// get y the connection by getting the mapping of the user
		// username message <- msg.getMessageWithoutCommand()
		y_username := getFirstWord(msg.getMessageWithoutCommand())
		y_conn, exist := getConnFromUserName(s, y_username)
		if !exist {
			msg.conn.Write([]byte(fmt.Sprintf("ERR %s\n", "3")))
			return
		}

		// send message to user, including the user name of the client
		// Get the actual message without the command and username
		actualMessage := removeFirstWord(msg.getMessageWithoutCommand())

		// Send the personal message to the specific user
		y_conn.Write([]byte(fmt.Sprintf("Private message from %s: %s\n", x_username, actualMessage)))

	case "EXIT":
		// NOTE: I am not going to be using the username message that the client sends, i am going to be using the connection and the mappings to make sure it exist.

		// EXIT username
		// make sure the username exist
		username := getUserNameFromConnection(s, msg.conn)
		if username == "" {
			msg.conn.Write([]byte("Error: Username not found.\n"))
			return
		}

		// deregister username
		delete(s.userNames, username)

		// send ACK to client
		var userList []string
		for username := range s.userNames {
			userList = append(userList, username)
		}
		numberOfUsers := len(s.userNames)

		userListMessage := fmt.Sprintf("%d %v\n", numberOfUsers, userList)

		msg.conn.Write([]byte(userListMessage))

		// brodcast that username has left the chat
		newUserMessage := fmt.Sprintf("%s has left the chat\n", username)
		for _, conn := range s.clients {
			if conn != msg.conn {
				conn.Write([]byte(newUserMessage))
			}
		}

		// NOTE: do not need to remove the client connection, since it will be dealth with in the defer acceptloop

		// Close the connection
		msg.conn.Close()

	default:
		msg.conn.Write([]byte(fmt.Sprintf("ERR: %s\n", "4")))

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
