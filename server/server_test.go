package server

import (
	"context"
	"log"
	"reflect"
	"testing"

	"github.com/tidwall/buntdb"
)

func TestNEwServer(t *testing.T) {
	ctx := context.Background()
	db, err := buntdb.Open(":memory:")
	if err != nil {
		log.Fatal("Open DB", err)
		return
	}
	defer db.Close()
	srv := NewServer(ctx, "", db)
	if reflect.TypeOf(srv) != reflect.TypeOf(&Server{}) {
		t.Fail()
	}
}
