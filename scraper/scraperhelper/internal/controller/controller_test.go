package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
)

var (
	receiverID = component.MustNewID("fakeReceiver")
	scraperID  = component.MustNewID("fakeScraper")
)

func TestGetSettings_DoesNotMutateReceiverSettings(t *testing.T) {
	core, observedLogs := observer.New(zap.DebugLevel)
	originalLogger := zap.New(core)
	rSet := receiver.Settings{
		ID: receiverID,
		TelemetrySettings: component.TelemetrySettings{
			Logger: originalLogger,
		},
	}
	scraperSettings := GetSettings(scraperID.Type(), rSet)
	rSet.Logger.Error("test log from receiver")
	scraperSettings.Logger.Error("test log from scraper")

	allLogs := observedLogs.All()
	require.Len(t, allLogs, 2)

	receiverLog := allLogs[0]
	assert.Equal(t, "test log from receiver", receiverLog.Message)
	assert.NotContains(t, receiverLog.ContextMap(), "scraper")

	scraperLog := allLogs[1]
	assert.Equal(t, "test log from scraper", scraperLog.Message)
	assert.Equal(t, scraperID.String(), scraperLog.ContextMap()["scraper"])
}
