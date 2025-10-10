package eddy

import (
	"testing"
	"time"
)

type fakeCacheKey string

func (k fakeCacheKey) String() string {
	return string(k)
}

func TestClientPoolGetSetAndRemove(t *testing.T) {
	pool := NewClientPool[string](time.Minute)
	key := fakeCacheKey("tenant:provider")

	if client := pool.GetClient(key); client.IsPresent() {
		t.Fatal("expected empty pool to return none")
	}

	pool.SetClient(key, "client-1")

	client := pool.GetClient(key)
	if !client.IsPresent() {
		t.Fatal("expected client to be present after SetClient")
	}

	if client.MustGet() != "client-1" {
		t.Fatalf("expected client 'client-1', got %q", client.MustGet())
	}

	pool.RemoveClient(key)

	if client := pool.GetClient(key); client.IsPresent() {
		t.Fatal("expected client to be removed")
	}
}

func TestClientPoolGetClientExpired(t *testing.T) {
	pool := NewClientPool[string](0)
	key := fakeCacheKey("tenant:provider")

	pool.SetClient(key, "client-1")

	if client := pool.GetClient(key); client.IsPresent() {
		t.Fatal("expected client to be expired immediately")
	}
}

func TestClientPoolCleanExpired(t *testing.T) {
	pool := NewClientPool[string](time.Minute)
	activeKey := fakeCacheKey("active")
	expiredKey := fakeCacheKey("expired")

	pool.SetClient(activeKey, "active-client")
	pool.SetClient(expiredKey, "expired-client")

	pool.mu.Lock()
	pool.clients[expiredKey.String()].Expiration = time.Now().Add(-time.Minute)
	pool.mu.Unlock()

	removed := pool.CleanExpired()
	if removed != 1 {
		t.Fatalf("expected to remove 1 expired client, removed %d", removed)
	}

	if client := pool.GetClient(expiredKey); client.IsPresent() {
		t.Fatal("expected expired client not to be present after CleanExpired")
	}

	if client := pool.GetClient(activeKey); !client.IsPresent() || client.MustGet() != "active-client" {
		t.Fatal("expected active client to remain in pool")
	}
}
