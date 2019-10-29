// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"context"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/scrape"
)

func Test_transaction(t *testing.T) {
	ms := &mockMetadataSvc{
		caches: map[string]*mockMetadataCache{
			"test_localhost:8080": {data: map[string]scrape.MetricMetadata{}},
		},
	}

	rn := "prometheus"

	t.Run("Commit Without Adding", func(t *testing.T) {
		mcon := newMockConsumer()
		tr := newTransaction(context.Background(), nil, true, rn, ms, mcon, testLogger)
		if got := tr.Commit(); got != nil {
			t.Errorf("expecting nil from Commit() but got err %v", got)
		}
	})

	t.Run("Rollback dose nothing", func(t *testing.T) {
		mcon := newMockConsumer()
		tr := newTransaction(context.Background(), nil, true, rn, ms, mcon, testLogger)
		if got := tr.Rollback(); got != nil {
			t.Errorf("expecting nil from Rollback() but got err %v", got)
		}
	})

	badLabels := labels.Labels([]labels.Label{{Name: "foo", Value: "bar"}})
	t.Run("Add One No Target", func(t *testing.T) {
		mcon := newMockConsumer()
		tr := newTransaction(context.Background(), nil, true, rn, ms, mcon, testLogger)
		if _, got := tr.Add(badLabels, time.Now().Unix()*1000, 1.0); got == nil {
			t.Errorf("expecting error from Add() but got nil")
		}
	})

	jobNotFoundLb := labels.Labels([]labels.Label{
		{Name: "instance", Value: "localhost:8080"},
		{Name: "job", Value: "test2"},
		{Name: "foo", Value: "bar"}})
	t.Run("Add One Job not found", func(t *testing.T) {
		mcon := newMockConsumer()
		tr := newTransaction(context.Background(), nil, true, rn, ms, mcon, testLogger)
		if _, got := tr.Add(jobNotFoundLb, time.Now().Unix()*1000, 1.0); got == nil {
			t.Errorf("expecting error from Add() but got nil")
		}
	})

	goodLabels := labels.Labels([]labels.Label{{Name: "instance", Value: "localhost:8080"},
		{Name: "job", Value: "test"},
		{Name: "__name__", Value: "foo"}})
	t.Run("Add One Good", func(t *testing.T) {
		mcon := newMockConsumer()
		tr := newTransaction(context.Background(), nil, true, rn, ms, mcon, testLogger)
		if _, got := tr.Add(goodLabels, time.Now().Unix()*1000, 1.0); got != nil {
			t.Errorf("expecting error == nil from Add() but got: %v\n", got)
		}
		tr.metricBuilder.startTime = 1.0 // set to a non-zero value
		if got := tr.Commit(); got != nil {
			t.Errorf("expecting nil from Commit() but got err %v", got)
		}
		expected := createNode("test", "localhost:8080", "http")
		md := mcon.md
		if !reflect.DeepEqual(md.Node, expected) {
			t.Errorf("generated node %v and expected node %v is different\n", md.Node, expected)
		}

		if len(md.Metrics) != 1 {
			t.Errorf("expecting one metrics, but got %v\n", len(md.Metrics))
		}
	})

	t.Run("Drop NaN value", func(t *testing.T) {
		mcon := newMockConsumer()
		tr := newTransaction(context.Background(), nil, true, rn, ms, mcon, testLogger)
		if _, got := tr.Add(goodLabels, time.Now().Unix()*1000, math.NaN()); got != nil {
			t.Errorf("expecting error == nil from Add() but got: %v\n", got)
		}
		if got := tr.Commit(); got != nil {
			t.Errorf("expecting nil from Commit() but got err %v", got)
		}

		if mcon.md != nil {
			t.Errorf("wanted nil, got %v\n", mcon.md)
		}
	})

}
