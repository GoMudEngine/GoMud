package connections

import (
	"net"
	"testing"
)

// These tests cover the registry half of the pre-login websocket leak.
//
// HandleWebSocketConnection lives in package main and needs a live upgraded
// socket, so it cannot be driven from here. What IS testable — and what the
// fix depends on — is the contract the new unconditional defer relies on:
// Remove() actually evicts a never-logged-in connection, and calling it a
// second time (which happens constantly, because the logged-in teardown paths
// and several early returns already call it) is a harmless no-op.

// TestRemoveReapsNeverLoggedInConnection is the direct observable named in the
// defect: a connection that is added and then dies without ever logging in
// must not remain counted against the connection limit.
func TestRemoveReapsNeverLoggedInConnection(t *testing.T) {

	baseline := ActiveConnectionCount()

	clientSide, peerSide := net.Pipe()
	defer clientSide.Close()
	defer peerSide.Close()

	cd := Add(clientSide, nil)
	id := cd.ConnectionId()

	t.Cleanup(func() {
		lock.Lock()
		delete(netConnections, id)
		lock.Unlock()
	})

	if cd.State() != Login {
		t.Fatalf("a freshly added connection should be in Login state, got %v", cd.State())
	}
	if got := ActiveConnectionCount(); got != baseline+1 {
		t.Fatalf("expected %d active connections after Add, got %d", baseline+1, got)
	}

	// The client hangs up mid-signup. This is what the new defer in
	// HandleWebSocketConnection does on every exit path.
	peerSide.Close()

	if err := Remove(id); err != nil {
		t.Fatalf("Remove of a live connection returned %v", err)
	}

	if got := ActiveConnectionCount(); got != baseline {
		t.Fatalf("connection leaked: expected %d active connections, got %d", baseline, got)
	}
	if _, stillThere := peekConnection(id); stillThere {
		t.Fatal("connection is still present in the registry after Remove")
	}
}

// TestRemoveIsIdempotent is the precondition for making the defer
// unconditional. A conditional defer ("only remove if not already removed")
// would reintroduce exactly the class of bug being fixed, so Remove has to
// tolerate being called again.
func TestRemoveIsIdempotent(t *testing.T) {

	baseline := ActiveConnectionCount()

	clientSide, peerSide := net.Pipe()
	defer clientSide.Close()
	defer peerSide.Close()

	id := Add(clientSide, nil).ConnectionId()

	t.Cleanup(func() {
		lock.Lock()
		delete(netConnections, id)
		lock.Unlock()
	})

	if err := Remove(id); err != nil {
		t.Fatalf("first Remove returned %v", err)
	}

	// Second and third calls: must not panic, must not double-count, must not
	// disturb any other connection.
	if err := Remove(id); err == nil {
		t.Fatal("expected 'connection not found' from a repeat Remove")
	}
	if err := Remove(id); err == nil {
		t.Fatal("expected 'connection not found' from a repeat Remove")
	}

	if got := ActiveConnectionCount(); got != baseline {
		t.Fatalf("expected %d active connections after repeated Remove, got %d", baseline, got)
	}
}

// TestBroadcastSkipsLoginStateConnections pins the reason the leak never
// self-healed: a stuck pre-login connection is skipped by Broadcast, so no
// later write failure can ever reap it. If this ever stops being true the
// comment on the defer in HandleWebSocketConnection needs revisiting.
func TestBroadcastSkipsLoginStateConnections(t *testing.T) {

	clientSide, peerSide := net.Pipe()
	defer clientSide.Close()
	defer peerSide.Close()

	id := Add(clientSide, nil).ConnectionId()
	defer Remove(id)

	// Not draining peerSide is safe: Broadcast must never reach the write.
	for _, sentId := range Broadcast([]byte("hello")) {
		if sentId == id {
			t.Fatal("Broadcast wrote to a connection still in Login state")
		}
	}
}

func peekConnection(id ConnectionId) (*ConnectionDetails, bool) {
	lock.RLock()
	defer lock.RUnlock()

	cd, ok := netConnections[id]
	return cd, ok
}
