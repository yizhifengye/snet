package snet

import (
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var _ net.Listener = &Listener{}

const (
	TYPE_NEWCONN byte = 0x00
	TYPE_RECONN  byte = 0xFF
)

type Listener struct {
	base         net.Listener
	config       Config
	acceptChan   chan net.Conn
	closed       bool
	closeOnce    sync.Once
	closeChan    chan struct{}
	atomicConnID uint64
	connsMutex   sync.Mutex
	conns        map[uint64]*Conn
}

func Listen(config Config, listenFunc func() (net.Listener, error)) (*Listener, error) {
	listener, err := listenFunc()
	if err != nil {
		return nil, err
	}
	l := &Listener{
		base:       listener,
		config:     config,
		closeChan:  make(chan struct{}),
		acceptChan: make(chan net.Conn, 1000),
		conns:      make(map[uint64]*Conn),
	}
	go l.acceptLoop()
	return l, nil
}

func (l *Listener) Addr() net.Addr {
	return l.base.Addr()
}

func (l *Listener) Close() error {
	l.closeOnce.Do(func() {
		l.closed = true
		close(l.closeChan)
	})
	return l.base.Close()
}

func (l *Listener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.acceptChan:
		return conn, nil
	case <-l.closeChan:
	}
	return nil, os.ErrInvalid
}

func (l *Listener) acceptLoop() {
	for {
		conn, err := l.base.Accept()
		if err != nil {
			if !l.closed {
				l.trace("accept failed: %v", err)
			}
			break
		}
		go l.handAccept(conn)
	}
}

func (l *Listener) handAccept(conn net.Conn) {
	var buf [1]byte
	if l.config.HandshakeTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(l.config.HandshakeTimeout))
		defer conn.SetReadDeadline(time.Time{})
	}

	if _, err := io.ReadFull(conn, buf[:]); err != nil {
		conn.Close()
		return
	}

	switch buf[0] {
	case TYPE_NEWCONN:
		l.handshake(conn)
	case TYPE_RECONN:
		l.reconn(conn)
	default:
		conn.Close()
	}
}

// handshake 处理新连接的握手
func (l *Listener) handshake(conn net.Conn) {
	sconn, err := AcceptOnConn(conn, l.config, func() uint64 {
		return atomic.AddUint64(&l.atomicConnID, 1)
	})
	if err != nil {
		l.trace("handshake failed: %v", err)
		conn.Close()
		return
	}
	sconn.listener = l
	l.putConn(sconn.id, sconn)
	select {
	case l.acceptChan <- sconn:
	case <-l.closeChan:
	}
}

// 重连
func (l *Listener) reconn(conn net.Conn) {
	_, _ = AcceptReconnOnConn(conn, l.config, l.getConn)
}

func (l *Listener) getConn(id uint64) (*Conn, bool) {
	l.connsMutex.Lock()
	defer l.connsMutex.Unlock()
	conn, exists := l.conns[id]
	return conn, exists
}

func (l *Listener) putConn(id uint64, conn *Conn) {
	l.connsMutex.Lock()
	defer l.connsMutex.Unlock()
	l.conns[id] = conn
}

func (l *Listener) delConn(id uint64) {
	l.connsMutex.Lock()
	defer l.connsMutex.Unlock()
	if _, exists := l.conns[id]; exists {
		delete(l.conns, id)
	}
}
