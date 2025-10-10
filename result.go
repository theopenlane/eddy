package eddy

import (
	"context"

	"github.com/samber/mo"
)

// Result represents the output of rule evaluation
type Result[T any, Output any, Config any] struct {
	// Builder is the client builder to use
	Builder Builder[T, Output, Config]
	// Output contains the credentials or output data needed to build the client
	Output Output
	// Config contains the configuration for the client
	Config Config
	// CacheKey is the key used to cache the built client
	CacheKey CacheKey
}

// Rule is a generic interface that evaluates context and returns a result
type Rule[T any, Output any, Config any] interface {
	Evaluate(ctx context.Context) mo.Option[Result[T, Output, Config]]
}

// RuleFunc is a function adapter for Rule interface
type RuleFunc[T any, Output any, Config any] struct {
	// EvaluateFunc is the function that evaluates the rule
	EvaluateFunc func(ctx context.Context) mo.Option[Result[T, Output, Config]]
}

// Evaluate implements Rule.Evaluate
func (r *RuleFunc[T, Output, Config]) Evaluate(ctx context.Context) mo.Option[Result[T, Output, Config]] {
	return r.EvaluateFunc(ctx)
}

// Resolver is a generic struct that handles rule-based resolution
type Resolver[T any, Output any, Config any] struct {
	rules []Rule[T, Output, Config]
}

// NewResolver creates a new resolver instance
func NewResolver[T any, Output any, Config any]() *Resolver[T, Output, Config] {
	return &Resolver[T, Output, Config]{
		rules: make([]Rule[T, Output, Config], 0),
	}
}

// AddRule adds a resolution rule to the resolver
func (r *Resolver[T, Output, Config]) AddRule(rule Rule[T, Output, Config]) *Resolver[T, Output, Config] {
	r.rules = append(r.rules, rule)
	return r
}

// Resolve evaluates rules in order and returns the first matching result
func (r *Resolver[T, Output, Config]) Resolve(ctx context.Context) mo.Option[Result[T, Output, Config]] {
	for _, rule := range r.rules {
		if result := rule.Evaluate(ctx); result.IsPresent() {
			return result
		}
	}

	return mo.None[Result[T, Output, Config]]()
}
