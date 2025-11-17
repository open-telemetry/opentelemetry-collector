// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/pdata"

var DefaultUnmarshalOptions = UnmarshalOptions{
	LazyDecoding: false,
}

type UnmarshalOptions struct {
	LazyDecoding bool
}
