package helpers

import (
	"context"
	"errors"
	"testing"

	"github.com/theopenlane/eddy"
	"github.com/theopenlane/utils/contextx"
)

type ruleTestClient struct {
	name string
}

type providerHint string

func TestMatchHintRuleEvaluate(t *testing.T) {
	ctx := context.Background()
	ctx = contextx.WithString(ctx, providerHint("match"))

	builder := &eddy.BuilderFunc[*ruleTestClient, string, int]{Type: "builder"}
	rule := &MatchHintRule[*ruleTestClient, string, int, providerHint]{
		Value: "match",
		Resolver: func(context.Context) (*eddy.ResolvedProvider[*ruleTestClient, string, int], error) {
			return &eddy.ResolvedProvider[*ruleTestClient, string, int]{
				Builder: builder,
				Output:  "creds",
				Config:  1,
			}, nil
		},
	}

	result := rule.Evaluate(ctx)
	if !result.IsPresent() {
		t.Fatal("expected match hint rule to return result")
	}

	res := result.MustGet()
	if res.Builder != builder {
		t.Fatal("expected builder to be propagated from resolver")
	}
}

func TestMatchHintRuleEvaluateNoMatch(t *testing.T) {
	rule := &MatchHintRule[*ruleTestClient, string, int, providerHint]{
		Value: "expected",
		Resolver: func(context.Context) (*eddy.ResolvedProvider[*ruleTestClient, string, int], error) {
			return &eddy.ResolvedProvider[*ruleTestClient, string, int]{}, nil
		},
	}

	if result := rule.Evaluate(context.Background()); result.IsPresent() {
		t.Fatal("expected no result when hint missing")
	}
}

func TestMatchHintRuleEvaluateResolverError(t *testing.T) {
	ctx := contextx.WithString(context.Background(), providerHint("match"))
	rule := &MatchHintRule[*ruleTestClient, string, int, providerHint]{
		Value: "match",
		Resolver: func(context.Context) (*eddy.ResolvedProvider[*ruleTestClient, string, int], error) {
			return nil, errors.New("resolver failed")
		},
	}

	if result := rule.Evaluate(ctx); result.IsPresent() {
		t.Fatal("expected no result when resolver returns error")
	}
}

func TestMatchHintAnyRuleEvaluate(t *testing.T) {
	ctx := contextx.WithString(context.Background(), providerHint("value-2"))
	builder := &eddy.BuilderFunc[*ruleTestClient, string, int]{Type: "any"}
	rule := &MatchHintAnyRule[*ruleTestClient, string, int, providerHint]{
		Values: []providerHint{"value-1", "value-2"},
		Resolver: func(context.Context) (*eddy.ResolvedProvider[*ruleTestClient, string, int], error) {
			return &eddy.ResolvedProvider[*ruleTestClient, string, int]{Builder: builder}, nil
		},
	}

	if result := rule.Evaluate(ctx); !result.IsPresent() {
		t.Fatal("expected rule to match one of provided hints")
	}
}

func TestMatchHintAnyRuleEvaluateNoMatch(t *testing.T) {
	ctx := contextx.WithString(context.Background(), providerHint("value-3"))
	rule := &MatchHintAnyRule[*ruleTestClient, string, int, providerHint]{
		Values: []providerHint{"value-1", "value-2"},
		Resolver: func(context.Context) (*eddy.ResolvedProvider[*ruleTestClient, string, int], error) {
			return &eddy.ResolvedProvider[*ruleTestClient, string, int]{}, nil
		},
	}

	if result := rule.Evaluate(ctx); result.IsPresent() {
		t.Fatal("expected no result when hint not in list")
	}
}

func TestFallbackChainRule(t *testing.T) {
	builder := &eddy.BuilderFunc[*ruleTestClient, string, int]{Type: "fallback"}

	rule := &FallbackChainRule[*ruleTestClient, string, int]{
		Resolvers: []func(context.Context) (*eddy.ResolvedProvider[*ruleTestClient, string, int], error){
			func(context.Context) (*eddy.ResolvedProvider[*ruleTestClient, string, int], error) {
				return nil, errors.New("first failed")
			},
			func(context.Context) (*eddy.ResolvedProvider[*ruleTestClient, string, int], error) {
				return &eddy.ResolvedProvider[*ruleTestClient, string, int]{Builder: builder}, nil
			},
		},
	}

	result := rule.Evaluate(context.Background())
	if !result.IsPresent() {
		t.Fatal("expected fallback rule to return second resolver result")
	}

	if result.MustGet().Builder != builder {
		t.Fatal("expected resolver result to be returned")
	}
}

func TestFallbackChainRuleNoResolverSucceeds(t *testing.T) {
	rule := &FallbackChainRule[*ruleTestClient, string, int]{
		Resolvers: []func(context.Context) (*eddy.ResolvedProvider[*ruleTestClient, string, int], error){
			func(context.Context) (*eddy.ResolvedProvider[*ruleTestClient, string, int], error) {
				return nil, errors.New("first failed")
			},
			func(context.Context) (*eddy.ResolvedProvider[*ruleTestClient, string, int], error) {
				return nil, nil
			},
		},
	}

	if result := rule.Evaluate(context.Background()); result.IsPresent() {
		t.Fatal("expected none when all resolvers fail")
	}
}

func TestConditionalRuleEvaluate(t *testing.T) {
	builder := &eddy.BuilderFunc[*ruleTestClient, string, int]{Type: "conditional"}

	rule := &ConditionalRule[*ruleTestClient, string, int]{
		Predicate: func(ctx context.Context) bool {
			v, ok := contextx.StringFrom[providerHint](ctx)
			return ok && v == "allowed"
		},
		Resolver: func(context.Context) (*eddy.ResolvedProvider[*ruleTestClient, string, int], error) {
			return &eddy.ResolvedProvider[*ruleTestClient, string, int]{Builder: builder}, nil
		},
	}

	ctx := contextx.WithString(context.Background(), providerHint("allowed"))
	if result := rule.Evaluate(ctx); !result.IsPresent() {
		t.Fatal("expected predicate passing to return result")
	}

	ctx = contextx.WithString(context.Background(), providerHint("denied"))
	if result := rule.Evaluate(ctx); result.IsPresent() {
		t.Fatal("expected predicate failing to return none")
	}
}

func TestConditionalRuleResolverError(t *testing.T) {
	rule := &ConditionalRule[*ruleTestClient, string, int]{
		Predicate: func(context.Context) bool { return true },
		Resolver: func(context.Context) (*eddy.ResolvedProvider[*ruleTestClient, string, int], error) {
			return nil, errors.New("failed")
		},
	}

	if result := rule.Evaluate(context.Background()); result.IsPresent() {
		t.Fatal("expected none when resolver returns error")
	}
}
