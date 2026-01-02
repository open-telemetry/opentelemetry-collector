//go:build go1.24
// +build go1.24

package configtls

import (
	"crypto/tls"
	"testing"
)

func TestX25519MLKEM768Value(t *testing.T) {
	if X25519MLKEM768 != tls.X25519MLKEM768 {
		t.Errorf("X25519MLKEM768 constant (%v) does not match tls.X25519MLKEM768 (%v)",
			X25519MLKEM768, tls.X25519MLKEM768)
	}
}
