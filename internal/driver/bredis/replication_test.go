package bredis

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type recordedRedisCommand struct {
	connection int
	name       string
}

type waitTestServer struct {
	listener net.Listener
	waitAcks int64
	mu       sync.Mutex
	commands []recordedRedisCommand
}

func newWaitTestServer(t *testing.T, waitAcks int64) *waitTestServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &waitTestServer{listener: listener, waitAcks: waitAcks}
	t.Cleanup(func() { _ = listener.Close() })
	go server.serve()
	return server
}

func (s *waitTestServer) serve() {
	connection := 0
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		connection++
		go s.serveConn(conn, connection)
	}
}

func (s *waitTestServer) serveConn(conn net.Conn, connection int) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	for {
		args, err := readRESPCommand(reader)
		if err != nil {
			return
		}
		name := strings.ToUpper(args[0])
		s.mu.Lock()
		s.commands = append(s.commands, recordedRedisCommand{connection: connection, name: name})
		s.mu.Unlock()
		switch name {
		case "HELLO":
			_, _ = writer.WriteString("-ERR unknown command 'hello'\r\n")
		case "XADD":
			_, _ = writer.WriteString("$3\r\n1-0\r\n")
		case "WAIT":
			_, _ = fmt.Fprintf(writer, ":%d\r\n", s.waitAcks)
		case "EVAL":
			_, _ = writer.WriteString("*3\r\n$9\r\nSCHEDULED\r\n$3\r\n1-0\r\n$1\r\n1\r\n")
		case "INFO":
			info := "# Replication\r\nrole:master\r\nconnected_slaves:0\r\n"
			_, _ = fmt.Fprintf(writer, "$%d\r\n%s\r\n", len(info), info)
		case "XACK", "XDEL", "HSETNX":
			_, _ = writer.WriteString(":1\r\n")
		default:
			_, _ = writer.WriteString("+OK\r\n")
		}
		_ = writer.Flush()
	}
}

func readRESPCommand(reader *bufio.Reader) ([]string, error) {
	header, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if len(header) == 0 || header[0] != '*' {
		return nil, fmt.Errorf("unexpected RESP header %q", header)
	}
	count, err := strconv.Atoi(strings.TrimSpace(header[1:]))
	if err != nil {
		return nil, err
	}
	args := make([]string, 0, count)
	for range count {
		lengthLine, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		length, err := strconv.Atoi(strings.TrimSpace(lengthLine[1:]))
		if err != nil {
			return nil, err
		}
		value := make([]byte, length+2)
		if _, err := io.ReadFull(reader, value); err != nil {
			return nil, err
		}
		args = append(args, string(value[:length]))
	}
	return args, nil
}

func (s *waitTestServer) client() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: s.listener.Addr().String(), Protocol: 2, DisableIdentity: true})
}

func (s *waitTestServer) snapshot() []recordedRedisCommand {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]recordedRedisCommand(nil), s.commands...)
}

func TestReplicationWaitUsesSameConnection(t *testing.T) {
	server := newWaitTestServer(t, 1)
	client := server.client()
	defer client.Close()
	wait := replicationWait{replicas: 1, timeout: time.Second}
	_, err := wait.execute(context.Background(), client, "queue:{slot}:stream", func(pipe redis.Pipeliner) redis.Cmder {
		return pipe.XAdd(context.Background(), &redis.XAddArgs{Stream: "queue:{slot}:stream", Values: map[string]any{"id": "1"}})
	})
	if err != nil {
		t.Fatal(err)
	}
	commands := server.snapshot()
	for index := range commands[:len(commands)-1] {
		if commands[index].name == "XADD" && commands[index+1].name == "WAIT" {
			if commands[index].connection != commands[index+1].connection {
				t.Fatal("XADD and WAIT used different connections")
			}
			return
		}
	}
	t.Fatalf("XADD followed by WAIT not found in %#v", commands)
}

