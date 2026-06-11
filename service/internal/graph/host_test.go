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

	done := make(chan struct{})
	go func() {
		ev := componentstatus.NewFatalErrorEvent(assert.AnError)
		host.NotifyComponentStatusChange(nil, ev)
		close(done)
	}()

	// The send should not block even though the channel is unbuffered
	// and nothing is reading from it.
	<-done
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
		assert.ErrorIs(t, err, assert.AnError)
	default:
		t.Fatal("expected error to be sent to channel")
	}
}
