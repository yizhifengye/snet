package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	// Connect directly to server using plain TCP
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	fmt.Println("Successfully connected to server using plain TCP")
	fmt.Printf("Local address: %s\n", conn.LocalAddr().String())
	fmt.Printf("Server address: %s\n", conn.RemoteAddr().String())

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	scanner := bufio.NewScanner(os.Stdin)

	// Start goroutine to receive messages from server
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

	fmt.Println("This is a plain TCP client for testing SNET server compatibility")
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

		// Send message directly as plain text
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

	fmt.Println("TCP client exiting")
}
