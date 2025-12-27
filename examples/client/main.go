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

	snet "github.com/yizhifengye/snet/go"
)

// initLogger initializes logbus logger with appropriate configuration
func initLogger() {
	logbus.Init(logbus.NewConf(
		logbus.WithCallerSkip(2),
		logbus.WithDev(true), // Enable development mode for better formatting
		logbus.WithLogLevel(zapcore.InfoLevel),
	))
	logbus.Info("Client logger initialized successfully")
}

func main() {
	// Initialize logger first
	initLogger()

	config := snet.Config{
		EnableCrypt:        false, // Must match server settings
		HandshakeTimeout:   time.Second * 10,
		RewriterBufferSize: 1024,
		ReconnWaitTimeout:  time.Second * 30,
	}

	// Connect to server
	conn, err := snet.Dial(config, func() (net.Conn, error) {
		return net.Dial("tcp", "localhost:8080")
	})
	if err != nil {
		logbus.Fatal("Failed to connect to server", logbus.ErrorField(err))
	}
	defer conn.Close()

	logbus.Info("Successfully connected to SNET server",
		logbus.String("local_addr", conn.LocalAddr().String()),
		logbus.String("server_addr", conn.RemoteAddr().String()),
		logbus.Bool("encryption", config.EnableCrypt))

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	scanner := bufio.NewScanner(os.Stdin)

	// Start goroutine to receive messages
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

	logbus.Info("Client exiting")
}
