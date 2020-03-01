package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Atsu-Imo/kumidb/server"
	"github.com/tidwall/buntdb"
)

// main
// main シグナルを監視する
// handlerListener tcpをacceptする. 1つ来るとhandleConnectionのgoroutineが新しくできる
func main() {
	signalChan := make(chan os.Signal, 1)
	// whitelist SIGINT
	signal.Ignore()
	signal.Notify(signalChan, syscall.SIGINT)
	srv := server.NewServer(context.Background(), "0.0.0.0:6379")

	err := srv.Listen()

	if err != nil {
		log.Fatal("Listen", err)
	}
	log.Println("tcp server start")

	select {
	// signalChanに(シグナル)が来るまで待機する
	case s := <-signalChan:
		switch s {
		case syscall.SIGINT:
			log.Println("Server shutdown start")
			// 子コンテキストにキャンセルを伝える
			srv.Shutdown()

			// すべてのgoroutineが閉じられるまで待つ
			srv.Wg.Wait()
			<-srv.ChClosed
			log.Println("Server shutdown end")
		default:
			panic("unexpected signal")
		}
	case <-srv.AcceptCtx.Done():
		log.Println("Server Error Occurred")
		// wait until all connection closed
		srv.Wg.Wait()
		<-srv.ChClosed
		log.Println("Server Shutdown Completed")
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
