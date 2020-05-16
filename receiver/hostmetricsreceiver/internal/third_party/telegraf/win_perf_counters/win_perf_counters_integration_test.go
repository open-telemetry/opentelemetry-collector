// +build windows

package win_perf_counters

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWinPerformanceQueryImpl(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	var query PerformanceQuery
	var hCounter PDH_HCOUNTER
	var err error
	query = &PerformanceQueryImpl{}

	err = query.Close()
	require.Error(t, err, "uninitialized query must return errors")

	_, err = query.AddCounterToQuery("")
	require.Error(t, err, "uninitialized query must return errors")
	assert.True(t, strings.Contains(err.Error(), "uninitialised"))

	_, err = query.AddEnglishCounterToQuery("")
	require.Error(t, err, "uninitialized query must return errors")
	assert.True(t, strings.Contains(err.Error(), "uninitialised"))

	err = query.CollectData()
	require.Error(t, err, "uninitialized query must return errors")
	assert.True(t, strings.Contains(err.Error(), "uninitialised"))

	err = query.Open()
	require.NoError(t, err)

	counterPath := "\\Processor Information(_Total)\\% Processor Time"

	hCounter, err = query.AddCounterToQuery(counterPath)
	require.NoError(t, err)
	assert.NotEqual(t, 0, hCounter)

	err = query.Close()
	require.NoError(t, err)

	err = query.Open()
	require.NoError(t, err)

	hCounter, err = query.AddEnglishCounterToQuery(counterPath)
	require.NoError(t, err)
	assert.NotEqual(t, 0, hCounter)

	cp, err := query.GetCounterPath(hCounter)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(cp, counterPath))

	err = query.CollectData()
	require.NoError(t, err)
	time.Sleep(time.Second)

	err = query.CollectData()
	require.NoError(t, err)

	_, err = query.GetFormattedCounterValueDouble(hCounter)
	require.NoError(t, err)

	now := time.Now()
	mtime, err := query.CollectDataWithTime()
	require.NoError(t, err)
	assert.True(t, mtime.Sub(now) < time.Second)

	counterPath = "\\Process(*)\\% Processor Time"
	paths, err := query.ExpandWildCardPath(counterPath)
	require.NoError(t, err)
	require.NotNil(t, paths)
	assert.True(t, len(paths) > 1)

	counterPath = "\\Process(_Total)\\*"
	paths, err = query.ExpandWildCardPath(counterPath)
	require.NoError(t, err)
	require.NotNil(t, paths)
	assert.True(t, len(paths) > 1)

	err = query.Open()
	require.NoError(t, err)

	counterPath = "\\Process(*)\\% Processor Time"
	hCounter, err = query.AddEnglishCounterToQuery(counterPath)
	require.NoError(t, err)
	assert.NotEqual(t, 0, hCounter)

	err = query.CollectData()
	require.NoError(t, err)
	time.Sleep(time.Second)

	err = query.CollectData()
	require.NoError(t, err)

	arr, err := query.GetFormattedCounterArrayDouble(hCounter)
	if phderr, ok := err.(*PdhError); ok && phderr.ErrorCode != PDH_INVALID_DATA && phderr.ErrorCode != PDH_CALC_NEGATIVE_VALUE {
		time.Sleep(time.Second)
		arr, err = query.GetFormattedCounterArrayDouble(hCounter)
	}
	require.NoError(t, err)
	assert.True(t, len(arr) > 0, "Too")

	err = query.Close()
	require.NoError(t, err)

}
