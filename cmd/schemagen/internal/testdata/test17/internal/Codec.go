// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

// Codec is a named interface type of another package.
type Codec interface {
	Encode() ([]byte, error)
}
