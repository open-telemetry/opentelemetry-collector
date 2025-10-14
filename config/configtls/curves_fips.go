// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build requirefips

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import "crypto/tls"

var tlsCurveTypes = map[string]tls.CurveID{
	"P256": tls.CurveP256,
	"P384": tls.CurveP384,
	"P521": tls.CurveP521,

	// The following X25519 curves are not available in FIPS mode, so we remove them from the map.
	// See also https://cs.opensource.google/go/go/+/refs/tags/go1.24.6:src/crypto/ecdh/x25519.go
	//"X25519": tls.X25519,
	//"X25519MLKEM768": tls.X25519MLKEM768,
}
