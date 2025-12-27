package snet

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
	"github.com/funny/crypto/dh64/go"
)

// AcceptOnConn 接受一个已建立的连接，并进行握手
func AcceptOnConn(conn net.Conn, config Config, generateConnID func() uint64) (*Conn, error) {
	if config.HandshakeTimeout > 0 {
		conn.SetDeadline(time.Now().Add(config.HandshakeTimeout))
		defer conn.SetDeadline(time.Time{})
	}

	var (
		buf    [24]byte
		field1 = buf[0:8]
		field2 = buf[8:16]
		field3 = buf[16:24]
	)
	// 读取客户端公钥
	if _, err := io.ReadFull(conn, field1); err != nil {
		conn.Close()
		return nil, err
	}

	connPubKey := binary.LittleEndian.Uint64(field1)
	if connPubKey == 0 {
		conn.Close()
		return nil, fmt.Errorf("zero public key")
	}

	privKey, pubKey := dh64.KeyPair()
	secret := dh64.Secret(privKey, connPubKey)

	connID := generateConnID()
	sconn, err := newConn(conn, connID, secret, config)
	if err != nil {
		conn.Close()
		return nil, err
	}

	binary.LittleEndian.PutUint64(field1, pubKey)
	binary.LittleEndian.PutUint64(field2, connID)
	sconn.writeCipher.XORKeyStream(field2, field2)
	rand.Read(field3)
	if _, err := conn.Write(buf[:]); err != nil {
		conn.Close()
		return nil, err
	}

	// 二次握手
	var buf2 [16]byte
	if _, err := io.ReadFull(conn, buf2[:]); err != nil {
		conn.Close()
		return nil, err
	}

	hash := md5.New()
	hash.Write(field3)
	hash.Write(sconn.key[:])
	md5sum := hash.Sum(nil)
	if !bytes.Equal(buf2[:], md5sum) {
		conn.Close()
		return nil, fmt.Errorf("twice handshake not equals")
	}

	return sconn, nil
}

// AcceptReconnOnConn 处理一个已建立连接的 snet 重连握手流程
func AcceptReconnOnConn(conn net.Conn, config Config, getConnByID func(uint64) (*Conn, bool)) (*Conn, error) {
	if config.ReconnWaitTimeout > 0 {
		conn.SetDeadline(time.Now().Add(config.ReconnWaitTimeout))
		defer conn.SetDeadline(time.Time{})
	}

	var (
		buf    [24 + md5.Size]byte
		buf2   [24]byte
		field1 = buf[0:8]
		field2 = buf[8:16]
		field3 = buf[16:24]
		field4 = buf[24 : 24+md5.Size]
	)
	if _, err := io.ReadFull(conn, buf[:]); err != nil {
		conn.Close()
		return nil, err
	}

	connID := binary.LittleEndian.Uint64(field1)
	sconn, exists := getConnByID(connID)
	if !exists {
		conn.Write(buf2[:]) // 返回空响应表示拒绝重连
		conn.Close()
		return nil, fmt.Errorf("conn %d not exists", connID)
	}

	hash := md5.New()
	hash.Write(buf[:24])
	hash.Write(sconn.key[:])
	md5sum := hash.Sum(nil)
	if !bytes.Equal(field4, md5sum) {
		conn.Write(buf2[:]) // 返回空响应表示拒绝重连
		conn.Close()
		return nil, fmt.Errorf("reconn hash mismatch")
	}

	writeCount := binary.LittleEndian.Uint64(field2)
	readCount := binary.LittleEndian.Uint64(field3)

	// 处理重连逻辑，返回新的连接
	sconn.handleReconn(conn, writeCount, readCount)
	return sconn, nil
}
