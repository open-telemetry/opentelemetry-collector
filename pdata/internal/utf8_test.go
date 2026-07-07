// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import "testing"

func TestValidateUTF8(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if !ValidateUTF8(nil) {
			t.Fatal("expected nil to be valid")
		}
	})

	t.Run("valid composite", func(t *testing.T) {
		type sample struct {
			S string
			A []string
		}
		if !ValidateUTF8(&sample{S: "ok", A: []string{"a", "b"}}) {
			t.Fatal("expected valid composite to be valid")
		}
	})

	t.Run("invalid string", func(t *testing.T) {
		if ValidateUTF8(string([]byte{0xff})) {
			t.Fatal("expected invalid string to be rejected")
		}
	})

	t.Run("invalid nested composite", func(t *testing.T) {
		type sample struct {
			A [1]string
		}
		if ValidateUTF8(sample{A: [1]string{string([]byte{0xff})}}) {
			t.Fatal("expected invalid nested value to be rejected")
		}
	})
}
