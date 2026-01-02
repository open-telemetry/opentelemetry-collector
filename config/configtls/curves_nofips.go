// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !requirefips

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import "crypto/tls"

// For backward compatability with older versions before Go 1.24.0
const (
	X25519MLKEM768 tls.CurveID = 0x11EC
)

var tlsCurveTypes = map[string]tls.CurveID{
	"P256":           tls.CurveP256,
	"P384":           tls.CurveP384,
	"P521":           tls.CurveP521,
	"X25519":         tls.X25519,
	"X25519MLKEM768": X25519MLKEM768,
}

// defaultCurvePreferences defines the default order of curve preferences.
// X25519MLKEM768 is prioritized for post-quantum security when compiled with Go 1.24+.
var defaultCurvePreferences = []tls.CurveID{
	X25519MLKEM768, // Post-quantum hybrid key exchange
	tls.X25519,     // Modern, fast elliptic curve
	tls.CurveP256,  // Widely supported
	tls.CurveP384,  // Higher security margin
	tls.CurveP521,  // Highest security margin
}
