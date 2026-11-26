// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/service/extensions"
)

func TestHost_NotifyComponentStatusChange_ReportersNeverBlock(t *testing.T) {
	ch := make(chan error, 1)
	host := &Host{AsyncErrorChannel: ch, ServiceExtensions: &extensions.Extensions{}}

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			host.NotifyComponentStatusChange(nil, componentstatus.NewFatalErrorEvent(assert.AnError))
		})
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("reporters blocked")
	}

	require.ErrorIs(t, <-ch, assert.AnError) // exactly one error retained
	assert.Empty(t, ch)                      // the other 99 were dropped
}

func TestHost_NotifyComponentStatusChange_FullChannelKeepsFirst(t *testing.T) {
	ch := make(chan error, 1)
	first := errors.New("first")
	ch <- first
	host := &Host{AsyncErrorChannel: ch, ServiceExtensions: &extensions.Extensions{}}

	host.NotifyComponentStatusChange(nil, componentstatus.NewFatalErrorEvent(errors.New("second")))

	require.ErrorIs(t, <-ch, first)
	assert.Empty(t, ch)
}
