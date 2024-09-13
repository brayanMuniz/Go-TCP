package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	HOSTPORT = "localhost:8080"
	TYPE     = "tcp"
)

// Read user input from stdin and return the trimmed string
func readUserInput() string {
	reader := bufio.NewReader(os.Stdin)
	userInput, err := reader.ReadString('\n')

	if err != nil {
		fmt.Println("Error reading input:", err)
		os.Exit(1) // Exit if there's an error reading input
	}

	trimmedInput := strings.TrimSpace(userInput)
	return trimmedInput
}

func connectToServer(userName string) {
	// get the TCP ADDR
	tcpServer, err := net.ResolveTCPAddr(TYPE, HOSTPORT)
	if err != nil {
		println("ResolveTCPAddr failed:", err.Error())
		os.Exit(1)
	}

	// connect to server
	conn, err := net.DialTCP(TYPE, nil, tcpServer)
	if err != nil {
		println("Dial failed:", err.Error())
		os.Exit(1)
	}

	// Ensure connection is closed when function returns
	defer conn.Close()

	// register with username
	registerString := fmt.Sprintf("REG %s", userName)
	_, err = conn.Write([]byte(registerString))
	if err != nil {
		println("Write data failed:", err.Error())
		os.Exit(1)
	}

	// buffer to get data
	received := make([]byte, 1024)
	_, err = conn.Read(received)
	if err != nil {
		println("Read data failed:", err.Error())
		os.Exit(1)
	}

	println("Received message:", string(received))

	// Loop to keep the connection open and allow further interaction
	for {
		fmt.Println("Enter a message to send to the server or type 'exit' to close the connection:")
		message := readUserInput()

		if message == "EXIT" {
			fmt.Println("Closing connection...")
			break
		}

		_, err = conn.Write([]byte(message))
		if err != nil {
			println("Write data failed:", err.Error())
			break
		}

		received := make([]byte, 1024)
		_, err = conn.Read(received)
		if err != nil {
			println("Read data failed:", err.Error())
			break
		}

		println("Received from server:", string(received))
	}

}

func main() {
	// Ask for the username before making the TCP connection
	fmt.Println("Enter your username: ")
	userName := readUserInput()

	// Establish the TCP connection and send the username
	connectToServer(userName)
}

