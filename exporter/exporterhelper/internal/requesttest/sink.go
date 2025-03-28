// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package requesttest // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

func NewSink() *Sink {
	return &Sink{}
}

type Sink struct {
	requestsCount int
	itemsCount    int
	bytesCount    int
	mu            sync.Mutex
	exportErr     error
}

func (s *Sink) Export(ctx context.Context, req request.Request) error {
	r := req.(*FakeRequest)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(r.Delay):
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.exportErr != nil {
		err := s.exportErr
		s.exportErr = nil
		return err
	}
	if r.Partial > 0 {
		s.requestsCount++
		s.itemsCount += r.Items - r.Partial
		return errorPartial{fr: &FakeRequest{
			Items:    r.Partial,
			Partial:  0,
			MergeErr: r.MergeErr,
			Delay:    r.Delay,
		}}
	}
	s.requestsCount++
	s.itemsCount += r.Items
	s.bytesCount += r.Bytes
	return nil
}

func (s *Sink) SetExportErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exportErr = err
}

func (s *Sink) RequestsCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requestsCount
}

func (s *Sink) ItemsCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.itemsCount
}

func (s *Sink) BytesCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bytesCount
}
