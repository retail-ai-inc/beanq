package bredis

import (
	"context"
	"strings"
	"testing"
)

func TestValidateReplicationTopologyDisabledDoesNotUseClient(t *testing.T) {
	if err := ValidateReplicationTopology(context.Background(), nil, 0); err != nil {
		t.Fatal(err)
	}
}

func TestParseReplicationInfoCountsOnlyOnlineReplicas(t *testing.T) {
	info := strings.Join([]string{
		"# Replication",
		"role:master",
		"connected_slaves:2",
		"slave0:ip=127.0.0.1,port=6380,state=online,offset=1,lag=0",
		"slave1:ip=127.0.0.1,port=6381,state=offline,offset=1,lag=0",
	}, "\r\n")
	role, online, err := parseReplicationInfo(info)
	if err != nil {
		t.Fatal(err)
	}
	if role != "master" || online != 1 {
		t.Fatalf("got role=%q online=%d", role, online)
	}
}

func TestParseReplicationInfoSupportsConnectedSlavesOnly(t *testing.T) {
	role, online, err := parseReplicationInfo("role:master\nconnected_slaves:2\n")
	if err != nil {
		t.Fatal(err)
	}
	if role != "master" || online != 2 {
		t.Fatalf("got role=%q online=%d", role, online)
	}
}
