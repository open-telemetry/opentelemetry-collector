// Copyright 2020 The OpenTelemetry Authors
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

package cortexexporter

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	"go.opentelemetry.io/collector/internal/data/testdata"

	proto "github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	// "github.com/stretchr/testify/require"
)

// TODO: try to run Test_PushMetrics after export() is in
// TODO: add bucket and histogram test cases for Test_PushMetrics
// TODO: check that NoError instead of NoNil is used at the right places
// TODO: add one line comment before each test stating criteria

// Test_handleScalarMetric checks whether data points within a single scalar metric can be added to a map of
// TimeSeries correctly.
// Test cases are two data point belonging to the same TimeSeries, two data point belonging different TimeSeries,
// and nil data points case.
func Test_handleScalarMetric(t *testing.T) {
	sameTs := map[string]*prompb.TimeSeries{
		// string signature of the data point is the key of the map
		typeMonotonicInt64 + "-name-same_ts_int_points_total" + lb1Sig: getTimeSeries(
			getPromLabels(label11, value11, label12, value12, "name", "same_ts_int_points_total"),
			getSample(float64(intVal1), time1),
			getSample(float64(intVal2), time1)),
	}
	differentTs := map[string]*prompb.TimeSeries{
		typeMonotonicInt64 + "-name-different_ts_int_points_total" + lb1Sig: getTimeSeries(
			getPromLabels(label11, value11, label12, value12, "name", "different_ts_int_points_total"),
			getSample(float64(intVal1), time1)),
		typeMonotonicInt64 + "-name-different_ts_int_points_total" + lb2Sig: getTimeSeries(
			getPromLabels(label21, value21, label22, value22, "name", "different_ts_int_points_total"),
			getSample(float64(intVal1), time2)),
	}

	tests := []struct {
		name        string
		m           *otlp.Metric
		returnError bool
		want        map[string]*prompb.TimeSeries
	}{
		{
			"invalid_nil_array",
			&otlp.Metric{
				MetricDescriptor:    getDescriptor("invalid_nil_array", monotonicInt64Comb, validCombinations),
				Int64DataPoints:     nil,
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			true,
			map[string]*prompb.TimeSeries{},
		},
		{
			"same_ts_int_points",
			&otlp.Metric{
				MetricDescriptor: getDescriptor("same_ts_int_points", monotonicInt64Comb, validCombinations),
				Int64DataPoints: []*otlp.Int64DataPoint{
					getIntDataPoint(lbs1, intVal1, time1),
					getIntDataPoint(lbs1, intVal2, time1),
				},
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			false,
			sameTs,
		},
		{
			"different_ts_int_points",
			&otlp.Metric{
				MetricDescriptor: getDescriptor("different_ts_int_points", monotonicInt64Comb, validCombinations),
				Int64DataPoints: []*otlp.Int64DataPoint{
					getIntDataPoint(lbs1, intVal1, time1),
					getIntDataPoint(lbs2, intVal1, time2),
				},
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			false,
			differentTs,
		},
	}
	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tsMap := map[string]*prompb.TimeSeries{}
			ce := &cortexExporter{}
			ok := ce.handleScalarMetric(tsMap, tt.m)
			if tt.returnError {
				assert.Error(t, ok)
				return
			}
			assert.Exactly(t, len(tt.want), len(tsMap))
			for k, v := range tsMap {
				require.NotNil(t, tt.want[k])
				assert.ElementsMatch(t, tt.want[k].Labels, v.Labels)
				assert.ElementsMatch(t, tt.want[k].Samples, v.Samples)
			}
		})
	}
}

