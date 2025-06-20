package snet

import "net"

// 一个轻量包装，让被抢读的 1 字节还能读回来
type rawConn struct {
    net.Conn
    prefix []byte
}

func (r *rawConn) Read(b []byte) (int, error) {
    if len(r.prefix) > 0 {
        n := copy(b, r.prefix)
        r.prefix = r.prefix[n:]
        return n, nil
    }
    return r.Conn.Read(b)
}

