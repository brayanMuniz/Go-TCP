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

func main() {
	// Connect to the TCP server at localhost:8080
	conn, err := net.Dial(TYPE, HOSTPORT)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	// Channel to signal when the client should close
	done := make(chan bool)

	// Goroutine to read from the server and print to the client
	go func() {
		for {
			// Read message from the server
			message, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil {
				fmt.Println("Error reading from server:", err)
				done <- true
				return
			}

			// Print the server message to the console
			fmt.Print("Message from server: ", message)
		}
	}()

	// Goroutine to read from stdin (keyboard) and send input to the server
	go func() {
		for {
			// Read user input from stdin
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("> ") // Display a prompt
			userInput, _ := reader.ReadString('\n')

			// Trim the input and send it to the server
			trimmedInput := strings.Trim(userInput, "\r\n")
			if trimmedInput == "" {
				continue // Ignore empty input
			}

			// Send the user input to the server
			_, err := fmt.Fprintln(conn, trimmedInput)
			if err != nil {
				fmt.Println("Error sending message:", err)
				done <- true
				return
			}

			// Exit client if user types "exit"
			if strings.ToLower(trimmedInput) == "exit" {
				done <- true
				return
			}
		}
	}()

	// Wait until the user types "exit" or there's an error
	<-done
	fmt.Println("Client exiting...")
}
