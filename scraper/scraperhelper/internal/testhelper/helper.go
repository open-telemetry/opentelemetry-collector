// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// package testhelper provides functionality used in tests in scraperhelper and xscraperhelper.

package testhelper // import "go.opentelemetry.io/collector/scraper/scraperhelper/internal/testhelper"

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/extensionscrapercontroller"
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

// MockHost implements component.Host with a configurable extensions map.
type MockHost struct {
	Extensions map[component.ID]component.Component
}

func (h *MockHost) GetExtensions() map[component.ID]component.Component {
	return h.Extensions
}

// MockControllerExtension implements extensionscrapercontroller.ControllerExtension.
// Its DeregisterFunc blocks until all in-flight Scrape calls have returned, as
// required by the ControllerExtension contract.
type MockControllerExtension struct {
	component.StartFunc
	component.ShutdownFunc
	RegisteredFunc extensionscrapercontroller.ScrapeFunc
	Deregistered   atomic.Bool
	RegisterErr    error
	DeregisterErr  error
	scrapeWG       sync.WaitGroup
}

func (m *MockControllerExtension) RegisterScraper(_ context.Context, scrapeFunc extensionscrapercontroller.ScrapeFunc) (extensionscrapercontroller.DeregisterFunc, error) {
	if m.RegisterErr != nil {
		return nil, m.RegisterErr
	}
	m.RegisteredFunc = scrapeFunc
	return func(context.Context) error {
		m.Deregistered.Store(true)
		m.scrapeWG.Wait()
		return m.DeregisterErr
	}, nil
}

// Scrape invokes the registered ScrapeFunc, tracking it via an internal
// WaitGroup so the DeregisterFunc blocks until it completes.
func (m *MockControllerExtension) Scrape(ctx context.Context) error {
	if m.RegisteredFunc == nil {
		return nil
	}
	m.scrapeWG.Add(1)
	defer m.scrapeWG.Done()
	return m.RegisteredFunc(ctx)
}
