package server

import (
	"context"
	"log"
	"net"
	"strings"
)

// Conn コネクションにまつわるあれこれの構造体
type Conn struct {
	svr     *Server
	conn    *net.TCPConn
	readCtx context.Context
	errRead context.CancelFunc
}

func newConn(svr *Server, conn *net.TCPConn) *Conn {
	readCtx, errRead := context.WithCancel(context.Background())
	return &Conn{
		svr:     svr,
		conn:    conn,
		readCtx: readCtx,
		errRead: errRead,
	}
}

func (c *Conn) handlerConnection() {
	defer func() {
		c.conn.Close()
		c.svr.Wg.Done()
	}()

	go c.handleRead()

	select {
	case <-c.readCtx.Done():
	case <-c.svr.ctx.Done():
	case <-c.svr.AcceptCtx.Done():
	}
}

func (c *Conn) handleRead() {
	defer c.errRead()

	buf := make([]byte, 4*1024)

	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok {
				switch {
				case ne.Temporary():
					continue
				}
			}
			log.Println("Read", err)
			return
		}

		input := string(buf[:n])
		if strings.HasPrefix(input, "insert") {
			c.conn.Write([]byte("inserted!"))
		} else if strings.HasPrefix(input, "select") {
			c.conn.Write([]byte("selected!"))
		} else {
			c.conn.Write([]byte("not command"))
		}
		n, err = c.conn.Write(buf[:n])
		if err != nil {
			log.Println("Write", err)
			return
		}
	}
}
