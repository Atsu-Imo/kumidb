package server

import (
	"context"
	"log"
	"net"
	"strings"
	"sync"
)

const (
	listenerCloseMatcher = "use of closed network connection"
)

func listenerCloseError(err error) bool {
	return strings.Contains(err.Error(), listenerCloseMatcher)
}

// Server サーバーにまつわるあれこれの構造体
type Server struct {
	addr      string
	listener  *net.TCPListener
	ctx       context.Context
	shutdown  context.CancelFunc
	AcceptCtx context.Context
	errAccept context.CancelFunc
	Wg        sync.WaitGroup
	ChClosed  chan struct{}
}

// NewServer TCP接続を待ち受けるServerを作る
func NewServer(parentCtx context.Context, addr string) *Server {
	acceptCtx, errAccept := context.WithCancel(parentCtx)
	// serverCtxはすべてのgoroutineが同じContext上で動いていることを示す
	// goroutine間の値の伝播
	// context.Background()が親のContext
	// context.WithCancel(ctx)によって子Contextと子Contextをキャンセルする関数が返る
	serverCtx, shutdown := context.WithCancel(context.Background())
	chCloed := make(chan struct{})
	return &Server{
		addr:      addr,
		ctx:       serverCtx,
		shutdown:  shutdown,
		AcceptCtx: acceptCtx,
		errAccept: errAccept,
		ChClosed:  chCloed,
	}
}

// Listen TCP接続を待ち受けるgoroutineを起動する
func (s *Server) Listen() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", s.addr)
	if err != nil {
		return err
	}

	l, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}
	s.listener = l

	go s.handleListener()
	return nil
}

// Shutdown TCP接続を待ち受けるリスナーをクローズする
// すでにContextが終了していた場合は何もしない
func (s *Server) Shutdown() {
	select {
	case <-s.ctx.Done():
	default:
		s.Shutdown()
		s.listener.Close()
	}
}

func (s *Server) handleListener() {
	defer func() {
		s.listener.Close()
		close(s.ChClosed)
	}()
	for {
		tcpConn, err := s.listener.AcceptTCP()
		if err != nil {
			if netError, ok := err.(net.Error); ok {
				if netError.Temporary() {
					log.Fatal("AcceptTCP", netError)
					continue
				}
			}
			//
			if listenerCloseError(err) {
				select {
				// Contextがキャンセルされたときに閉じられるチャネルを返す
				case <-s.ctx.Done():
					return
				default:
					// 何もしない
				}
			}
			log.Fatal("AcceptTCP", err)
			s.errAccept()
			return
		}
		// コネクションが1つできた
		s.Wg.Add(1)
		conn := newConn(s, tcpConn)
		go conn.handlerConnection()
	}
}
