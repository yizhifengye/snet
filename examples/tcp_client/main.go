package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/sandwich-go/logbus"
	"go.uber.org/zap/zapcore"
)

// initLogger initializes logbus logger with appropriate configuration
func initLogger() {
	logbus.Init(logbus.NewConf(
		logbus.WithCallerSkip(2),
		logbus.WithDev(true), // Enable development mode for better formatting
		logbus.WithLogLevel(zapcore.InfoLevel),
	))
	logbus.Info("TCP client logger initialized successfully")
}

func main() {
	// Initialize logger first
	initLogger()

	// Connect directly to server using plain TCP
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		logbus.Fatal("Failed to connect to server", logbus.ErrorField(err))
	}
	defer conn.Close()

	logbus.Info("Successfully connected to server using plain TCP",
		logbus.String("local_addr", conn.LocalAddr().String()),
		logbus.String("server_addr", conn.RemoteAddr().String()))

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	scanner := bufio.NewScanner(os.Stdin)

	// Start goroutine to receive messages from server
	go func() {
		for {
			message, err := reader.ReadString('\n')
			if err != nil {
				logbus.Error("Failed to read server message", logbus.ErrorField(err))
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
			logbus.Error("Failed to send message",
				logbus.String("message", message),
				logbus.ErrorField(err))
			break
		}
		writer.Flush()

		logbus.Debug("Sent message to server", logbus.String("message", message))

		// If it's a quit command, wait for server response then exit
		if strings.ToLower(strings.TrimSpace(message)) == "quit" ||
			strings.ToLower(strings.TrimSpace(message)) == "exit" {
			time.Sleep(100 * time.Millisecond)
			break
		}
	}

	logbus.Info("TCP client exiting")
}
