package eddy_test

import (
	"context"
	"fmt"
	"time"

	"github.com/theopenlane/eddy"
	"github.com/theopenlane/eddy/helpers"
	"github.com/theopenlane/utils/contextx"
)

// Mock types for examples
type StorageClient interface {
	Upload(ctx context.Context, data []byte) error
	Download(ctx context.Context, key string) ([]byte, error)
}

type S3Client struct {
	endpoint string
	bucket   string
}

func (c *S3Client) Upload(ctx context.Context, data []byte) error {
	return nil
}

func (c *S3Client) Download(ctx context.Context, key string) ([]byte, error) {
	return nil, nil
}

type StorageCredentials struct {
	AccessKey string
	SecretKey string
	Region    string
}

type StorageConfig struct {
	Bucket   string
	Endpoint string
	Timeout  time.Duration
}

type StorageCacheKey struct {
	TenantID   string
	ProviderID string
}

func (k StorageCacheKey) String() string {
	return fmt.Sprintf("%s:%s", k.TenantID, k.ProviderID)
}

// S3Builder implements Builder interface
type S3Builder struct{}

func (b *S3Builder) Build(ctx context.Context, creds StorageCredentials, config StorageConfig) (StorageClient, error) {
	return &S3Client{
		endpoint: config.Endpoint,
		bucket:   config.Bucket,
	}, nil
}

func (b *S3Builder) ProviderType() string {
	return "s3"
}

// Context hint types
type ProviderHint string
type EnvironmentHint string
type TenantHint string

// Example: Rules provide builders directly
func Example_builderInRules() {
	// Create a resolver with rules that provide builders
	resolver := eddy.NewResolver[StorageClient, StorageCredentials, StorageConfig]()

	// Define builders once (these are stateless factories, reused across rules)
	s3Builder := &S3Builder{}

	// Rule 1: S3 provider
	resolver.AddRule(&helpers.MatchHintRule[StorageClient, StorageCredentials, StorageConfig, ProviderHint]{
		Value: "s3",
		Resolver: func(ctx context.Context) (*eddy.ResolvedProvider[StorageClient, StorageCredentials, StorageConfig], error) {
			// Resolve credentials from environment, vault, etc.
			creds := StorageCredentials{
				AccessKey: "access-key",
				SecretKey: "secret-key",
				Region:    "us-east-1",
			}

			config := StorageConfig{
				Bucket:   "my-bucket",
				Endpoint: "s3.amazonaws.com",
				Timeout:  30 * time.Second,
			}

			return &eddy.ResolvedProvider[StorageClient, StorageCredentials, StorageConfig]{
				Builder: s3Builder, // Rule provides the builder
				Output:  creds,
				Config:  config,
			}, nil
		},
	})

	// Setup context with hint
	ctx := context.Background()
	ctx = contextx.WithString(ctx, ProviderHint("s3"))

	// Resolve returns Result with Builder included
	result := resolver.Resolve(ctx)
	if !result.IsPresent() {
		fmt.Println("no provider resolved")
		return
	}

	res := result.MustGet()

	// Create client service and pool
	pool := eddy.NewClientPool[StorageClient](1 * time.Hour)
	service := eddy.NewClientService[StorageClient, StorageCredentials, StorageConfig](pool)

	// Build cache key
	cacheKey := StorageCacheKey{
		TenantID:   "tenant-1",
		ProviderID: res.Builder.ProviderType(), // Get type from builder
	}

	// GetClient now takes the builder directly from the result
	client := service.GetClient(ctx, cacheKey, res.Builder, res.Output, res.Config)
	if !client.IsPresent() {
		fmt.Println("failed to get client")
		return
	}

	fmt.Println("client created successfully")
	// Output: client created successfully
}

// Example: Multiple rules can share the same builder
func Example_sharedBuilder() {
	resolver := eddy.NewResolver[StorageClient, StorageCredentials, StorageConfig]()

	// Single builder instance shared by multiple rules
	s3Builder := &S3Builder{}

	// Production S3 rule
	resolver.AddRule(&helpers.ConditionalRule[StorageClient, StorageCredentials, StorageConfig]{
		Predicate: func(ctx context.Context) bool {
			env, ok := contextx.StringFrom[EnvironmentHint](ctx)
			return ok && env == "production"
		},
		Resolver: func(ctx context.Context) (*eddy.ResolvedProvider[StorageClient, StorageCredentials, StorageConfig], error) {
			return &eddy.ResolvedProvider[StorageClient, StorageCredentials, StorageConfig]{
				Builder: s3Builder, // Same builder
				Output: StorageCredentials{
					AccessKey: "prod-key",
					SecretKey: "prod-secret",
					Region:    "us-east-1",
				},
				Config: StorageConfig{
					Bucket:  "production-bucket",
					Timeout: 30 * time.Second,
				},
			}, nil
		},
	})

	// Development S3 rule
	resolver.AddRule(&helpers.ConditionalRule[StorageClient, StorageCredentials, StorageConfig]{
		Predicate: func(ctx context.Context) bool {
			env, ok := contextx.StringFrom[EnvironmentHint](ctx)
			return ok && env == "development"
		},
		Resolver: func(ctx context.Context) (*eddy.ResolvedProvider[StorageClient, StorageCredentials, StorageConfig], error) {
			return &eddy.ResolvedProvider[StorageClient, StorageCredentials, StorageConfig]{
				Builder: s3Builder, // Same builder, different creds
				Output: StorageCredentials{
					AccessKey: "dev-key",
					SecretKey: "dev-secret",
					Region:    "us-west-2",
				},
				Config: StorageConfig{
					Bucket:  "development-bucket",
					Timeout: 10 * time.Second,
				},
			}, nil
		},
	})

	ctx := context.Background()
	ctx = contextx.WithString(ctx, EnvironmentHint("production"))

	result := resolver.Resolve(ctx)
	if result.IsPresent() {
		res := result.MustGet()
		fmt.Printf("resolved provider type: %s\n", res.Builder.ProviderType())
	}
	// Output: resolved provider type: s3
}