// Test_handleHistogramMetric checks whether data points(sum, count, buckets) within a single Histogram metric can be
// added to a map of TimeSeries correctly.
// Test cases are a histogram data point with two buckets and nil data points case.
func Test_handleHistogramMetric(t *testing.T) {
	sum := "sum"
	count := "count"
	bucket1 := "bucket1"
	bucket2 := "bucket2"
	bucketInf := "bucketInf"
	histPoint := otlp.HistogramDataPoint{
		Labels:            lbs1,
		StartTimeUnixNano: 0,
		TimeUnixNano:      time1,
		Count:             uint64(intVal2),
		Sum:               floatVal2,
		Buckets: []*otlp.HistogramDataPoint_Bucket{
			{uint64(intVal1),
				nil,
			},
			{uint64(intVal1),
				nil,
			},
		},
		ExplicitBounds: []float64{
			floatVal1,
			floatVal2,
		},
	}
	// string signature of the data point is the key of the map
	sigs := map[string]string{
		sum:   typeHistogram + "-name-" + name1 + "_sum" + lb1Sig,
		count: typeHistogram + "-name-" + name1 + "_count" + lb1Sig,
		bucket1: typeHistogram + "-" + "le-" + strconv.FormatFloat(floatVal1, 'f', -1, 64) +
			"-name-" + name1 + "_bucket" + lb1Sig,
		bucket2: typeHistogram + "-" + "le-" + strconv.FormatFloat(floatVal2, 'f', -1, 64) +
			"-name-" + name1 + "_bucket" + lb1Sig,
		bucketInf: typeHistogram + "-" + "le-" + "+Inf" +
			"-name-" + name1 + "_bucket" + lb1Sig,
	}
	lbls := map[string][]prompb.Label{
		sum:   append(promLbs1, getPromLabels("name", name1+"_sum")...),
		count: append(promLbs1, getPromLabels("name", name1+"_count")...),
		bucket1: append(promLbs1, getPromLabels("name", name1+"_bucket", "le",
			strconv.FormatFloat(floatVal1, 'f', -1, 64))...),
		bucket2: append(promLbs1, getPromLabels("name", name1+"_bucket", "le",
			strconv.FormatFloat(floatVal2, 'f', -1, 64))...),
		bucketInf: append(promLbs1, getPromLabels("name", name1+"_bucket", "le",
			"+Inf")...),
	}
	tests := []struct {
		name        string
		m           otlp.Metric
		returnError bool
		want        map[string]*prompb.TimeSeries
	}{
		{
			"invalid_nil_array",
			otlp.Metric{
				MetricDescriptor:    getDescriptor("invalid_nil_array", histogramComb, validCombinations),
				Int64DataPoints:     nil,
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			true,
			map[string]*prompb.TimeSeries{},
		},
		{
			"single_histogram_point",
			otlp.Metric{
				MetricDescriptor:    getDescriptor(name1+"", histogramComb, validCombinations),
				Int64DataPoints:     nil,
				DoubleDataPoints:    nil,
				HistogramDataPoints: []*otlp.HistogramDataPoint{&histPoint},
				SummaryDataPoints:   nil,
			},
			false,
			map[string]*prompb.TimeSeries{
				sigs[sum]:       getTimeSeries(lbls[sum], getSample(floatVal2, time1)),
				sigs[count]:     getTimeSeries(lbls[count], getSample(float64(intVal2), time1)),
				sigs[bucket1]:   getTimeSeries(lbls[bucket1], getSample(float64(intVal1), time1)),
				sigs[bucket2]:   getTimeSeries(lbls[bucket2], getSample(float64(intVal1), time1)),
				sigs[bucketInf]: getTimeSeries(lbls[bucketInf], getSample(float64(intVal2), time1)),
			},
		},
	}

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tsMap := map[string]*prompb.TimeSeries{}
			ce := &cortexExporter{}
			ok := ce.handleHistogramMetric(tsMap, &tt.m)
			if tt.returnError {
				assert.Error(t, ok)
				return
			}
			assert.Exactly(t, len(tt.want), len(tsMap))
			for k, v := range tsMap {
				require.NotNil(t, tt.want[k], k)
				assert.ElementsMatch(t, tt.want[k].Labels, v.Labels)
				assert.ElementsMatch(t, tt.want[k].Samples, v.Samples)
			}
		})
	}
}

