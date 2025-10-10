package eddy

// CacheKey is the interface that cache keys must implement
// Implementations must be comparable to work as map keys
type CacheKey interface {
	String() string
}
