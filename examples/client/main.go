package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	snet "snet/go"
)

func main() {
	config := snet.Config{
		EnableCrypt:        false, // Must match server settings
		HandshakeTimeout:   time.Second * 10,
		RewriterBufferSize: 1024,
		ReconnWaitTimeout:  time.Second * 30,
	}

	// Connect to server
	conn, err := snet.Dial(config, func() (net.Conn, error) {
		return net.Dial("tcp", "localhost:8081")
	})
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	fmt.Println("Successfully connected to SNET server")
	fmt.Printf("Local address: %s\n", conn.LocalAddr().String())
	fmt.Printf("Server address: %s\n", conn.RemoteAddr().String())

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	scanner := bufio.NewScanner(os.Stdin)

	// Start goroutine to receive messages
	go func() {
		for {
			message, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("Failed to read server message: %v\n", err)
				return
			}
			fmt.Printf("Server: %s", message)
		}
	}()

	fmt.Println("Enter messages to send to server (type 'quit' to exit):")

	// Message sending loop
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		message := scanner.Text()
		if strings.TrimSpace(message) == "" {
			continue
		}

		// Send message
		_, err := writer.WriteString(message + "\n")
		if err != nil {
			fmt.Printf("Failed to send message: %v\n", err)
			break
		}
		writer.Flush()

		// If it's a quit command, wait for server response then exit
		if strings.ToLower(strings.TrimSpace(message)) == "quit" ||
			strings.ToLower(strings.TrimSpace(message)) == "exit" {
			time.Sleep(100 * time.Millisecond)
			break
		}
	}

	fmt.Println("Client exiting")
}
