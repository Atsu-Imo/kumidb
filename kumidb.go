package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/tidwall/buntdb"
)

const (
	listenerCloseMatcher = "use of closed network connection"
)

// handlerListener
// handlerConnection コネクション内部での処理
func handleListener(serverCtx context.Context, listener *net.TCPListener, wg *sync.WaitGroup, chClosed chan struct{}) {
	defer func() {
		listener.Close()
		close(chClosed)
	}()
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			if netError, ok := err.(net.Error); ok {
				if netError.Temporary() {
					log.Fatal("AcceptTCP", netError)
				}
			}
			//
			if listenerCloseError(err) {
				select {
				// Contextがキャンセルされたときに閉じられるチャネルを返す
				case <-serverCtx.Done():
					return
				default:
					// 何もしない
				}
			}
		}
		// コネクションが1つできた
		wg.Add(1)
		go handlerConnection(serverCtx, conn, wg)
	}
}

// handlerConnection
func handlerConnection(serverCtx context.Context, conn *net.TCPConn, wg *sync.WaitGroup) {
	defer func() {
		conn.Close()
		wg.Done()
	}()

	readCtx, errRead := context.WithCancel(context.Background())
	go handleRead(conn, errRead)

	select {
	case <-readCtx.Done():
	case <-serverCtx.Done():
	}
}

func handleRead(conn *net.TCPConn, errRead context.CancelFunc) {
	defer errRead()

	buf := make([]byte, 4*1024)

	for {
		n, err := conn.Read(buf)
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

		n, err = conn.Write(buf[:n])
		if err != nil {
			log.Println("Write", err)
			return
		}
	}
}

func listenerCloseError(err error) bool {
	return strings.Contains(err.Error(), listenerCloseMatcher)
}

// main
// main シグナルを監視する
// handlerListener tcpをacceptする. 1つ来るとhandleConnectionのgoroutineが新しくできる
func main() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:6379")
	if err != nil {
		log.Fatal("ResolveTCPAddr", err)
		return
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Fatal("ListenTCP", err)
	}

	signalChan := make(chan os.Signal, 1)
	// whitelist SIGINT
	signal.Ignore()
	signal.Notify(signalChan, syscall.SIGINT)

	var wg sync.WaitGroup
	chClosed := make(chan struct{})

	// serverCtxはすべてのgoroutineが同じContext上で動いていることを示す
	// goroutine間の値の伝播
	// context.Background()が親のContext
	// context.WithCancel(ctx)によって子Contextと子Contextをキャンセルする関数が返る
	serverCtx, shutdown := context.WithCancel(context.Background())

	go handleListener(serverCtx, listener, &wg, chClosed)

	log.Println("tcp server start")

	// signalChanに(シグナル)が来るまで待機する
	s := <-signalChan
	switch s {
	case syscall.SIGINT:
		log.Println("Server shutdown start")
		// 子コンテキストにキャンセルを伝える
		shutdown()
		listener.Close()

		// すべてのgoroutineが閉じられるまで待つ
		wg.Wait()
		<-chClosed
		log.Println("Server shutdown end")
	default:
		panic("unexpected signal")
	}
}

func sample() {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	err = db.Update(func(tx *buntdb.Tx) error {
		_, _, err = tx.Set("key", "value", nil)
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Print(`"key" is saved!`)
	err = db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get("key")
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("value is %s", val)
		return nil
	})
	defer db.Close()
}
