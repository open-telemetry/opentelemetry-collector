// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package xscraperhelper provides utilities for scrapers.
package xscraperhelper // import "go.opentelemetry.io/collector/scraper/scraperhelper/xscraperhelper"

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
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper/scrapererror"
	"go.opentelemetry.io/collector/scraper/scraperhelper/internal/controller"
	"go.opentelemetry.io/collector/scraper/scraperhelper/xscraperhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/scraper/xscraper"
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

func TestScrapeProfilesDataOp(t *testing.T) {
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
		sm, err := xscraper.NewProfiles(func(context.Context) (pprofile.Profiles, error) {
			return testdata.GenerateProfiles(params[i].items), params[i].err
		})
		require.NoError(t, err)
		sf, err := wrapObsProfiles(sm, receiverID, scraperID, set)
		require.NoError(t, err)
		_, err = sf.ScrapeProfiles(parentCtx)
		require.ErrorIs(t, err, params[i].err)
	}

	spans := tel.SpanRecorder.Ended()
	require.Len(t, spans, len(params))

	var scrapedProfileRecords, erroredProfileRecords int
	for i, span := range spans {
		assert.Equal(t, "scraper/"+scraperID.String()+"/ScrapeProfiles", span.Name())
		switch {
		case params[i].err == nil:
			scrapedProfileRecords += params[i].items
			require.Contains(t, span.Attributes(), attribute.Int64(scrapedProfileRecordsKey, int64(params[i].items)))
			require.Contains(t, span.Attributes(), attribute.Int64(erroredProfileRecordsKey, 0))
			assert.Equal(t, codes.Unset, span.Status().Code)
		case errors.Is(params[i].err, errFake):
			// Since we get an error, we cannot record any metrics because we don't know if the returned pprofile.Profiles is valid instance.
			require.Contains(t, span.Attributes(), attribute.Int64(scrapedProfileRecordsKey, 0))
			require.Contains(t, span.Attributes(), attribute.Int64(erroredProfileRecordsKey, 0))
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		case errors.Is(params[i].err, partialErrFake):
			scrapedProfileRecords += params[i].items
			erroredProfileRecords += 2
			require.Contains(t, span.Attributes(), attribute.Int64(scrapedProfileRecordsKey, int64(params[i].items)))
			require.Contains(t, span.Attributes(), attribute.Int64(erroredProfileRecordsKey, 2))
			assert.Equal(t, codes.Error, span.Status().Code)
			assert.Equal(t, params[i].err.Error(), span.Status().Description)
		default:
			t.Fatalf("unexpected err param: %v", params[i].err)
		}
	}

	checkScraperProfiles(t, tel, receiverID, scraperID, int64(scrapedProfileRecords), int64(erroredProfileRecords))
}

func TestCheckScraperProfiles(t *testing.T) {
	tel := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tel.Shutdown(context.Background())) })

	sm, err := xscraper.NewProfiles(func(context.Context) (pprofile.Profiles, error) {
		return testdata.GenerateProfiles(7), nil
	})
	require.NoError(t, err)
	sf, err := wrapObsProfiles(sm, receiverID, scraperID, tel.NewTelemetrySettings())
	require.NoError(t, err)
	_, err = sf.ScrapeProfiles(context.Background())
	require.NoError(t, err)

	checkScraperProfiles(t, tel, receiverID, scraperID, 7, 0)
}

func TestScrapeProfilesDataOp_LogsScraperID(t *testing.T) {
	tel := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tel.Shutdown(context.Background())) })

	core, observedLogs := observer.New(zap.ErrorLevel)
	telset := tel.NewTelemetrySettings()
	telset.Logger = zap.New(core)

	rSet := receiver.Settings{
		ID:                receiverID,
		TelemetrySettings: telset,
	}
	set := controller.GetSettings(scraperID.Type(), rSet)

	sm, err := xscraper.NewProfiles(func(context.Context) (pprofile.Profiles, error) {
		return pprofile.NewProfiles(), errFake
	})
	require.NoError(t, err)
	sf, err := wrapObsProfiles(sm, receiverID, scraperID, set.TelemetrySettings)
	require.NoError(t, err)
	_, err = sf.ScrapeProfiles(context.Background())
	require.ErrorIs(t, err, errFake)

	errorLogs := observedLogs.FilterLevelExact(zap.ErrorLevel).All()
	require.Len(t, errorLogs, 1)
	assert.Equal(t, "Error scraping profiles", errorLogs[0].Message)
	assert.Equal(t, scraperID.String(), errorLogs[0].ContextMap()["scraper"])
	assert.Equal(t, errFake.Error(), errorLogs[0].ContextMap()["error"])
}

func checkScraperProfiles(t *testing.T, tel *componenttest.Telemetry, receiver, scraper component.ID, scrapedProfileRecords, erroredProfileRecords int64) {
	metadatatest.AssertEqualScraperScrapedProfileRecords(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(receiverKey, receiver.String()),
					attribute.String(scraperKey, scraper.String())),
				Value: scrapedProfileRecords,
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	metadatatest.AssertEqualScraperErroredProfileRecords(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(
					attribute.String(receiverKey, receiver.String()),
					attribute.String(scraperKey, scraper.String())),
				Value: erroredProfileRecords,
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
}
