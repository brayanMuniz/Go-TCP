# Go-TCP

This project is a simple TCP-based chat application that allows multiple clients to connect to a server, register a username, and send messages. It supports both public messages (broadcast to all users) and private messages between two users.

## Features
- **User Registration**: Clients register with a unique username.
- **Public Messages**: Clients can send public messages that are broadcast to all connected users.
- **Private Messages**: Clients can send private messages to specific users.
- **Graceful Exit**: Clients can gracefully disconnect from the server by sending an "EXIT" message.
- **Error Handling**: The server handles errors such as duplicate usernames, invalid message formats, and unknown users.

## Requirements
- Go programming language (1.16+)

## How to Run

### 1. Start the Server
To run the server, execute the following command:

`go run server.go`

The server will start on port `8080` and listen for incoming connections.

### 2. Run the Client
To run a client and connect to the server, execute the following command:

`go run client.go <server_address>`

Replace `<server_address>` with the IP address of the server.

For example, if the server is running locally:

go run client.go 127.0.0.1

### 3. Client Registration
After starting the client, you will be prompted to enter a username. The username must meet the following requirements:
- Must be unique (not already in use by another client).
- Must not exceed 20 characters.
- Must not contain spaces.

Once a valid username is provided, the server will confirm the registration.

### 4. Sending Messages

- **Public Messages**: To send a public message, simply type the message and press `Enter`. All connected users will receive the message.
  
- **Private Messages**: To send a private message to a specific user, use the following format:

  PMSG <username> <message>

  Replace `<username>` with the recipient's username and `<message>` with the content of the private message. Only the specified user will receive the message.

### 5. Exit
To gracefully disconnect from the server, type:

EXIT

The server will confirm the disconnection, and the client will exit.

## Error Messages
The server will return the following error messages in case of invalid input:
- `ERR 0`: Username already taken.
- `ERR 1`: Username is too long (exceeds 20 characters).
- `ERR 2`: Username contains spaces.
- `ERR 3`: Unknown user when attempting to send a private message.
- `ERR 4`: Invalid message format.

## Project Structure

- `client.go`: Handles the client-side functionality, including connecting to the server, user registration, sending and receiving messages.
- `server.go`: Manages the server-side functionality, including accepting client connections, handling user registrations, broadcasting messages, and managing private messages.

## Example Usage

### Public Message
John: Hello everyone!

### Private Message
PMSG Jane How are you?

### Exit
EXIT
