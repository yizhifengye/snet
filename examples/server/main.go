package main

import (
	"bufio"
	"fmt"
	"net"
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
	logbus.Info("Logger initialized successfully")
}

func main() {
	// Initialize logger first
	initLogger()

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
		logbus.Fatal("Failed to start server", logbus.ErrorField(err))
	}
	defer listener.Close()

	logbus.Info("SNET server started",
		logbus.String("address", listener.Addr().String()),
		logbus.Bool("encryption", config.EnableCrypt))

	logbus.Info("Waiting for client connections...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			logbus.Error("Failed to accept connection", logbus.ErrorField(err))
			continue
		}

		// Start a goroutine for each client
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()
	logbus.Info("Client connected", logbus.String("client_addr", clientAddr))

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
			logbus.Info("Client disconnected",
				logbus.String("client_addr", clientAddr),
				logbus.ErrorField(err))
			break
		}

		message = strings.TrimSpace(message)
		logbus.Debug("Received message",
			logbus.String("client_addr", clientAddr),
			logbus.String("message", message))

		// Handle special commands
		switch strings.ToLower(message) {
		case "quit", "exit":
			response := "Goodbye!\n"
			writer.WriteString(response)
			writer.Flush()
			logbus.Info("Client actively disconnected", logbus.String("client_addr", clientAddr))
			return
		case "time":
			response := fmt.Sprintf("Current time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
			writer.WriteString(response)
			logbus.Debug("Sent time response", logbus.String("client_addr", clientAddr))
		case "help":
			help := "Available commands:\n- time: Get current time\n- quit/exit: Disconnect\n- help: Show help\n"
			writer.WriteString(help)
			logbus.Debug("Sent help response", logbus.String("client_addr", clientAddr))
		default:
			// Echo message
			response := fmt.Sprintf("Server received: %s\n", message)
			writer.WriteString(response)
			logbus.Debug("Echoed message",
				logbus.String("client_addr", clientAddr),
				logbus.String("response", strings.TrimSpace(response)))
		}

		writer.Flush()
	}
}
