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
	"X25519MLKEM768": tls.X25519MLKEM768,
}
