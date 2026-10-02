// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !linux

package confignet // import "go.opentelemetry.io/collector/config/confignet"

import (
	"context"
	"errors"
	"net"
	"time"
)

var errVsockUnsupported = errors.New("vsock transport is only supported on Linux")

func dialVsock(_ context.Context, _ string, _ time.Duration) (net.Conn, error) {
	return nil, errVsockUnsupported
}

func listenVsock(_ string) (net.Listener, error) {
	return nil, errVsockUnsupported
}
