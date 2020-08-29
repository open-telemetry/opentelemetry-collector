// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package obsreportscraper

import (
	"context"

	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/internal/dataold"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

type scraper struct {
	delegate              internal.Scraper
	scrapeMetricsSpanName string
}

// WrapScraper wraps an internal.Scraper and provides observability support.
func WrapScraper(delegate internal.Scraper, typeStr string) internal.Scraper {
	return &scraper{delegate: delegate, scrapeMetricsSpanName: scrapeMetricsSpanName(typeStr)}
}

func (s *scraper) Initialize(ctx context.Context) error {
	return s.delegate.Initialize(ctx)
}

func (s *scraper) Close(ctx context.Context) error {
	return s.delegate.Close(ctx)
}

// ScrapeMetrics
func (s *scraper) ScrapeMetrics(ctx context.Context) (dataold.MetricSlice, error) {
	// TODO: Add metrics.
	ctx, span := trace.StartSpan(ctx, s.scrapeMetricsSpanName)
	defer span.End()

	ms, err := s.delegate.ScrapeMetrics(ctx)

	if err != nil {
		span.SetStatus(trace.Status{Code: trace.StatusCodeUnknown, Message: err.Error()})
	}
	return ms, err
}