// Test_handleSummaryMetric checks whether data points(sum, count, quantiles) within a single Summary metric can be
// added to a map of TimeSeries correctly.
// Test cases are a summary data point with two quantiles and nil data points case.
func Test_handleSummaryMetric(t *testing.T) {
	sum := "sum"
	count := "count"
	q1 := "quantile1"
	q2 := "quantile2"
	// string signature is the key of the map
	sigs := map[string]string{
		sum:   typeSummary + "-name-" + name1 + "_sum" + lb1Sig,
		count: typeSummary + "-name-" + name1 + "_count" + lb1Sig,
		q1: typeSummary + "-name-" + name1 + "-" + "quantile-" +
			strconv.FormatFloat(floatVal1, 'f', -1, 64) + lb1Sig,
		q2: typeSummary + "-name-" + name1 + "-" + "quantile-" +
			strconv.FormatFloat(floatVal2, 'f', -1, 64) + lb1Sig,
	}
	lbls := map[string][]prompb.Label{
		sum:   append(promLbs1, getPromLabels("name", name1+"_sum")...),
		count: append(promLbs1, getPromLabels("name", name1+"_count")...),
		q1: append(promLbs1, getPromLabels("name", name1, "quantile",
			strconv.FormatFloat(floatVal1, 'f', -1, 64))...),
		q2: append(promLbs1, getPromLabels("name", name1, "quantile",
			strconv.FormatFloat(floatVal2, 'f', -1, 64))...),
	}
	summaryPoint := otlp.SummaryDataPoint{
		Labels:            lbs1,
		StartTimeUnixNano: 0,
		TimeUnixNano:      uint64(time1),
		Count:             uint64(intVal2),
		Sum:               floatVal2,
		PercentileValues: []*otlp.SummaryDataPoint_ValueAtPercentile{
			{floatVal1,
				floatVal1,
			},
			{floatVal2,
				floatVal1,
			},
		},
	}
	tests := []struct {
		name        string
		m           otlp.Metric
		returnError bool
		want        map[string]*prompb.TimeSeries
	}{
		{
			"invalid_nil_array",
			otlp.Metric{
				MetricDescriptor:    getDescriptor("invalid_nil_array", summaryComb, validCombinations),
				Int64DataPoints:     nil,
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			true,
			map[string]*prompb.TimeSeries{},
		},
		{
			"single_summary_point",
			otlp.Metric{
				MetricDescriptor:    getDescriptor(name1, summaryComb, validCombinations),
				Int64DataPoints:     nil,
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   []*otlp.SummaryDataPoint{&summaryPoint},
			},
			false,
			map[string]*prompb.TimeSeries{
				sigs[sum]:   getTimeSeries(lbls[sum], getSample(floatVal2, time1)),
				sigs[count]: getTimeSeries(lbls[count], getSample(float64(intVal2), time1)),
				sigs[q1]:    getTimeSeries(lbls[q1], getSample(float64(intVal1), time1)),
				sigs[q2]:    getTimeSeries(lbls[q2], getSample(float64(intVal1), time1)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tsMap := map[string]*prompb.TimeSeries{}
			ce := &cortexExporter{}
			ok := ce.handleSummaryMetric(tsMap, &tt.m)
			if tt.returnError {
				assert.Error(t, ok)
				return
			}
			assert.Exactly(t, len(tt.want), len(tsMap))
			for k, v := range tsMap {
				require.NotNil(t, tt.want[k], k)
				assert.ElementsMatch(t, tt.want[k].Labels, v.Labels)
				assert.ElementsMatch(t, tt.want[k].Samples, v.Samples)
			}
		})
	}
}

// Test_newCortexExporter checks that a new exporter instance with non-nil fields is initialized
func Test_newCortexExporter(t *testing.T) {
	config := &Config{
		ExporterSettings:   configmodels.ExporterSettings{},
		TimeoutSettings:    exporterhelper.TimeoutSettings{},
		QueueSettings:      exporterhelper.QueueSettings{},
		RetrySettings:      exporterhelper.RetrySettings{},
		Namespace:          "",
		HTTPClientSettings: confighttp.HTTPClientSettings{Endpoint: ""},
	}
	c, _ := config.HTTPClientSettings.ToClient()
	ce, err := newCortexExporter(config.HTTPClientSettings.Endpoint, config.Namespace, c)
	require.NoError(t, err)
	require.NotNil(t, ce)
	assert.NotNil(t, ce.namespace)
	assert.NotNil(t, ce.endpoint)
	assert.NotNil(t, ce.client)
	assert.NotNil(t, ce.closeChan)
	assert.NotNil(t, ce.wg)
}

// Test_shutdown checks after shutdown is called, incoming calls to pushMetrics return error.
func Test_shutdown(t *testing.T) {
	ce := &cortexExporter{
		wg:        new(sync.WaitGroup),
		closeChan: make(chan struct{}),
	}
	wg := new(sync.WaitGroup)
	errChan := make(chan error, 5)
	err := ce.shutdown(context.Background())
	require.NoError(t, err)
	errChan = make(chan error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok := ce.pushMetrics(context.Background(),
				pdatautil.MetricsFromInternalMetrics(testdata.GenerateMetricDataEmpty()))
			errChan <- ok
		}()
	}
	wg.Wait()
	close(errChan)
	for ok := range errChan {
		assert.Error(t, ok)
	}
}

//Test whether or not the Server receives the correct TimeSeries.
//Currently considering making this test an iterative for loop of multiple TimeSeries
//Much akin to Test_pushMetrics
func Test_export(t *testing.T) {
	//First we will instantiate a dummy TimeSeries instance to pass into both the export call and compare the http request
	labels := getPromLabels(label11, value11, label12, value12, label21, value21, label22, value22)
	sample1 := getSample(floatVal1, time1)
	sample2 := getSample(floatVal2, time2)
	ts1 := getTimeSeries(labels, sample1, sample2)

	//This is supposed to represent the Cortex gateway instance, and how the Cortex Gateway receives and parses the writeRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//The following is a handler function that reads the sent httpRequest, unmarshals, and checks if the WriteRequest
		//preserves the TimeSeries data correctly
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		require.NotNil(t,body)
		//Receives the http requests and unzip, unmarshals, and extracts TimeSeries
		assert.Equal(t, "0.1.0", r.Header.Get("X-Prometheus-Remote-Write-Version"))
		assert.Equal(t, "snappy", r.Header.Get("Content-Encoding"))
		writeReq := &prompb.WriteRequest{}
		unzipped := []byte{}

		dest,err := snappy.Decode(unzipped, body)
		require.NoError(t,err)

		ok := proto.Unmarshal(dest, writeReq)
		require.NoError(t, ok)

		assert.EqualValues(t, 1, len(writeReq.Timeseries))
		require.NotNil(t, writeReq.GetTimeseries())
		assert.Equal(t, *ts1, writeReq.GetTimeseries()[0])
		w.WriteHeader(http.StatusAccepted)
	}))

	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)
	endpoint := serverURL.String()
	err = runExportPipeline(t, ts1, endpoint)
	assert.NoError(t, err)
}