// Example: Fallback chain with different builders
func Example_fallbackChain() {
	s3Builder := &S3Builder{}
	// Could have other builders like &R2Builder{}, &GCSBuilder{}, etc.

	resolver := eddy.NewResolver[StorageClient, StorageCredentials, StorageConfig]()

	// Try multiple credential sources in order
	resolver.AddRule(&helpers.FallbackChainRule[StorageClient, StorageCredentials, StorageConfig]{
		Resolvers: []func(context.Context) (*eddy.ResolvedProvider[StorageClient, StorageCredentials, StorageConfig], error){
			// Try database credentials first
			func(ctx context.Context) (*eddy.ResolvedProvider[StorageClient, StorageCredentials, StorageConfig], error) {
				// Would fetch from database...
				return nil, fmt.Errorf("no database credentials")
			},
			// Fall back to environment variables
			func(ctx context.Context) (*eddy.ResolvedProvider[StorageClient, StorageCredentials, StorageConfig], error) {
				return &eddy.ResolvedProvider[StorageClient, StorageCredentials, StorageConfig]{
					Builder: s3Builder,
					Output: StorageCredentials{
						AccessKey: "env-key",
						SecretKey: "env-secret",
						Region:    "us-east-1",
					},
					Config: StorageConfig{
						Bucket:  "env-bucket",
						Timeout: 30 * time.Second,
					},
				}, nil
			},
		},
	})

	ctx := context.Background()
	result := resolver.Resolve(ctx)

	if result.IsPresent() {
		res := result.MustGet()
		fmt.Printf("resolved with builder: %s\n", res.Builder.ProviderType())
	}
	// Output: resolved with builder: s3
}

// Example: Multitenancy with builder-in-rules
func Example_multitenancy() {
	s3Builder := &S3Builder{}

	resolver := eddy.NewResolver[StorageClient, StorageCredentials, StorageConfig]()

	// Each tenant can have different credentials, same builder
	resolver.AddRule(&helpers.MatchHintRule[StorageClient, StorageCredentials, StorageConfig, TenantHint]{
		Value: "tenant-A",
		Resolver: func(ctx context.Context) (*eddy.ResolvedProvider[StorageClient, StorageCredentials, StorageConfig], error) {
			return &eddy.ResolvedProvider[StorageClient, StorageCredentials, StorageConfig]{
				Builder: s3Builder,
				Output: StorageCredentials{
					AccessKey: "tenant-a-key",
					SecretKey: "tenant-a-secret",
					Region:    "us-east-1",
				},
				Config: StorageConfig{
					Bucket:  "tenant-a-bucket",
					Timeout: 30 * time.Second,
				},
			}, nil
		},
	})

	resolver.AddRule(&helpers.MatchHintRule[StorageClient, StorageCredentials, StorageConfig, TenantHint]{
		Value: "tenant-B",
		Resolver: func(ctx context.Context) (*eddy.ResolvedProvider[StorageClient, StorageCredentials, StorageConfig], error) {
			return &eddy.ResolvedProvider[StorageClient, StorageCredentials, StorageConfig]{
				Builder: s3Builder, // Same builder
				Output: StorageCredentials{
					AccessKey: "tenant-b-key",
					SecretKey: "tenant-b-secret",
					Region:    "us-west-2",
				},
				Config: StorageConfig{
					Bucket:  "tenant-b-bucket",
					Timeout: 30 * time.Second,
				},
			}, nil
		},
	})

	pool := eddy.NewClientPool[StorageClient](1 * time.Hour)
	service := eddy.NewClientService[StorageClient, StorageCredentials, StorageConfig](pool)

	// Tenant A request
	ctxA := context.Background()
	ctxA = contextx.WithString(ctxA, TenantHint("tenant-A"))

	resultA := resolver.Resolve(ctxA)
	if resultA.IsPresent() {
		resA := resultA.MustGet()
		cacheKeyA := StorageCacheKey{
			TenantID:   "tenant-A",
			ProviderID: resA.Builder.ProviderType(),
		}
		clientA := service.GetClient(ctxA, cacheKeyA, resA.Builder, resA.Output, resA.Config)
		if clientA.IsPresent() {
			fmt.Println("tenant-A client created")
		}
	}

	// Tenant B request
	ctxB := context.Background()
	ctxB = contextx.WithString(ctxB, TenantHint("tenant-B"))

	resultB := resolver.Resolve(ctxB)
	if resultB.IsPresent() {
		resB := resultB.MustGet()
		cacheKeyB := StorageCacheKey{
			TenantID:   "tenant-B",
			ProviderID: resB.Builder.ProviderType(),
		}
		clientB := service.GetClient(ctxB, cacheKeyB, resB.Builder, resB.Output, resB.Config)
		if clientB.IsPresent() {
			fmt.Println("tenant-B client created")
		}
	}

	// Output:
	// tenant-A client created
	// tenant-B client created
}
