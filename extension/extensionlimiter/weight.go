// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

// WeightKey is an enum type for common rate limits.  The
// StandardAllKeys, StandardMiddlewareKeys, and
// StandardNotMiddlewareKeys methods return the list of middleware
// keys that can be automatically configured through middleware and
// not.
type WeightKey string

// Predefined weight keys for common rate limits.  This is not meant
// to be a closed set, new weight keys may be added in the future,
// possibly to restrict other kinds of event (e.g., auths, retries).
//
// Providers should return errors when they do not recognize a weight
// key.
const (
	// WeightKeyNetworkBytes is for network bytes. This is
	// typically used with rate limiters.
	WeightKeyNetworkBytes WeightKey = "network_bytes"

	// WeightKeyRequestCount can be used to limit the rate or
	// total concurrent number of requests (i.e., pipeline data
	// objects). This is typically used with both rate and
	// resource limiters.
	WeightKeyRequestCount WeightKey = "request_count"

	// WeightKeyRequestItems can be used to limit the rate or
	// total concurrent number of items (log records, metric data
	// points, spans, profiles).  This is typically used with both
	// rate and resource limiters.
	WeightKeyRequestItems WeightKey = "request_items"

	// WeightKeyMemorySize is typically used with ResourceLimiters
	// for limiting active memory usage.
	WeightKeyMemorySize WeightKey = "memory_size"
)

// StandardAllKeys is all the keys that can be automatically
// implemented by middleware and/or limiterhelper.
func StandardAllKeys() []WeightKey {
	return []WeightKey{
		WeightKeyNetworkBytes,
		WeightKeyRequestCount,
		WeightKeyRequestItems,
		WeightKeyMemorySize,
	}
}

// StandardMiddlewareKeys are typically handled in middleware for
// protocols that support it.  Receivers should be careful not to
// re-apply these limits, especially not to twice-limit by
// WeightKeyRequestItems.
func StandardMiddlewareKeys() []WeightKey {
	return []WeightKey{
		WeightKeyNetworkBytes,
		WeightKeyRequestCount,
	}
}

// StandardNotMiddlewareKeys are the keys that are typically not
// handled through middlware because they are protocol specific and
// generally easier to handle after the input has become pdata.
func StandardNotMiddlewareKeys() []WeightKey {
	return []WeightKey{
		WeightKeyRequestItems,
		WeightKeyMemorySize,
	}
}
