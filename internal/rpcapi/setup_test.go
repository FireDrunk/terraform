// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/internal/rpcapi/rawrpc"
)

func TestSetupServer_Handshake(t *testing.T) {
	called := 0
	server := newSetupServer(func(ctx context.Context, req *rawrpc.Handshake_Request, stopper *stopper) (*rawrpc.ServerCapabilities, error) {
		called++
		if got, want := req.Config.Credentials["localterraform.com"].Token, "boop"; got != want {
			t.Fatalf("incorrect token. got %q, want %q", got, want)
		}
		return &rawrpc.ServerCapabilities{}, nil
	})

	req := &rawrpc.Handshake_Request{
		Capabilities: &rawrpc.ClientCapabilities{},
		Config: &rawrpc.Config{
			Credentials: map[string]*rawrpc.HostCredential{
				"localterraform.com": {
					Token: "boop",
				},
			},
		},
	}
	_, err := server.Handshake(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if called != 1 {
		t.Errorf("unexpected initOthers call count %d, want 1", called)
	}

	_, err = server.Handshake(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "handshake already completed") {
		t.Fatalf("unexpected error: %s", err)
	}
	if called != 1 {
		t.Errorf("unexpected initOthers call count %d, want 1", called)
	}
}

func TestSetupServer_Stop(t *testing.T) {
	var s *stopper
	server := newSetupServer(func(ctx context.Context, req *rawrpc.Handshake_Request, stopper *stopper) (*rawrpc.ServerCapabilities, error) {
		s = stopper
		return &rawrpc.ServerCapabilities{}, nil
	})
	_, err := server.Handshake(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if s == nil {
		t.Fatal("stopper not passed to initOthers")
	}

	var wg sync.WaitGroup

	var stops []stopChan
	for range 2 {
		stops = append(stops, s.add())
		wg.Add(1)
	}

	for _, stop := range stops {
		stop := stop
		go func() {
			<-stop
			wg.Done()
		}()
	}

	server.Stop(context.Background(), nil)

	wg.Wait()
}