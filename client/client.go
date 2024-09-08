package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

const (
	HOSTPORT = "localhost:8080"
	TYPE     = "tcp"
)

func main() {
	// Connect to tcp server at localhost:8080
	conn, err := net.Dial(TYPE, HOSTPORT)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	// Send a message to the server
	fmt.Fprintln(conn, "Hello from client!")

	// Receive server response
	message, _ := bufio.NewReader(conn).ReadString('\n')
	fmt.Print("Message from server: ", message)

	// Keep the client running until a key is pressed
	fmt.Println("Press Enter to exit...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
