package eddy

import (
	"context"
)

// Builder builds client instances with output and configuration
type Builder[T any, Output any, Config any] interface {
	// Build constructs a client instance using the provided output and config
	Build(ctx context.Context, output Output, config Config) (T, error)
	// ProviderType returns the provider type identifier for cache key construction
	ProviderType() string
}

// BuilderFunc is a function adapter for Builder interface
// Use this when you want to create a Builder from a function without defining a new type
//
// Example:
//
//	builder := &BuilderFunc[*s3.Client, S3Credentials, S3Config]{
//	    Type: "s3",
//	    Func: func(ctx context.Context, output S3Credentials, config S3Config) (*s3.Client, error) {
//	        return buildS3Client(ctx, output, config)
//	    },
//	}
type BuilderFunc[T any, Output any, Config any] struct {
	// Type is the provider type identifier
	Type string
	// Func is the function that builds the client
	Func func(context.Context, Output, Config) (T, error)
}

// Build implements Builder.Build
func (b *BuilderFunc[T, Output, Config]) Build(ctx context.Context, output Output, config Config) (T, error) {
	return b.Func(ctx, output, config)
}

// ProviderType implements Builder.ProviderType
func (b *BuilderFunc[T, Output, Config]) ProviderType() string {
	return b.Type
}