func TestReplicationWaitReusesPooledConnection(t *testing.T) {
	server := newWaitTestServer(t, 1)
	client := server.client()
	defer client.Close()
	wait := replicationWait{replicas: 1, timeout: time.Second}
	for range 2 {
		_, err := wait.execute(context.Background(), client, "queue:{slot}:stream", func(pipe redis.Pipeliner) redis.Cmder {
			return pipe.XAdd(context.Background(), &redis.XAddArgs{Stream: "queue:{slot}:stream", Values: map[string]any{"id": "1"}})
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	var connections []int
	for _, command := range server.snapshot() {
		if command.name == "XADD" {
			connections = append(connections, command.connection)
		}
	}
	if len(connections) != 2 || connections[0] != connections[1] {
		t.Fatalf("expected pooled connection reuse, got %v", connections)
	}
}

func TestAckAndDeleteWaitOrder(t *testing.T) {
	server := newWaitTestServer(t, 1)
	client := server.client()
	defer client.Close()
	wait := replicationWait{replicas: 1, timeout: time.Second}
	if err := ackAndDelete(context.Background(), client, wait, "queue:{slot}:stream", "workers", "1-0"); err != nil {
		t.Fatal(err)
	}
	commands := server.snapshot()
	for i := 0; i+2 < len(commands); i++ {
		if commands[i].name == "XACK" && commands[i+1].name == "XDEL" && commands[i+2].name == "WAIT" {
			if commands[i].connection != commands[i+1].connection || commands[i].connection != commands[i+2].connection {
				t.Fatalf("commands used different connections: %#v", commands[i:i+3])
			}
			return
		}
	}
	t.Fatalf("XACK/XDEL/WAIT order not found in %#v", commands)
}

func TestValidateReplicationTopologyRejectsStandaloneWithoutReplicas(t *testing.T) {
	server := newWaitTestServer(t, 0)
	client := server.client()
	defer client.Close()
	err := ValidateReplicationTopology(context.Background(), client, 1)
	if err == nil {
		t.Fatal("expected insufficient replica error")
	}
	if !strings.Contains(err.Error(), client.Options().Addr) ||
		!strings.Contains(err.Error(), "required 1, actual 0") {
		t.Fatalf("unexpected topology error: %v", err)
	}
}

func TestReplicationWaitInsufficientAcknowledgementsIsAmbiguous(t *testing.T) {
	server := newWaitTestServer(t, 0)
	client := server.client()
	defer client.Close()
	wait := replicationWait{replicas: 1, timeout: 25 * time.Millisecond}
	_, err := wait.execute(context.Background(), client, "queue:{slot}:stream", func(pipe redis.Pipeliner) redis.Cmder {
		return pipe.XAdd(context.Background(), &redis.XAddArgs{Stream: "queue:{slot}:stream", Values: map[string]any{"id": "1"}})
	})
	if !errors.Is(err, ErrAmbiguousCommit) {
		t.Fatalf("expected ambiguous commit, got %v", err)
	}
	replicationErr, ok := errors.AsType[*ReplicationNotConfirmedError](err)
	if !ok || replicationErr.Acknowledged != 0 || replicationErr.Required != 1 {
		t.Fatalf("unexpected replication error: %#v", replicationErr)
	}
}

func TestSequenceQueueWaitUsesEvalOnFreshNode(t *testing.T) {
	server := newWaitTestServer(t, 1)
	client := server.client()
	defer client.Close()
	topology := newSequenceQueueTopology("p", "c", "t", 1)
	store := newSequenceQueueStore(client, topology, 10, time.Second)
	result, err := store.enqueueWithWait(context.Background(), "order-1", map[string]any{"id": "message-1"}, replicationWait{replicas: 1, timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if result.Code != "SCHEDULED" || result.SchedulerID != "1-0" || result.Pending != 1 {
		t.Fatalf("unexpected enqueue result: %#v", result)
	}
	commands := server.snapshot()
	for index := range commands[:len(commands)-1] {
		if commands[index].name == "EVAL" && commands[index+1].name == "WAIT" {
			return
		}
	}
	t.Fatalf("EVAL followed by WAIT not found in %#v", commands)
}

func TestWaitTimeoutMillisecondsRoundsUp(t *testing.T) {
	if waitTimeoutMilliseconds(500*time.Microsecond) != 1 {
		t.Fail()
	}
}

func TestClusterRedirect(t *testing.T) {
	for _, test := range []struct {
		err     string
		kind    string
		address string
	}{
		{err: "MOVED 1234 127.0.0.1:7001", kind: "MOVED", address: "127.0.0.1:7001"},
		{err: "ASK 4321 127.0.0.1:7002", kind: "ASK", address: "127.0.0.1:7002"},
	} {
		cmd := redis.NewCmd(context.Background())
		cmd.SetErr(errors.New(test.err))
		kind, address, ok := clusterRedirect(cmd)
		if !ok || kind != test.kind || address != test.address {
			t.Fatalf("clusterRedirect(%q) = %q, %q, %v", test.err, kind, address, ok)
		}
	}
}
