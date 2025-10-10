// package eddy contains the proposed v2 API for the cp package.
//
// # Core API Surface
//
// The core package provides the fundamental building blocks:
//   - Rule interface - evaluates context and returns optional Result
//   - RuleFunc - function adapter for Rule
//   - Resolver - evaluates rules in order, returns first match
//   - Result - contains Builder, Output, and Config
//   - Builder interface - builds client instances
//   - ClientService - manages client pooling and caching
//   - ClientPool - thread-safe cache with TTL expiration
//   - CacheKey interface - generic cache key abstraction
//   - ResolvedProvider - returned by resolver functions
//
// # Helpers Package
//
// The helpers/ subdirectory contains convenience rules for common patterns.
// These are NOT part of the core API - they're batteries-included examples
// showing how to consume the core package. You can use them or build your own.
//
// Available helpers:
//   - MatchHintRule - match a specific hint value from context
//   - MatchHintAnyRule - match any of multiple hint values from context
//   - FallbackChainRule - try resolvers in order
//   - ConditionalRule - custom predicate logic
//
// # Key Design Decisions
//
//   - Rules provide Builder instances directly in their results
//   - No central builder registry - simpler dispatch model
//   - Builder is a stateless factory, shared across rules as needed
//   - Cache distinguishes clients by TenantID + Builder.ProviderType()
//   - Integration with contextx for type-safe context values
//
// # Context Value Patterns
//
//   - Use contextx.With/From for singleton data (one value per type)
//   - Use contextx.WithString/StringFrom with typed strings for multiple distinct string values
//
// Compare this package with pkg/cp to see the differences.
package eddy
