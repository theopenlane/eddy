[![Go Report Card](https://goreportcard.com/badge/github.com/theopenlane/eddy)](https://goreportcard.com/report/github.com/theopenlane/eddy)
[![Build status](https://badge.buildkite.com/a94b6f3d9a6f1ff4ff0de222f775d826a29d08a86e6414d6d8.svg)](https://buildkite.com/theopenlane/eddy?branch=main)
[![Go Reference](https://pkg.go.dev/badge/github.com/theopenlane/eddy.svg)](https://pkg.go.dev/github.com/theopenlane/eddy)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache2.0-brightgreen.svg)](https://opensource.org/licenses/Apache-2.0)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=theopenlane_eddye&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=theopenlane_eddy)

# eddy

`eddy` is a type-safe, rule-driven client resolution package for Go. It lets you:

- describe how to obtain service-specific clients via composable rules
- provide builders that know how to turn rule output into clients
- reuse cached client instances with per-tenant TTLs

Everything is expressed using Go generics so you keep static typing across the flow (builder → resolver → client service). Optional values travel through the system as [`mo.Option`](https://pkg.go.dev/github.com/samber/mo#Option), which keeps error handling explicit without resorting to sentinel values.

> The module targets Go `1.25`, but any recent Go release with generics support works fine.

## Installation

```bash
go get github.com/theopenlane/eddy
```

If you want the batteries-included helper rules (for context hints, fallback chains, etc.), also pull in:

```bash
go get github.com/theopenlane/eddy/helpers
```

## Core Building Blocks

- `Resolver`: evaluates rules in order and returns the first match as an `eddy.Result`.
- `Rule` / `RuleFunc`: your decision logic. Rules can return `mo.None` to signal “not me”.
- `Result`: bundles the `Builder`, output data (credentials), configuration, and a suggested `CacheKey`.
- `Builder`: turns a rule result into a concrete client. `BuilderFunc` adapts plain functions.
- `ClientPool`: a TTL-protected cache keyed by anything that satisfies the `CacheKey` interface.
- `ClientService`: reads from the pool, falls back to the supplied builder, and optionally clones rule data for safety.

Because everything is generic, you pick the client, output, and config types that make sense for your service surface.

## Caching Clients

Create a pool with the TTL that makes sense for your provider:

```go
pool := eddy.NewClientPool[*MyClient](30 * time.Minute)
```

`ClientService` wraps this pool and handles cache-aside logic. Defensive copies help when your resolver returns mutable state:

```go
service := eddy.NewClientService[*MyClient, Credentials, Config](
	pool,
	eddy.WithOutputClone[*MyClient](func(in Credentials) Credentials {
		return in.Clone()
	}),
	eddy.WithConfigClone[*MyClient](func(cfg Config) Config {
		return cfg.Clone()
	}),
)
```

`GetClient` returns an `mo.Option[*MyClient]`; if the builder returns an error you will see `mo.None` and the pool is left untouched. Periodically call `pool.CleanExpired()` if you want to eagerly discard stale entries instead of waiting for them to be overwritten.

## Building Rules Without Helpers

The fluent `RuleBuilder` covers many common patterns without leaving the core API:

```go
rule := eddy.NewRule[*MyClient, Credentials, Config]().
	WhenFunc(func(ctx context.Context) bool {
		return isTrustedTenant(ctx)
	}).
	WhenFunc(func(ctx context.Context) bool {
		return hasFeatureFlag(ctx, "beta-provider")
	}).
	Resolve(func(ctx context.Context) (*eddy.ResolvedProvider[*MyClient, Credentials, Config], error) {
		creds, cfg, err := loadProviderData(ctx)
		if err != nil {
			return nil, err
		}
		return &eddy.ResolvedProvider[*MyClient, Credentials, Config]{
			Builder: myBuilder,
			Output:  creds,
			Config:  cfg,
		}, nil
	})

resolver.AddRule(rule)
```

`RuleBuilder` short-circuits on the first failing condition, so expensive lookups only happen when everything else lines up.

## Helper Rules

The `helpers` package demonstrates how to consume the core types and covers common scenarios:

- `MatchHintRule`: match an exact typed string hint in the context.
- `MatchHintAnyRule`: match any of a set of hints.
- `FallbackChainRule`: try resolvers in sequence until one succeeds.
- `ConditionalRule`: wrap an arbitrary predicate plus resolver.

Every helper simply returns an `eddy.Result`, so mixing and matching with your own rules is seamless. The helpers rely on [`contextx`](https://github.com/theopenlane/utils/tree/main/contextx) for typed context values, but you can adapt them to any scheme you prefer.

## Working With Options

Most API surfaces return `mo.Option[T]`. The important helpers are:

- `IsPresent() bool` / `IsAbsent() bool`
- `MustGet()` – panics if absent, great for tests
- `OrElse(defaultValue)` – supply a fallback

Use whatever style fits your codebase best; the package stays unopinionated about error handling.

## Housekeeping Tips

- Call `ClientPool.RemoveClient` when you know a client should be evicted before TTL.
- Use `CleanExpired()` from a background job if you expect many idle entries.
- Keep your `CacheKey` deterministic and stringified; the library only needs something that implements `String() string`.
- Register builders per provider type and share them across rules; they are meant to be stateless factories.

## Advanced Example

The snippet below wires the major pieces together: a rule that resolves an S3-like client when a context hint is present, and a `ClientService` that caches instances per tenant.

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/theopenlane/eddy"
	"github.com/theopenlane/eddy/helpers"
	"github.com/theopenlane/utils/contextx"
)

type ProviderHint string

type storageClient struct {
	endpoint string
	bucket   string
}

type storageCreds struct {
	AccessKey string
	SecretKey string
	Region    string
}

type storageConfig struct {
	Bucket   string
	Endpoint string
	Timeout  time.Duration
}

type tenantCacheKey struct {
	TenantID string
	Type     string
}

func (k tenantCacheKey) String() string {
	return fmt.Sprintf("%s:%s", k.TenantID, k.Type)
}

func main() {
	ctx := contextx.WithString(context.Background(), ProviderHint("s3"))

	s3Builder := &eddy.BuilderFunc[*storageClient, storageCreds, storageConfig]{
		Type: "s3",
		Func: func(ctx context.Context, out storageCreds, cfg storageConfig) (*storageClient, error) {
			return &storageClient{endpoint: cfg.Endpoint, bucket: cfg.Bucket}, nil
		},
	}

	resolver := eddy.NewResolver[*storageClient, storageCreds, storageConfig]()
	resolver.AddRule(&helpers.MatchHintRule[*storageClient, storageCreds, storageConfig, ProviderHint]{
		Value: "s3",
		Resolver: func(context.Context) (*eddy.ResolvedProvider[*storageClient, storageCreds, storageConfig], error) {
			return &eddy.ResolvedProvider[*storageClient, storageCreds, storageConfig]{
				Builder: s3Builder,
				Output: storageCreds{
					AccessKey: "access",
					SecretKey: "secret",
					Region:    "us-east-1",
				},
				Config: storageConfig{
					Bucket:   "my-bucket",
					Endpoint: "s3.amazonaws.com",
					Timeout:  30 * time.Second,
				},
			}, nil
		},
	})

	result := resolver.Resolve(ctx)
	if !result.IsPresent() {
		panic("no provider matched")
	}

	pool := eddy.NewClientPool[*storageClient](time.Hour)
	service := eddy.NewClientService[*storageClient, storageCreds, storageConfig](pool)

	res := result.MustGet()
	cacheKey := tenantCacheKey{
		TenantID: "tenant-123",
		Type:     res.Builder.ProviderType(),
	}

	client := service.GetClient(ctx, cacheKey, res.Builder, res.Output, res.Config)
	if !client.IsPresent() {
		panic("builder failed")
	}

	fmt.Printf("reusing client for bucket %s\n", client.MustGet().bucket)
}
```

## License

Distributed under the [Apache 2.0 License](LICENSE).
