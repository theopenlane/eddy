package eddy

import (
	"context"

	"github.com/samber/mo"
)

// NewRule creates a rule builder for static resolution
func NewRule[T any, Output any, Config any]() *RuleBuilder[T, Output, Config] {
	return &RuleBuilder[T, Output, Config]{}
}

// RuleBuilder provides a fluent interface for creating resolution rules
type RuleBuilder[T any, Output any, Config any] struct {
	conditions []func(context.Context) bool
}

// WhenFunc adds a custom condition function
func (b *RuleBuilder[T, Output, Config]) WhenFunc(condition func(context.Context) bool) *RuleBuilder[T, Output, Config] {
	b.conditions = append(b.conditions, condition)
	return b
}

// ResolvedProvider represents a resolved provider configuration
// This is returned by resolver functions
type ResolvedProvider[T any, Output any, Config any] struct {
	// Builder is the client builder to use
	Builder Builder[T, Output, Config]
	// Output contains the credentials or output data needed to build the client
	Output Output
	// Config contains the configuration for the client
	Config Config
}

// Resolve creates a rule that uses a function to resolve the provider
func (b *RuleBuilder[T, Output, Config]) Resolve(resolver func(context.Context) (*ResolvedProvider[T, Output, Config], error)) Rule[T, Output, Config] {
	conditions := b.conditions
	return &RuleFunc[T, Output, Config]{
		EvaluateFunc: func(ctx context.Context) mo.Option[Result[T, Output, Config]] {
			for _, condition := range conditions {
				if !condition(ctx) {
					return mo.None[Result[T, Output, Config]]()
				}
			}

			provider, err := resolver(ctx)
			if err != nil || provider == nil {
				return mo.None[Result[T, Output, Config]]()
			}

			return mo.Some(Result[T, Output, Config]{
				Builder: provider.Builder,
				Output:  provider.Output,
				Config:  provider.Config,
			})
		},
	}
}
