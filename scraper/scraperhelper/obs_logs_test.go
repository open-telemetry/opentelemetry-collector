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
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper/internal/metadatatest"
)

func TestScrapeLogsDataOp(t *testing.T) {
	tel := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tel.Shutdown(context.Background())) })

	set := tel.NewTelemetrySettings()

	parentCtx, parentSpan := set.TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	params := []testParams{
		{items: 23, err: partialErrFake},
		{items: 29, err: errFake},
		{items: 15, err: nil},
	}
	for i := range params {
		sm, err := scraper.NewLogs(func(context.Context) (plog.Logs, error) {
			return testdata.GenerateLogs(params[i].items), params[i].err
		})
		require.NoError(t, err)
		sf, err := wrapObsLogs(sm, receiverID, scraperID, set)
		require.NoError(t, err)
		_, err = sf.ScrapeLogs(parentCtx)
		require.ErrorIs(t, err, params[i].err)
	}

	spans := tel.SpanRecorder.Ended()
	require.Len(t, spans, len(params))

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

	checkScraperLogs(t, tel, receiverID, scraperID, int64(scrapedLogRecords), int64(erroredLogRecords))
}

func TestCheckScraperLogs(t *testing.T) {
	tel := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tel.Shutdown(context.Background())) })

	sm, err := scraper.NewLogs(func(context.Context) (plog.Logs, error) {
		return testdata.GenerateLogs(7), nil
	})
	require.NoError(t, err)
	sf, err := wrapObsLogs(sm, receiverID, scraperID, tel.NewTelemetrySettings())
	require.NoError(t, err)
	_, err = sf.ScrapeLogs(context.Background())
	require.NoError(t, err)

	checkScraperLogs(t, tel, receiverID, scraperID, 7, 0)
}

func checkScraperLogs(t *testing.T, tel *componenttest.Telemetry, receiver, scraper component.ID, scrapedLogRecords, erroredLogRecords int64) {
	metadatatest.AssertEqualScraperScrapedLogRecords(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(receiverKey, receiver.String()),
					attribute.String(scraperKey, scraper.String())),
				Value: scrapedLogRecords,
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	metadatatest.AssertEqualScraperErroredLogRecords(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(receiverKey, receiver.String()),
					attribute.String(scraperKey, scraper.String())),
				Value: erroredLogRecords,
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}
