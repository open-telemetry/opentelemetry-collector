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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/receiver/scrapererror"
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
	tt, err := componenttest.SetupTelemetry(receiverID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	parentCtx, parentSpan := tt.TelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
	defer parentSpan.End()

	params := []testParams{
		{items: 23, err: partialErrFake},
		{items: 29, err: errFake},
		{items: 15, err: nil},
	}
	for i := range params {
		var sf ScrapeFunc
		sf, err = newObsMetrics(func(context.Context) (pmetric.Metrics, error) {
			return testdata.GenerateMetrics(params[i].items), params[i].err
		}, receiverID, scraperID, tt.TelemetrySettings())
		require.NoError(t, err)
		_, err = sf.ScrapeMetrics(parentCtx)
		require.ErrorIs(t, err, params[i].err)
	}

	spans := tt.SpanRecorder.Ended()
	require.Equal(t, len(params), len(spans))

	var scrapedMetricPoints, erroredMetricPoints int
	for i, span := range spans {
		assert.Equal(t, "scraper/"+scraperID.String()+"/ScrapeMetrics", span.Name())
		switch {
		case params[i].err == nil:
			scrapedMetricPoints += params[i].items
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: scrapedMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: erroredMetricPointsKey, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			// Since we get an error, we cannot record any metrics because we don't know if the returned pmetric.Metrics is valid instance.
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: scrapedMetricPointsKey, Value: attribute.Int64Value(0)})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: erroredMetricPointsKey, Value: attribute.Int64Value(0)})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		case errors.Is(params[i].err, partialErrFake):
			scrapedMetricPoints += params[i].items
			erroredMetricPoints += 2
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: scrapedMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
			require.Contains(t, span.Attributes(), attribute.KeyValue{Key: erroredMetricPointsKey, Value: attribute.Int64Value(2)})
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected err param: %v", params[i].err)
		}
	}

	require.NoError(t, tt.CheckScraperMetrics(receiverID, scraperID, int64(scrapedMetricPoints), int64(erroredMetricPoints)))
}

func TestCheckScraperMetrics(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(receiverID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	var sf ScrapeFunc
	sf, err = newObsMetrics(func(context.Context) (pmetric.Metrics, error) {
		return testdata.GenerateMetrics(7), nil
	}, receiverID, scraperID, tt.TelemetrySettings())
	require.NoError(t, err)
	_, err = sf.ScrapeMetrics(context.Background())
	assert.NoError(t, err)

	require.NoError(t, tt.CheckScraperMetrics(receiverID, scraperID, 7, 0))
	require.Error(t, tt.CheckScraperMetrics(receiverID, scraperID, 7, 7))
	require.Error(t, tt.CheckScraperMetrics(receiverID, scraperID, 0, 0))
	require.Error(t, tt.CheckScraperMetrics(receiverID, scraperID, 0, 7))
}
