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

type Client struct {
	conn     net.Conn
	username string
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Connect() error {
	conn, err := net.Dial(TYPE, HOSTPORT)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *Client) Register(username string) error {
	c.username = username
	_, err := c.conn.Write([]byte("REG " + strings.TrimSpace(username) + "\n")) // Trim whitespace
	if err != nil {
		return err
	}
	return nil
}

// Read a single response from the server
func (c *Client) ReadResponse() (string, error) {
	buffer := make([]byte, 2048)
	n, err := c.conn.Read(buffer)
	if err != nil {
		return "", err
	}
	return string(buffer[:n]), nil
}

// Read messages from the server in a loop
func (c *Client) ReadLoop() {
	for {
		// Read the response from the server
		buffer := make([]byte, 2048)
		n, err := c.conn.Read(buffer)
		if err != nil {
			// NOTE: This might be the worst way to handle this, but it gets the job done.
			if strings.Contains(err.Error(), "use of closed network connection") || strings.Contains(err.Error(), "EOF") {
				fmt.Println("Connection closed. Exiting read loop.")
				return
			}
			fmt.Println("Error reading from server:", err)
			break
		}

		// Convert the buffer to a string and print it
		serverMessage := string(buffer[:n])
		fmt.Println(serverMessage)
	}
}

// Write messages to the server
func (c *Client) WriteLoop() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		message := scanner.Text()
		if strings.TrimSpace(message) == "" {
			continue
		}

		// Send the message to the server
		_, err := c.conn.Write([]byte(message + "\n"))
		if err != nil {
			fmt.Println("Error sending message:", err)
			break
		}

		// If the client sends "EXIT", close the connection
		if strings.HasPrefix(message, "EXIT") {
			// Wait for server response (ACK)
			response, err := c.ReadResponse()
			if err != nil {
				// WARNING:
				// fmt.Println("Failed to receive response:", err)
				return
			}

			fmt.Println(response)

			c.Close() // Immediately close the connection when "EXIT" is sent
			return    // Break out of the loop and return
		}
	}
}

// Close the connection when finished
func (c *Client) Close() {
	c.conn.Close()
}

func main() {
	client := NewClient()

	// Connect to the server
	err := client.Connect()
	if err != nil {
		fmt.Println("Failed to connect to the server:", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)

	// Registration loop
	for {
		fmt.Print("Enter your username: ")
		scanner.Scan()
		username := scanner.Text()

		err = client.Register(username)
		if err != nil {
			fmt.Println("Failed to register username:", err)
			return
		}

		// Wait for server response
		response, err := client.ReadResponse()
		if err != nil {
			fmt.Println("Failed to receive response:", err)
			return
		}

		// Handle server response
		if strings.HasPrefix(response, "ERR") {
			if strings.Contains(response, "0") {
				fmt.Println("Error: Username already taken. Please try a different username.")
			} else if strings.Contains(response, "1") {
				fmt.Println("Error: Username too long. Please enter a username shorter than 20 characters.")
			} else if strings.Contains(response, "2") {
				fmt.Println("Error: Username contains spaces. Please enter a username without spaces.")
			} else {
				fmt.Println("Unknown error. Please try again.")
			}
			// Prompt for a new username
			continue
		} else {
			// Registration successful
			fmt.Println(response) // Print any welcome message from the server
			break
		}
	}

	// Start reading and writing concurrently
	go client.ReadLoop()
	client.WriteLoop()

	// When WriteLoop ends (EXIT), the program terminates
	fmt.Println("Disconnected from server.")
}
