// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

type Type int32

const (
	TypeDouble Type = iota
	TypeFloat

	TypeInt32
	TypeInt64
	TypeUint32
	TypeUint64

	TypeSInt32
	TypeSInt64

	TypeFixed32
	TypeFixed64
	TypeSFixed32
	TypeSFixed64

	TypeBool
	TypeEnum

	TypeString
	TypeBytes

	TypeMessage
)
