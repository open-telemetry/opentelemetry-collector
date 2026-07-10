// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/service/extensions"
)

func TestHost_NotifyComponentStatusChange_NonBlockingOnFullChannel(t *testing.T) {
	ch := make(chan error, 1)
	queuedErr := errors.New("queued error")
	ch <- queuedErr
	host := &Host{
		AsyncErrorChannel: ch,
		ServiceExtensions: &extensions.Extensions{},
	}

	ev := componentstatus.NewFatalErrorEvent(errors.New("new error"))
	host.NotifyComponentStatusChange(nil, ev)

	// A full channel should preserve the error that is already queued.
	require.ErrorIs(t, <-ch, queuedErr)
	require.Empty(t, ch)
}

func TestHost_NotifyComponentStatusChange_SendsWhenChannelReady(t *testing.T) {
	ch := make(chan error, 1)
	host := &Host{
		AsyncErrorChannel: ch,
		ServiceExtensions: &extensions.Extensions{},
	}

	ev := componentstatus.NewFatalErrorEvent(assert.AnError)
	host.NotifyComponentStatusChange(nil, ev)

	select {
	case err := <-ch:
		require.ErrorIs(t, err, assert.AnError)
	default:
		t.Fatal("expected error to be sent to channel")
	}
}
