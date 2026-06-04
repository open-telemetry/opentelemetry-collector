// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build js && wasm

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"syscall"
)

const (
	SIGHUP  = syscall.Signal(-1)
	SIGTERM = syscall.Signal(-1)
)
