package eddy

import (
	"context"
	"errors"
	"testing"

	"github.com/samber/mo"
)

type resolverTestClient struct {
	name string
}

func TestResolverReturnsFirstMatchingResult(t *testing.T) {
	ctx := context.Background()
	resolver := NewResolver[*resolverTestClient, string, int]()

	firstRuleCalled := false
	secondRuleCalled := false

	resolver.AddRule(&RuleFunc[*resolverTestClient, string, int]{
		EvaluateFunc: func(context.Context) mo.Option[Result[*resolverTestClient, string, int]] {
			firstRuleCalled = true
			return mo.None[Result[*resolverTestClient, string, int]]()
		},
	})

	expected := &resolverTestClient{name: "second"}
	resolver.AddRule(&RuleFunc[*resolverTestClient, string, int]{
		EvaluateFunc: func(context.Context) mo.Option[Result[*resolverTestClient, string, int]] {
			secondRuleCalled = true
			return mo.Some(Result[*resolverTestClient, string, int]{
				Builder: &BuilderFunc[*resolverTestClient, string, int]{
					Type: "res",
					Func: func(context.Context, string, int) (*resolverTestClient, error) {
						return expected, nil
					},
				},
				Output: "creds",
				Config: 1,
			})
		},
	})

	result := resolver.Resolve(ctx)
	if !result.IsPresent() {
		t.Fatal("expected resolver to return a result")
	}

	if !firstRuleCalled || !secondRuleCalled {
		t.Fatalf("expected both rules to be evaluated, got first=%v second=%v", firstRuleCalled, secondRuleCalled)
	}

	res := result.MustGet()
	client, err := res.Builder.Build(ctx, res.Output, res.Config)
	if err != nil {
		t.Fatalf("unexpected error building client: %v", err)
	}
	if client != expected {
		t.Fatalf("expected client %v, got %v", expected, client)
	}
}

func TestResolverReturnsNoneWhenNoRulesMatch(t *testing.T) {
	resolver := NewResolver[*resolverTestClient, string, int]()

	resolver.AddRule(&RuleFunc[*resolverTestClient, string, int]{
		EvaluateFunc: func(context.Context) mo.Option[Result[*resolverTestClient, string, int]] {
			return mo.None[Result[*resolverTestClient, string, int]]()
		},
	})

	if result := resolver.Resolve(context.Background()); result.IsPresent() {
		t.Fatal("expected resolver to return none when no rules match")
	}
}

func TestRuleBuilderConditions(t *testing.T) {
	ctx := context.Background()

	builder := NewRule[*resolverTestClient, string, int]().
		WhenFunc(func(context.Context) bool { return true }).
		WhenFunc(func(context.Context) bool { return true })

	provider := &ResolvedProvider[*resolverTestClient, string, int]{
		Builder: &BuilderFunc[*resolverTestClient, string, int]{Type: "pass"},
		Output:  "creds",
		Config:  5,
	}

	rule := builder.Resolve(func(context.Context) (*ResolvedProvider[*resolverTestClient, string, int], error) {
		return provider, nil
	})

	result := rule.Evaluate(ctx)
	if !result.IsPresent() {
		t.Fatal("expected rule to return result when all conditions pass")
	}

	res := result.MustGet()
	if res.Builder.ProviderType() != "pass" {
		t.Fatalf("expected provider type 'pass', got %q", res.Builder.ProviderType())
	}

	if res.Output != provider.Output || res.Config != provider.Config {
		t.Fatalf("expected result to contain provider data")
	}
}

func TestRuleBuilderConditionFailureStopsEvaluation(t *testing.T) {
	ctx := context.Background()
	var resolverCalled bool

	rule := NewRule[*resolverTestClient, string, int]().
		WhenFunc(func(context.Context) bool { return false }).
		Resolve(func(context.Context) (*ResolvedProvider[*resolverTestClient, string, int], error) {
			resolverCalled = true
			return nil, nil
		})

	if result := rule.Evaluate(ctx); result.IsPresent() {
		t.Fatal("expected no result when condition fails")
	}

	if resolverCalled {
		t.Fatal("expected resolver not to be called when condition fails")
	}
}

func TestRuleBuilderResolverError(t *testing.T) {
	rule := NewRule[*resolverTestClient, string, int]().
		Resolve(func(context.Context) (*ResolvedProvider[*resolverTestClient, string, int], error) {
			return nil, errors.New("resolver failed")
		})

	if result := rule.Evaluate(context.Background()); result.IsPresent() {
		t.Fatal("expected none when resolver returns error")
	}
}
