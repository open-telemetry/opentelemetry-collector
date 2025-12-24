// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// package testhelper provides functionality used in tests in scraperhelper and xscraperhelper.

package testhelper // import "go.opentelemetry.io/collector/scraper/scraperhelper/internal/testhelper"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/collector/component"
)

type TestInitialize struct {
	ch  chan bool
	err error
}

func NewTestInitialize(ch chan bool, err error) *TestInitialize {
	return &TestInitialize{
		ch:  ch,
		err: err,
	}
}

func (ts *TestInitialize) Start(context.Context, component.Host) error {
	ts.ch <- true
	return ts.err
}

type TestClose struct {
	ch  chan bool
	err error
}

func NewTestClose(ch chan bool, err error) *TestClose {
	return &TestClose{
		ch:  ch,
		err: err,
	}
}

func (ts *TestClose) Shutdown(context.Context) error {
	ts.ch <- true
	return ts.err
}

func AssertChannelCalled(t *testing.T, ch chan bool, message string) {
	select {
	case <-ch:
	default:
		assert.Fail(t, message)
	}
}

func AssertChannelsCalled(t *testing.T, chs []chan bool, message string) {
	for _, ic := range chs {
		AssertChannelCalled(t, ic, message)
	}
}

func AssertScraperSpan(t *testing.T, expectedErr error, spans []sdktrace.ReadOnlySpan, expectedSpanName string) {
	expectedStatusCode := codes.Unset
	expectedStatusMessage := ""
	if expectedErr != nil {
		expectedStatusCode = codes.Error
		expectedStatusMessage = expectedErr.Error()
	}

	scraperSpan := false
	for _, span := range spans {
		if span.Name() == expectedSpanName {
			scraperSpan = true
			assert.Equal(t, expectedStatusCode, span.Status().Code)
			assert.Equal(t, expectedStatusMessage, span.Status().Description)
			break
		}
	}
	assert.True(t, scraperSpan)
}
