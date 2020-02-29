package main

import (
	"log"
	"net"

	"github.com/tidwall/buntdb"
)

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
	for {
		defer listener.Close()
		conn, err := listener.AcceptTCP()
		if err != nil {
			if netError, ok := err.(net.Error); ok {
				if netError.Temporary() {
					log.Fatal("AcceptTCP", netError)
				}
			}
		}
		go func() {
			defer conn.Close()
			buf := make([]byte, 1024)
			for {
				n, err := conn.Read(buf)
				if err != nil {
					if netError, ok := err.(net.Error); ok {
						if netError.Temporary() {
							log.Fatal("Read", netError)
							continue
						}
					}
					return
				}
				n, err = conn.Write(buf[:n])
				if err != nil {
					log.Fatal("Write", err)
				}
			}
		}()
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
