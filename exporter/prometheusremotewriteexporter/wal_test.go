// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheusremotewriteexporter

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func doNothingExportSink(_ context.Context, reqL []*prompb.WriteRequest) []error {
	_ = reqL
	return nil
}

func TestWALCreation_nilConfig(t *testing.T) {
	config := (*walConfig)(nil)
	pwal, err := newWAL(config, doNothingExportSink)
	require.Equal(t, err, errNilConfig)
	require.Nil(t, pwal)
}

func TestWALCreation_nonNilConfig(t *testing.T) {
	config := &walConfig{Directory: t.TempDir()}
	pwal, err := newWAL(config, doNothingExportSink)
	require.NotNil(t, pwal)
	assert.Nil(t, err)
	pwal.stop()
}

func orderByLabelValueForEach(reqL []*prompb.WriteRequest) {
	for _, req := range reqL {
		orderByLabelValue(req)
	}
}

func orderByLabelValue(wreq *prompb.WriteRequest) {
	// Sort the timeSeries by their labels.
	type byLabelMessage struct {
		label  *prompb.Label
		sample *prompb.Sample
	}

	for _, timeSeries := range wreq.Timeseries {
		bMsgs := make([]*byLabelMessage, 0, len(wreq.Timeseries)*10)
		for i := range timeSeries.Labels {
			bMsgs = append(bMsgs, &byLabelMessage{
				label:  &timeSeries.Labels[i],
				sample: &timeSeries.Samples[i],
			})
		}
		sort.Slice(bMsgs, func(i, j int) bool {
			return bMsgs[i].label.Value < bMsgs[j].label.Value
		})

		for i := range bMsgs {
			timeSeries.Labels[i] = *bMsgs[i].label
			timeSeries.Samples[i] = *bMsgs[i].sample
		}
	}
}

func TestWALStopManyTimes(t *testing.T) {
	tempDir := t.TempDir()
	config := &walConfig{
		Directory:         tempDir,
		TruncateFrequency: 60 * time.Microsecond,
		NBeforeTruncation: 1,
	}
	pwal, err := newWAL(config, doNothingExportSink)
	require.Nil(t, err)
	require.NotNil(t, pwal)

	// Ensure that invoking .stop() multiple times doesn't cause a panic, but actually
	// First close should NOT return an error.
	err = pwal.stop()
	require.Nil(t, err)
	for i := 0; i < 4; i++ {
		// Every invocation to .stop() should return an errAlreadyClosed.
		err = pwal.stop()
		require.Equal(t, err, errAlreadyClosed)
	}
}

func TestWAL_persist(t *testing.T) {
	// Unit tests that requests written to the WAL persist.
	config := &walConfig{Directory: t.TempDir()}

	pwal, err := newWAL(config, doNothingExportSink)
	require.Nil(t, err)

	// 1. Write out all the entries.
	reqL := []*prompb.WriteRequest{
		{
			Timeseries: []prompb.TimeSeries{
				{
					Labels:  []prompb.Label{{Name: "ts1l1", Value: "ts1k1"}},
					Samples: []prompb.Sample{{Value: 1, Timestamp: 100}},
				},
			},
		},
		{
			Timeseries: []prompb.TimeSeries{
				{
					Labels:  []prompb.Label{{Name: "ts2l1", Value: "ts2k1"}},
					Samples: []prompb.Sample{{Value: 2, Timestamp: 200}},
				},
				{
					Labels:  []prompb.Label{{Name: "ts1l1", Value: "ts1k1"}},
					Samples: []prompb.Sample{{Value: 1, Timestamp: 100}},
				},
			},
		},
	}

	ctx := context.Background()
	err = pwal.start(ctx)
	require.Nil(t, err)
	defer pwal.stop()

	err = pwal.persistToWAL(reqL)
	require.Nil(t, err)

	// 2. Read all the entries from the WAL itself, guided by the indices available,
	// and ensure that they are exactly in order as we'd expect them.
	wal := pwal.wal
	start, err := wal.FirstIndex()
	require.Nil(t, err)
	end, err := wal.LastIndex()
	require.Nil(t, err)

	var reqLFromWAL []*prompb.WriteRequest
	for i := start; i <= end; i++ {
		req, err := pwal.readPrompbFromWAL(ctx, i)
		require.Nil(t, err)
		reqLFromWAL = append(reqLFromWAL, req)
	}

	orderByLabelValueForEach(reqL)
	orderByLabelValueForEach(reqLFromWAL)
	require.Equal(t, reqLFromWAL[0], reqL[0])
	require.Equal(t, reqLFromWAL[1], reqL[1])
}
