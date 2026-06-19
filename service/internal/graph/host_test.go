// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/service/extensions"
)

func TestHost_NotifyComponentStatusChange_NonBlockingOnFullChannel(t *testing.T) {
	ch := make(chan error)
	host := &Host{
		AsyncErrorChannel: ch,
		ServiceExtensions: &extensions.Extensions{},
	}

	ev := componentstatus.NewFatalErrorEvent(assert.AnError)
	host.NotifyComponentStatusChange(nil, ev)

	// Non-blocking send should return immediately without blocking.
	// No goroutine should be leaked.
	select {
	case <-ch:
		t.Fatal("expected channel to be empty since no reader is running")
	default:
	}
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
