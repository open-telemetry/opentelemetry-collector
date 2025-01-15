// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/scraper/scrapererror"
	"go.opentelemetry.io/collector/scraper/scraperhelper/internal/metadatatest"
)

var (
	receiverID = component.MustNewID("fakeReceiver")
	scraperID  = component.MustNewID("fakeScraper")

	errFake        = errors.New("errFake")
	partialErrFake = scrapererror.NewPartialScrapeError(errFake, 2)
)

type testParams struct {
	items int
	err   error
}

func TestScrapeMetricsDataOp(t *testing.T) {
	tt := metadatatest.SetupTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	tel := tt.NewTelemetrySettings()
	// TODO: Add capability for tracing testing in metadatatest.
	spanRecorder := new(tracetest.SpanRecorder)
	tel.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))

	parentCtx, parentSpan := tel.TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	params := []testParams{
		{items: 23, err: partialErrFake},
		{items: 29, err: errFake},
		{items: 15, err: nil},
	}
	for i := range params {
		sf, err := newObsMetrics(func(context.Context) (pmetric.Metrics, error) {
			return testdata.GenerateMetrics(params[i].items), params[i].err
		}, receiverID, scraperID, tel)
		require.NoError(t, err)
		_, err = sf.ScrapeMetrics(parentCtx)
		require.ErrorIs(t, err, params[i].err)
	}

	spans := spanRecorder.Ended()
	require.Equal(t, len(params), len(spans))

	var scrapedMetricPoints, erroredMetricPoints int
	for i, span := range spans {
		assert.Equal(t, "scraper/"+scraperID.String()+"/ScrapeMetrics", span.Name())
		switch {
		case params[i].err == nil:
			scrapedMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.Int64(scrapedMetricPointsKey, int64(params[i].items)))
			require.Contains(t, span.Attributes(), attribute.Int64(erroredMetricPointsKey, 0))
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			// Since we get an error, we cannot record any metrics because we don't know if the returned pmetric.Metrics is valid instance.
			require.Contains(t, span.Attributes(), attribute.Int64(scrapedMetricPointsKey, 0))
			require.Contains(t, span.Attributes(), attribute.Int64(erroredMetricPointsKey, 0))
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		case errors.Is(params[i].err, partialErrFake):
			scrapedMetricPoints += params[i].items
			erroredMetricPoints += 2
			require.Contains(t, span.Attributes(), attribute.Int64(scrapedMetricPointsKey, int64(params[i].items)))
			require.Contains(t, span.Attributes(), attribute.Int64(erroredMetricPointsKey, 2))
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected err param: %v", params[i].err)
		}
	}

	checkScraperMetrics(t, tt, receiverID, scraperID, int64(scrapedMetricPoints), int64(erroredMetricPoints))
}

func TestCheckScraperMetrics(t *testing.T) {
	tt := metadatatest.SetupTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	sf, err := newObsMetrics(func(context.Context) (pmetric.Metrics, error) {
		return testdata.GenerateMetrics(7), nil
	}, receiverID, scraperID, tt.NewTelemetrySettings())
	require.NoError(t, err)
	_, err = sf.ScrapeMetrics(context.Background())
	require.NoError(t, err)

	checkScraperMetrics(t, tt, receiverID, scraperID, 7, 0)
}

func checkScraperMetrics(t *testing.T, tt metadatatest.Telemetry, receiver component.ID, scraper component.ID, scrapedMetricPoints, erroredMetricPoints int64) {
	tt.AssertMetrics(t, []metricdata.Metrics{
		{
			Name:        "otelcol_scraper_scraped_metric_points",
			Description: "Number of metric points successfully scraped. [alpha]",
			Unit:        "{datapoints}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Attributes: attribute.NewSet(
							attribute.String(receiverKey, receiver.String()),
							attribute.String(scraperKey, scraper.String())),
						Value: scrapedMetricPoints,
					},
				},
			},
		},
		{
			Name:        "otelcol_scraper_errored_metric_points",
			Description: "Number of metric points that were unable to be scraped. [alpha]",
			Unit:        "{datapoints}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Attributes: attribute.NewSet(
							attribute.String(receiverKey, receiver.String()),
							attribute.String(scraperKey, scraper.String())),
						Value: erroredMetricPoints,
					},
				},
			},
		},
	}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}
