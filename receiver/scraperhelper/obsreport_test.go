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
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

var (
	receiverID = component.MustNewID("fakeReceiver")
	scraperID  = component.MustNewID("fakeScraper")

	errFake        = errors.New("errFake")
	partialErrFake = scrapererror.NewPartialScrapeError(errFake, 1)
)

type testParams struct {
	items int
	err   error
}

func TestScrapeMetricsDataOp(t *testing.T) {
	testTelemetry(t, receiverID, func(t *testing.T, tt componenttest.TestTelemetry) {
		parentCtx, parentSpan := tt.TelemetrySettings().TracerProvider.Tracer("test").Start(context.Background(), t.Name())
		defer parentSpan.End()

		params := []testParams{
			{items: 23, err: partialErrFake},
			{items: 29, err: errFake},
			{items: 15, err: nil},
		}
		for i := range params {
			scrp, err := newScraper(ObsReportSettings{
				ReceiverID:             receiverID,
				Scraper:                scraperID,
				ReceiverCreateSettings: receiver.CreateSettings{ID: receiverID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
			})
			require.NoError(t, err)
			ctx := scrp.StartMetricsOp(parentCtx)
			assert.NotNil(t, ctx)
			scrp.EndMetricsOp(ctx, params[i].items, params[i].err)
		}

		spans := tt.SpanRecorder.Ended()
		require.Equal(t, len(params), len(spans))

		var scrapedMetricPoints, erroredMetricPoints int
		for i, span := range spans {
			assert.Equal(t, "scraper/"+receiverID.String()+"/"+scraperID.String()+"/MetricsScraped", span.Name())
			switch {
			case params[i].err == nil:
				scrapedMetricPoints += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.ScrapedMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.ErroredMetricPointsKey, Value: attribute.Int64Value(0)})
				assert.Equal(t, codes.Unset, span.Status().Code)
			case errors.Is(params[i].err, errFake):
				erroredMetricPoints += params[i].items
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.ScrapedMetricPointsKey, Value: attribute.Int64Value(0)})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.ErroredMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Equal(t, params[i].err.Error(), span.Status().Description)

			case errors.Is(params[i].err, partialErrFake):
				scrapedMetricPoints += params[i].items
				erroredMetricPoints++
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.ScrapedMetricPointsKey, Value: attribute.Int64Value(int64(params[i].items))})
				require.Contains(t, span.Attributes(), attribute.KeyValue{Key: obsmetrics.ErroredMetricPointsKey, Value: attribute.Int64Value(1)})
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Equal(t, params[i].err.Error(), span.Status().Description)
			default:
				t.Fatalf("unexpected err param: %v", params[i].err)
			}
		}

		require.NoError(t, tt.CheckScraperMetrics(receiverID, scraperID, int64(scrapedMetricPoints), int64(erroredMetricPoints)))
	})
}

func TestCheckScraperMetricsViews(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(receiverID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	s, err := NewObsReport(ObsReportSettings{
		ReceiverID:             receiverID,
		Scraper:                scraperID,
		ReceiverCreateSettings: receiver.CreateSettings{ID: receiverID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := s.StartMetricsOp(context.Background())
	require.NotNil(t, ctx)
	s.EndMetricsOp(ctx, 7, nil)

	assert.NoError(t, tt.CheckScraperMetrics(receiverID, scraperID, 7, 0))
	assert.Error(t, tt.CheckScraperMetrics(receiverID, scraperID, 7, 7))
	assert.Error(t, tt.CheckScraperMetrics(receiverID, scraperID, 0, 0))
	assert.Error(t, tt.CheckScraperMetrics(receiverID, scraperID, 0, 7))
}

func testTelemetry(t *testing.T, id component.ID, testFunc func(t *testing.T, tt componenttest.TestTelemetry)) {
	tt, err := componenttest.SetupTelemetry(id)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	testFunc(t, tt)
}
