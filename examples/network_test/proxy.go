package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/sandwich-go/logbus"
	"go.uber.org/zap/zapcore"
)

type NetworkProxy struct {
	listenAddr string
	targetAddr string

	// Network simulation parameters
	dropRate     float64 // Packet drop rate (0.0 - 1.0)
	delayMin     time.Duration
	delayMax     time.Duration
	disconnectAt time.Duration // Disconnect after this duration

	connections map[net.Conn]net.Conn
	mu          sync.Mutex
}

// initLogger initializes logbus logger with appropriate configuration
func initLogger() {
	logbus.Init(logbus.NewConf(
		logbus.WithCallerSkip(2),
		logbus.WithDev(true), // Enable development mode for better formatting
		logbus.WithLogLevel(zapcore.InfoLevel),
	))
	logbus.Info("Network proxy logger initialized successfully")
}

func NewNetworkProxy(listenAddr, targetAddr string) *NetworkProxy {
	return &NetworkProxy{
		listenAddr:  listenAddr,
		targetAddr:  targetAddr,
		connections: make(map[net.Conn]net.Conn),
	}
}

func (p *NetworkProxy) SetDropRate(rate float64) {
	p.dropRate = rate
}

func (p *NetworkProxy) SetDelay(min, max time.Duration) {
	p.delayMin = min
	p.delayMax = max
}

func (p *NetworkProxy) SetDisconnectAfter(duration time.Duration) {
	p.disconnectAt = duration
}

func (p *NetworkProxy) Start() error {
	listener, err := net.Listen("tcp", p.listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", p.listenAddr, err)
	}
	defer listener.Close()

	logbus.Info("Network proxy started",
		logbus.String("listen_addr", p.listenAddr),
		logbus.String("target_addr", p.targetAddr),
		logbus.Float64("drop_rate_percent", p.dropRate*100),
		logbus.Duration("delay_min", p.delayMin),
		logbus.Duration("delay_max", p.delayMax))

	if p.disconnectAt > 0 {
		logbus.Info("Automatic disconnect configured", logbus.Duration("disconnect_after", p.disconnectAt))
	}

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			logbus.Error("Failed to accept connection", logbus.ErrorField(err))
			continue
		}

		go p.handleConnection(clientConn)
	}
}

func (p *NetworkProxy) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// Connect to target server
	serverConn, err := net.Dial("tcp", p.targetAddr)
	if err != nil {
		logbus.Error("Failed to connect to target",
			logbus.String("target_addr", p.targetAddr),
			logbus.ErrorField(err))
		return
	}
	defer serverConn.Close()

	p.mu.Lock()
	p.connections[clientConn] = serverConn
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		delete(p.connections, clientConn)
		p.mu.Unlock()
	}()

	logbus.Info("New connection established",
		logbus.String("client_addr", clientConn.RemoteAddr().String()),
		logbus.String("target_addr", p.targetAddr))

	// Set up disconnect timer if specified
	var disconnectTimer *time.Timer
	if p.disconnectAt > 0 {
		disconnectTimer = time.AfterFunc(p.disconnectAt, func() {
			logbus.Info("Disconnecting connection due to timer",
				logbus.String("client_addr", clientConn.RemoteAddr().String()),
				logbus.Duration("after", p.disconnectAt))
			clientConn.Close()
			serverConn.Close()
		})
		defer disconnectTimer.Stop()
	}

	// Start proxying data in both directions
	var wg sync.WaitGroup
	wg.Add(2)

	// Client to Server
	go func() {
		defer wg.Done()
		p.proxyData(clientConn, serverConn, "client->server")
	}()

	// Server to Client
	go func() {
		defer wg.Done()
		p.proxyData(serverConn, clientConn, "server->client")
	}()

	wg.Wait()
	logbus.Info("Connection closed", logbus.String("client_addr", clientConn.RemoteAddr().String()))
}

func (p *NetworkProxy) proxyData(src, dst net.Conn, direction string) {
	buffer := make([]byte, 4096)

	for {
		n, err := src.Read(buffer)
		if err != nil {
			if err != io.EOF {
				logbus.Debug("Read error",
					logbus.String("direction", direction),
					logbus.ErrorField(err))
			}
			return
		}

		// Simulate packet drop
		if p.dropRate > 0 && rand.Float64() < p.dropRate {
			logbus.Debug("Packet dropped",
				logbus.String("direction", direction),
				logbus.Int("bytes", n))
			continue
		}

		// Simulate network delay
		if p.delayMax > 0 {
			delay := p.delayMin
			if p.delayMax > p.delayMin {
				delay += time.Duration(rand.Int63n(int64(p.delayMax - p.delayMin)))
			}
			time.Sleep(delay)
		}

		_, err = dst.Write(buffer[:n])
		if err != nil {
			logbus.Debug("Write error",
				logbus.String("direction", direction),
				logbus.ErrorField(err))
			return
		}

		logbus.Debug("Data proxied",
			logbus.String("direction", direction),
			logbus.Int("bytes", n))
	}
}

func main() {
	var (
		listenAddr   = flag.String("listen", ":8081", "Proxy listen address")
		targetAddr   = flag.String("target", "localhost:8080", "Target server address")
		dropRate     = flag.Float64("drop", 0.0, "Packet drop rate (0.0-1.0)")
		delayMin     = flag.Duration("delay-min", 0, "Minimum delay")
		delayMax     = flag.Duration("delay-max", 0, "Maximum delay")
		disconnectAt = flag.Duration("disconnect", 0, "Disconnect after duration (0 = never)")
	)
	flag.Parse()

	// Initialize logger first
	initLogger()

	rand.Seed(time.Now().UnixNano())

	proxy := NewNetworkProxy(*listenAddr, *targetAddr)
	proxy.SetDropRate(*dropRate)
	proxy.SetDelay(*delayMin, *delayMax)
	proxy.SetDisconnectAfter(*disconnectAt)

	if err := proxy.Start(); err != nil {
		logbus.Fatal("Proxy failed", logbus.ErrorField(err))
	}
}
