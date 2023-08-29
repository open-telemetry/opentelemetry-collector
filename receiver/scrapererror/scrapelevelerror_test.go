package scrapererror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestScrapeLevelErrorsAddAndCombine(t *testing.T) {
	errs := &ScrapeLevelErrors{}

	errs.Add(errors.New("it messed up"))
	errs.Add(errors.New("it messed up twice"))

	err := errs.CombineErrors()

	assert.False(t, IsPartialScrapeError(err))
}

func TestScrapeLevelErrorsAddWarningAndCombine(t *testing.T) {
	errs := &ScrapeLevelErrors{}

	errs.AddWarning(errors.New("it messed up"))
	errs.AddWarning(errors.New("it messed up twice"))

	err := errs.CombineErrors()

	assert.Nil(t, err)
}

func TestScrapeLevelErrorsAddPartialAndCombine(t *testing.T) {
	errs := &ScrapeLevelErrors{}

	errs.AddPartial(1, errors.New("it messed up"))
	errs.AddPartial(2, errors.New("it messed up twice"))

	err := errs.CombineErrors()

	assert.True(t, IsPartialScrapeError(err))
	var partialErr PartialScrapeError
	assert.True(t, errors.As(err, &partialErr))
	assert.Equal(t, partialErr.Failed, 3)
}

func TestScrapeLevelErrorsLogAll(t *testing.T) {
	testCases := []struct {
		name     string
		errs     []error
		warnings []error
		debugs   []error
	}{
		{
			name:     "one of each",
			errs:     []error{errors.New("failed bad")},
			warnings: []error{errors.New("failed not that bad")},
			debugs:   []error{errors.New("just in case you want to see it")},
		},
		{
			name:     "just warnings and debugs",
			warnings: []error{errors.New("failed not that bad")},
			debugs:   []error{errors.New("just in case you want to see it")},
		},
		{
			name:     "multiple of each",
			errs:     []error{errors.New("failed bad"), errors.New("so so bad")},
			warnings: []error{errors.New("failed not that bad"), errors.New("this too")},
			debugs:   []error{errors.New("just in case you want to see it"), errors.New("do you want to see this")},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := &ScrapeLevelErrors{}
			for _, err := range tc.errs {
				errs.Add(err)
			}
			for _, err := range tc.warnings {
				errs.AddWarning(err)
			}
			for _, err := range tc.debugs {
				errs.AddDebug(err)
			}

			observedCore, observedLogs := observer.New(zapcore.DebugLevel)
			logger := zap.New(observedCore)

			errs.ZapLogAll(logger, "testreceiver")

			expectedLogs := 0
			if len(tc.errs) > 0 {
				expectedLogs++
			}
			if len(tc.warnings) > 0 {
				expectedLogs++
			}
			if len(tc.debugs) > 0 {
				expectedLogs++
			}
			assert.Equal(t, observedLogs.Len(), expectedLogs)

			logEntries := observedLogs.All()
			currLog := 0
			if len(tc.errs) > 0 {
				assertLogContainsErrors(t, tc.errs, logEntries[currLog], "error")
				currLog++
			}
			if len(tc.warnings) > 0 {
				assertLogContainsErrors(t, tc.warnings, logEntries[currLog], "warning")
				currLog++
			}
			if len(tc.debugs) > 0 {
				assertLogContainsErrors(t, tc.debugs, logEntries[currLog], "debug")
				currLog++
			}
		})
	}
}

func assertLogContainsErrors(t *testing.T, errs []error, entry observer.LoggedEntry, field string) {
	t.Helper()

	err, ok := entry.ContextMap()[field]
	assert.True(t, ok, "expected log entry to contain field %s", field)
	for _, e := range errs {
		assert.Contains(t, err, e.Error())
	}
}
