// Package helpers provides convenience helpers for common rule patterns
//
// These helpers are NOT part of the core API - they're batteries-included
// examples showing how to consume the proposed package. You can use them
// as-is or as inspiration for your own custom rules
//
// All helpers can be replicated using the core RuleFunc type directly
//
// Available helpers:
//   - MatchHintRule: Match a specific hint value
//   - MatchHintAnyRule: Match any of multiple hint values
//   - FallbackChainRule: Try multiple resolvers in order
//   - ConditionalRule: Custom predicate logic
package helpers
