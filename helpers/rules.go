package helpers

import (
	"context"
	"slices"

	"github.com/samber/mo"
	"github.com/theopenlane/eddy"
	"github.com/theopenlane/utils/contextx"
)

// MatchHintRule creates a rule that matches a hint value using typed strings from context
//
// Example:
//
//	type ProviderHint string
//	rule := &MatchHintRule[StorageClient, StorageCredentials, StorageConfig, ProviderHint]{
//	    Value: "s3",
//	    Resolver: resolveS3Provider,
//	}
type MatchHintRule[T any, Output any, Config any, HintType ~string] struct {
	// Value is the expected hint value
	Value HintType
	// Resolver is called when the hint matches to resolve the provider
	Resolver func(context.Context) (*eddy.ResolvedProvider[T, Output, Config], error)
}

// Evaluate implements Rule.Evaluate
func (r *MatchHintRule[T, Output, Config, HintType]) Evaluate(ctx context.Context) mo.Option[eddy.Result[T, Output, Config]] {
	hint, ok := contextx.StringFrom[HintType](ctx)
	if !ok || hint != r.Value {
		return mo.None[eddy.Result[T, Output, Config]]()
	}

	provider, err := r.Resolver(ctx)
	if err != nil || provider == nil {
		return mo.None[eddy.Result[T, Output, Config]]()
	}

	return mo.Some(eddy.Result[T, Output, Config]{
		Builder: provider.Builder,
		Output:  provider.Output,
		Config:  provider.Config,
	})
}

// MatchHintAnyRule creates a rule that matches any of the provided hint values using typed strings from context
//
// Example:
//
//	type ProviderHint string
//	rule := &MatchHintAnyRule[StorageClient, StorageCredentials, StorageConfig, ProviderHint]{
//	    Values: []ProviderHint{"s3", "r2"},
//	    Resolver: resolveObjectStorageProvider,
//	}
type MatchHintAnyRule[T any, Output any, Config any, HintType ~string] struct {
	// Values are the acceptable hint values
	Values []HintType
	// Resolver is called when the hint matches any value to resolve the provider
	Resolver func(context.Context) (*eddy.ResolvedProvider[T, Output, Config], error)
}

// Evaluate implements Rule.Evaluate
func (r *MatchHintAnyRule[T, Output, Config, HintType]) Evaluate(ctx context.Context) mo.Option[eddy.Result[T, Output, Config]] {
	hint, ok := contextx.StringFrom[HintType](ctx)
	if !ok {
		return mo.None[eddy.Result[T, Output, Config]]()
	}

	matched := slices.Contains(r.Values, hint)

	if !matched {
		return mo.None[eddy.Result[T, Output, Config]]()
	}

	provider, err := r.Resolver(ctx)
	if err != nil || provider == nil {
		return mo.None[eddy.Result[T, Output, Config]]()
	}

	return mo.Some(eddy.Result[T, Output, Config]{
		Builder: provider.Builder,
		Output:  provider.Output,
		Config:  provider.Config,
	})
}

// FallbackChainRule tries resolvers in order until one succeeds
//
// Example:
//
//	rule := &FallbackChainRule[StorageClient, StorageCredentials, StorageConfig]{
//	    Resolvers: []func(context.Context) (*eddy.ResolvedProvider[StorageCredentials, StorageConfig], error){
//	        resolveDatabaseCredentials,
//	        resolveEnvCredentials,
//	        resolveDefaultCredentials,
//	    },
//	}
type FallbackChainRule[T any, Output any, Config any] struct {
	// Resolvers are tried in order until one succeeds
	Resolvers []func(context.Context) (*eddy.ResolvedProvider[T, Output, Config], error)
}

// Evaluate implements Rule.Evaluate
func (r *FallbackChainRule[T, Output, Config]) Evaluate(ctx context.Context) mo.Option[eddy.Result[T, Output, Config]] {
	for _, resolver := range r.Resolvers {
		provider, err := resolver(ctx)
		if err == nil && provider != nil {
			return mo.Some(eddy.Result[T, Output, Config]{
				Builder: provider.Builder,
				Output:  provider.Output,
				Config:  provider.Config,
			})
		}
	}
	return mo.None[eddy.Result[T, Output, Config]]()
}

// ConditionalRule creates a rule with a custom predicate
//
// Example:
//
//	rule := &ConditionalRule[StorageClient, StorageCredentials, StorageConfig]{
//	    Predicate: func(ctx context.Context) bool {
//	        return auth.IsSystemAdmin(ctx)
//	    },
//	    Resolver: resolveAdminProvider,
//	}
type ConditionalRule[T any, Output any, Config any] struct {
	// Predicate is called to determine if this rule matches
	Predicate func(context.Context) bool
	// Resolver is called when the predicate returns true to resolve the provider
	Resolver func(context.Context) (*eddy.ResolvedProvider[T, Output, Config], error)
}

// Evaluate implements Rule.Evaluate
func (r *ConditionalRule[T, Output, Config]) Evaluate(ctx context.Context) mo.Option[eddy.Result[T, Output, Config]] {
	if !r.Predicate(ctx) {
		return mo.None[eddy.Result[T, Output, Config]]()
	}

	provider, err := r.Resolver(ctx)
	if err != nil || provider == nil {
		return mo.None[eddy.Result[T, Output, Config]]()
	}

	return mo.Some(eddy.Result[T, Output, Config]{
		Builder: provider.Builder,
		Output:  provider.Output,
		Config:  provider.Config,
	})
}
