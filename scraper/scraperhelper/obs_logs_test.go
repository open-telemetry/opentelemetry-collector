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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/scraper/scraperhelper/internal/metadatatest"
)

func TestScrapeLogsDataOp(t *testing.T) {
	tt := metadatatest.SetupTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	tel := tt.NewTelemetrySettings()

	parentCtx, parentSpan := tel.TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	params := []testParams{
		{items: 23, err: partialErrFake},
		{items: 29, err: errFake},
		{items: 15, err: nil},
	}
	for i := range params {
		sf, err := newObsLogs(func(context.Context) (plog.Logs, error) {
			return testdata.GenerateLogs(params[i].items), params[i].err
		}, receiverID, scraperID, tel)
		require.NoError(t, err)
		_, err = sf.ScrapeLogs(parentCtx)
		require.ErrorIs(t, err, params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Equal(t, len(params), len(spans))

	var scrapedLogRecords, erroredLogRecords int
	for i, span := range spans {
		assert.Equal(t, "scraper/"+scraperID.String()+"/ScrapeLogs", span.Name())
		switch {
		case params[i].err == nil:
			scrapedLogRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.Int64(scrapedLogRecordsKey, int64(params[i].items)))
			require.Contains(t, span.Attributes(), attribute.Int64(erroredLogRecordsKey, 0))
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			// Since we get an error, we cannot record any metrics because we don't know if the returned plog.Logs is valid instance.
			require.Contains(t, span.Attributes(), attribute.Int64(scrapedLogRecordsKey, 0))
			require.Contains(t, span.Attributes(), attribute.Int64(erroredLogRecordsKey, 0))
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		case errors.Is(params[i].err, partialErrFake):
			scrapedLogRecords += params[i].items
			erroredLogRecords += 2
			require.Contains(t, span.Attributes(), attribute.Int64(scrapedLogRecordsKey, int64(params[i].items)))
			require.Contains(t, span.Attributes(), attribute.Int64(erroredLogRecordsKey, 2))
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected err param: %v", params[i].err)
		}
	}

	checkScraperLogs(t, tt, receiverID, scraperID, int64(scrapedLogRecords), int64(erroredLogRecords))
}

func TestCheckScraperLogs(t *testing.T) {
	tt := metadatatest.SetupTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	sf, err := newObsLogs(func(context.Context) (plog.Logs, error) {
		return testdata.GenerateLogs(7), nil
	}, receiverID, scraperID, tt.NewTelemetrySettings())
	require.NoError(t, err)
	_, err = sf.ScrapeLogs(context.Background())
	require.NoError(t, err)

	checkScraperLogs(t, tt, receiverID, scraperID, 7, 0)
}

func checkScraperLogs(t *testing.T, tt metadatatest.Telemetry, receiver component.ID, scraper component.ID, scrapedLogRecords int64, erroredLogRecords int64) {
	tt.AssertMetrics(t, []metricdata.Metrics{
		{
			Name:        "otelcol_scraper_scraped_log_records",
			Description: "Number of log records successfully scraped. [alpha]",
			Unit:        "{datapoints}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Attributes: attribute.NewSet(
							attribute.String(receiverKey, receiver.String()),
							attribute.String(scraperKey, scraper.String())),
						Value: scrapedLogRecords,
					},
				},
			},
		},
		{
			Name:        "otelcol_scraper_errored_log_records",
			Description: "Number of log records that were unable to be scraped. [alpha]",
			Unit:        "{datapoints}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Attributes: attribute.NewSet(
							attribute.String(receiverKey, receiver.String()),
							attribute.String(scraperKey, scraper.String())),
						Value: erroredLogRecords,
					},
				},
			},
		},
	}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}