func runExportPipeline(t *testing.T, ts *prompb.TimeSeries, endpoint string) error {
	//First we will construct a TimeSeries array from the testutils package
	testmap := make(map[string]*prompb.TimeSeries)
	testmap["test"] = ts

	HTTPClient := http.DefaultClient
	//after this, instantiate a CortexExporter with the current HTTP client and endpoint set to passed in endpoint
	ce, err := newCortexExporter("test", endpoint, HTTPClient)
	if err != nil {
		return err
	}
	err = ce.export(context.Background(), testmap)
	return err
}

// Bug{@huyan0} success case pass but it should fail; this is because the server gets no request because export() is
// empty. This test cannot run until export is finished.
// Test_pushMetrics the correctness and the number of points
func Test_pushMetrics(t *testing.T) {
	noTempBatch := pdatautil.MetricsFromInternalMetrics(testdata.GenerateMetricDataManyMetricsSameResource(10))
	noDescBatch := pdatautil.MetricsFromInternalMetrics(testdata.GenerateMetricDataMetricTypeInvalid())
	// 10 counter metrics, 2 points in each. Two TimeSeries in total
	batch := testdata.GenerateMetricDataManyMetricsSameResource(10)
	setCumulative(&batch)
	successBatch := pdatautil.MetricsFromInternalMetrics(batch)

	tests := []struct {
		name                 string
		md                   *pdata.Metrics
		reqTestFunc          func(t *testing.T, r *http.Request)
		httpResponseCode     int
		numDroppedTimeSeries int
		returnErr            bool
	}{
		{
			"no_desc_case",
			&noDescBatch,
			nil,
			http.StatusAccepted,
			pdatautil.MetricCount(noDescBatch),
			true,
		},
		{
			"no_temp_case",
			&noTempBatch,
			nil,
			http.StatusAccepted,
			pdatautil.MetricCount(noTempBatch),
			true,
		},
		{
			"http_error_case",
			&noTempBatch,
			nil,
			http.StatusForbidden,
			pdatautil.MetricCount(noTempBatch),
			true,
		},
		{
			"success_case",
			&successBatch,
			func(t *testing.T, r *http.Request) {
				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}

				buf := make([]byte, len(body))
				dest, err := snappy.Decode(buf, body)
				assert.Equal(t, "0.1.0", r.Header.Get("x-prometheus-remote-write-version"))
				assert.Equal(t, "snappy", r.Header.Get("content-encoding"))
				assert.NotNil(t, r.Header.Get("tenant-id"))
				require.NoError(t,err)
				wr := &prompb.WriteRequest{}
				ok  := proto.Unmarshal(dest, wr)
				require.Nil(t, ok)
				assert.EqualValues(t, 2, len(wr.Timeseries))
			},
			http.StatusAccepted,
			0,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.reqTestFunc != nil {
					tt.reqTestFunc(t, r)
				}
				w.WriteHeader(tt.httpResponseCode)
			}))

			defer server.Close()

			serverURL, uErr := url.Parse(server.URL)
			assert.NoError(t, uErr)

			config := createDefaultConfig().(*Config)
			assert.NotNil(t, config)
			// c, err := config.HTTPClientSettings.ToClient()
			// assert.Nil(t, err)
			c := http.DefaultClient
			sender, nErr := newCortexExporter(config.Namespace, serverURL.String(), c)
			require.NoError(t, nErr)
			numDroppedTimeSeries, err := sender.pushMetrics(context.Background(), *tt.md)
			assert.Equal(t, tt.numDroppedTimeSeries, numDroppedTimeSeries)
			if tt.returnErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

		})
	}
}
