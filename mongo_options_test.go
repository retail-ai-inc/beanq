package beanq

import (
	"testing"
	"time"
)

func TestMongoClientOptions(t *testing.T) {
	tests := []struct {
		name      string
		config    *Mongo
		wantHost  string
		wantAuth  bool
		wantError bool
	}{
		{
			name:      "nil config",
			wantError: true,
		},
		{
			name: "host with colon port",
			config: &Mongo{
				Host:                  "localhost",
				Port:                  ":27017",
				Database:              "beanq",
				ConnectTimeOut:        time.Second,
				MaxConnectionPoolSize: 10,
				MaxConnectionLifeTime: time.Minute,
			},
			wantHost: "localhost:27017",
		},
		{
			name: "with auth",
			config: &Mongo{
				Host:     "mongo",
				Port:     "27018",
				Database: "beanq",
				UserName: "user",
				Password: "pass",
			},
			wantHost: "mongo:27018",
			wantAuth: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := mongoClientOptions(tt.config)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("mongoClientOptions error: %v", err)
			}
			if got := opts.Hosts[0]; got != tt.wantHost {
				t.Fatalf("host = %q, want %q", got, tt.wantHost)
			}
			if tt.wantAuth && opts.Auth == nil {
				t.Fatal("expected auth options")
			}
			if !tt.wantAuth && opts.Auth != nil {
				t.Fatal("did not expect auth options")
			}
		})
	}
}
