package eddy

import (
	"context"

	"github.com/samber/mo"
)

// ClientService manages client pooling and provides cached client instances
// The builder is provided directly by the rule evaluation result rather than via a registry
type ClientService[T any, Output any, Config any] struct {
	pool       *ClientPool[T]
	outputCopy func(Output) Output
	configCopy func(Config) Config
}

// ServiceOption configures a ClientService
type ServiceOption[T any, Output any, Config any] func(*ClientService[T, Output, Config])

// WithOutputClone sets the output cloning function for defensive copying
func WithOutputClone[T any, Output any, Config any](cloneFn func(Output) Output) ServiceOption[T, Output, Config] {
	return func(s *ClientService[T, Output, Config]) {
		s.outputCopy = cloneFn
	}
}

// WithConfigClone sets the config cloning function for defensive copying
func WithConfigClone[T any, Output any, Config any](cloneFn func(Config) Config) ServiceOption[T, Output, Config] {
	return func(s *ClientService[T, Output, Config]) {
		s.configCopy = cloneFn
	}
}

// NewClientService creates a new client service with the specified pool
func NewClientService[T any, Output any, Config any](pool *ClientPool[T], opts ...ServiceOption[T, Output, Config]) *ClientService[T, Output, Config] {
	s := &ClientService[T, Output, Config]{
		pool: pool,
		outputCopy: func(c Output) Output {
			return c
		},
		configCopy: func(c Config) Config {
			return c
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// GetClient retrieves a client from cache or builds a new one using the provided builder
// The builder is provided directly from the rule evaluation result
func (s *ClientService[T, Output, Config]) GetClient(ctx context.Context, key CacheKey, builder Builder[T, Output, Config], output Output, config Config) mo.Option[T] {
	if cached := s.pool.GetClient(key); cached.IsPresent() {
		return cached
	}

	// Build new client using the provided builder
	// Creates defensive copies of output and config to avoid
	// potential side effects from external modifications
	client, err := builder.Build(ctx, s.outputCopy(output), s.configCopy(config))
	if err != nil {
		return mo.None[T]()
	}

	s.pool.SetClient(key, client)

	return mo.Some(client)
}

// Pool returns the underlying client pool
func (s *ClientService[T, Output, Config]) Pool() *ClientPool[T] {
	return s.pool
}
