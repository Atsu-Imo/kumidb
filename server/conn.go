package server

import (
	"context"
	"log"
	"net"
	"strings"

	"github.com/tidwall/buntdb"
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
			c.conn.Write([]byte("insert start\n"))
			data := strings.Split(input, " ")
			pair := strings.Split(data[1], ":")
			c.insert(pair[0], pair[1])
			c.conn.Write([]byte("insert finish\n"))
		} else if strings.HasPrefix(input, "select") {
			c.conn.Write([]byte("select start\n"))
			data := strings.Split(input, " ")
			result := c.find(strings.TrimRight(data[1], "\n"))
			c.conn.Write([]byte(result + "\n"))
			c.conn.Write([]byte("select end\n"))
		} else {
			c.conn.Write([]byte("not command\n"))
		}
	}
}

func (c *Conn) insert(key string, value string) {
	c.svr.Db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, value, nil)
		return err
	})
}

func (c *Conn) find(key string) string {
	var value string
	// うまくいってない
	err := c.svr.Db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(key)
		if err != nil {
			return err
		}
		value = val
		return nil
	})
	if err != nil {
		return "not found"
	}
	return value
}
