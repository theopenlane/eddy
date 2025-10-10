package eddy

import (
	"sync"
	"time"

	"github.com/samber/mo"
)

// ClientEntry wraps a client instance with expiration metadata
type ClientEntry[T any] struct {
	// Client is the cached client instance
	Client T
	// Expiration is when this cache entry expires
	Expiration time.Time
}

// ClientPool holds cached client instances with TTL expiration
type ClientPool[T any] struct {
	mu      sync.RWMutex
	clients map[string]*ClientEntry[T]
	ttl     time.Duration
}

// NewClientPool creates a new client pool with the specified TTL
func NewClientPool[T any](ttl time.Duration) *ClientPool[T] {
	return &ClientPool[T]{
		clients: make(map[string]*ClientEntry[T]),
		ttl:     ttl,
	}
}

// GetClient retrieves a client from the pool if it exists and hasn't expired
func (p *ClientPool[T]) GetClient(key CacheKey) mo.Option[T] {
	p.mu.RLock()
	defer p.mu.RUnlock()

	keyStr := key.String()
	if entry, exists := p.clients[keyStr]; exists && time.Now().Before(entry.Expiration) {
		return mo.Some(entry.Client)
	}

	return mo.None[T]()
}

// SetClient stores a client in the pool with TTL expiration
func (p *ClientPool[T]) SetClient(key CacheKey, client T) {
	p.mu.Lock()
	defer p.mu.Unlock()

	keyStr := key.String()
	p.clients[keyStr] = &ClientEntry[T]{
		Client:     client,
		Expiration: time.Now().Add(p.ttl),
	}
}

// RemoveClient removes a client from the pool
func (p *ClientPool[T]) RemoveClient(key CacheKey) {
	p.mu.Lock()
	defer p.mu.Unlock()

	keyStr := key.String()
	delete(p.clients, keyStr)
}

// CleanExpired removes expired clients from the pool and returns the count of removed clients
func (p *ClientPool[T]) CleanExpired() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	removed := 0
	for key, entry := range p.clients {
		if now.After(entry.Expiration) {
			delete(p.clients, key)
			removed++
		}
	}
	return removed
}
