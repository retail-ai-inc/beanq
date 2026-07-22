package bmongo

import (
	"context"
	"testing"
	"time"
)

func TestClientOptionsValidateAddress(t *testing.T) {
	store := &MongoStore{host: "::1", port: ":27017", connectTimeOut: time.Second}
	opts, err := store.clientOptions()
	if err != nil {
		t.Fatalf("clientOptions returned error: %v", err)
	}
	if len(opts.Hosts) != 1 || opts.Hosts[0] != "[::1]:27017" {
		t.Fatalf("Hosts = %#v", opts.Hosts)
	}

	store.port = "invalid"
	if _, err := store.clientOptions(); err == nil {
		t.Fatal("expected invalid port error")
	}
	store.host, store.port = "", "27017"
	if _, err := store.clientOptions(); err == nil {
		t.Fatal("expected empty host error")
	}
}

func TestClientOptionsAllowEmptyPassword(t *testing.T) {
	store := &MongoStore{host: "localhost", port: "27017", database: "beanq", userName: "user"}
	opts, err := store.clientOptions()
	if err != nil {
		t.Fatalf("clientOptions returned error: %v", err)
	}
	if opts.Auth == nil || opts.Auth.Username != "user" || opts.Auth.Password != "" || opts.Auth.AuthSource != "beanq" {
		t.Fatalf("unexpected authentication options: %#v", opts.Auth)
	}
	store.database = ""
	opts, err = store.clientOptions()
	if err != nil || opts.Auth.AuthSource != "admin" {
		t.Fatalf("default authentication source = %#v, error = %v", opts.Auth, err)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	store := &MongoStore{}
	if err := store.Close(context.Background()); err != nil {
		t.Fatalf("first Close returned error: %v", err)
	}
	if err := store.Close(context.Background()); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
	if !store.closed {
		t.Fatal("store was not marked closed")
	}
}
