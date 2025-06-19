package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	snet "snet/go"
)

func main() {
	config := snet.Config{
		EnableCrypt:        false, // Can be set to true to enable encryption
		HandshakeTimeout:   time.Second * 10,
		RewriterBufferSize: 1024,
		ReconnWaitTimeout:  time.Second * 30,
	}

	// Start server
	listener, err := snet.Listen(config, func() (net.Listener, error) {
		return net.Listen("tcp", ":8080")
	})
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	fmt.Printf("SNET server started on %s\n", listener.Addr().String())
	fmt.Println("Waiting for client connections...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Start a goroutine for each client
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()
	fmt.Printf("Client connected: %s\n", clientAddr)

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Send welcome message
	welcome := "Welcome to SNET server!\n"
	writer.WriteString(welcome)
	writer.Flush()

	for {
		// Read client message
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Client %s disconnected: %v\n", clientAddr, err)
			break
		}

		message = strings.TrimSpace(message)
		fmt.Printf("Received message from %s: %s\n", clientAddr, message)

		// Handle special commands
		switch strings.ToLower(message) {
		case "quit", "exit":
			response := "Goodbye!\n"
			writer.WriteString(response)
			writer.Flush()
			fmt.Printf("Client %s actively disconnected\n", clientAddr)
			return
		case "time":
			response := fmt.Sprintf("Current time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
			writer.WriteString(response)
		case "help":
			help := "Available commands:\n- time: Get current time\n- quit/exit: Disconnect\n- help: Show help\n"
			writer.WriteString(help)
		default:
			// Echo message
			response := fmt.Sprintf("Server received: %s\n", message)
			writer.WriteString(response)
		}

		writer.Flush()
	}
}
