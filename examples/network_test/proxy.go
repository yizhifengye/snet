package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
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

	fmt.Printf("Network proxy started on %s -> %s\n", p.listenAddr, p.targetAddr)
	fmt.Printf("Drop rate: %.2f%%, Delay: %v-%v\n", p.dropRate*100, p.delayMin, p.delayMax)
	if p.disconnectAt > 0 {
		fmt.Printf("Will disconnect after: %v\n", p.disconnectAt)
	}

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
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
		log.Printf("Failed to connect to target %s: %v", p.targetAddr, err)
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

	fmt.Printf("New connection: %s -> %s\n", clientConn.RemoteAddr(), p.targetAddr)

	// Set up disconnect timer if specified
	var disconnectTimer *time.Timer
	if p.disconnectAt > 0 {
		disconnectTimer = time.AfterFunc(p.disconnectAt, func() {
			fmt.Printf("Disconnecting connection after %v\n", p.disconnectAt)
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
	fmt.Printf("Connection closed: %s\n", clientConn.RemoteAddr())
}

func (p *NetworkProxy) proxyData(src, dst net.Conn, direction string) {
	buffer := make([]byte, 4096)

	for {
		n, err := src.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error (%s): %v", direction, err)
			}
			return
		}

		// Simulate packet drop
		if p.dropRate > 0 && rand.Float64() < p.dropRate {
			fmt.Printf("Dropped packet (%s): %d bytes\n", direction, n)
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
			log.Printf("Write error (%s): %v", direction, err)
			return
		}
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

	rand.Seed(time.Now().UnixNano())

	proxy := NewNetworkProxy(*listenAddr, *targetAddr)
	proxy.SetDropRate(*dropRate)
	proxy.SetDelay(*delayMin, *delayMax)
	proxy.SetDisconnectAfter(*disconnectAt)

	if err := proxy.Start(); err != nil {
		log.Fatalf("Proxy failed: %v", err)
	}
}
