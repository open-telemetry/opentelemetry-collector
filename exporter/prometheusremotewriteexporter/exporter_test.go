// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheusremotewriteexporter

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"

	proto "github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1old"
	"go.opentelemetry.io/collector/internal/dataold"
	"go.opentelemetry.io/collector/internal/dataold/testdataold"
)

// Test_handleScalarMetric checks whether data points within a single scalar metric can be added to a map of
// TimeSeries correctly.
// Test cases are two data point belonging to the same TimeSeries, two data point belonging different TimeSeries,
// and nil data points case.
func Test_handleScalarMetric(t *testing.T) {
	sameTs := map[string]*prompb.TimeSeries{
		// string signature of the data point is the key of the map
		typeMonotonicInt64 + "-__name__-same_ts_int_points_total" + lb1Sig: getTimeSeries(
			getPromLabels(label11, value11, label12, value12, nameStr, "same_ts_int_points_total"),
			getSample(float64(intVal1), msTime1),
			getSample(float64(intVal2), msTime1)),
	}
	differentTs := map[string]*prompb.TimeSeries{
		typeMonotonicDouble + "-__name__-different_ts_double_points_total" + lb1Sig: getTimeSeries(
			getPromLabels(label11, value11, label12, value12, nameStr, "different_ts_double_points_total"),
			getSample(floatVal1, msTime1)),
		typeMonotonicDouble + "-__name__-different_ts_double_points_total" + lb2Sig: getTimeSeries(
			getPromLabels(label21, value21, label22, value22, nameStr, "different_ts_double_points_total"),
			getSample(floatVal2, msTime2)),
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
			"invalid_type_array",
			&otlp.Metric{
				MetricDescriptor:    getDescriptor("invalid_type_array", histogramComb, validCombinations),
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
			"different_ts_double_points",
			&otlp.Metric{
				MetricDescriptor: getDescriptor("different_ts_double_points", monotonicDoubleComb, validCombinations),
				Int64DataPoints:  nil,
				DoubleDataPoints: []*otlp.DoubleDataPoint{
					getDoubleDataPoint(lbs1, floatVal1, time1),
					getDoubleDataPoint(lbs2, floatVal2, time2),
				},
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
			prw := &PrwExporter{}
			ok := prw.handleScalarMetric(tsMap, tt.m)
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
	histPoint := getHistogramDataPoint(
		lbs1,
		time1,
		floatVal2,
		uint64(intVal2), []float64{floatVal1, floatVal2},
		[]uint64{uint64(intVal1), uint64(intVal1)})

	// string signature of the data point is the key of the map
	sigs := map[string]string{
		sum:   typeHistogram + "-" + nameStr + "-" + name1 + "_sum" + lb1Sig,
		count: typeHistogram + "-" + nameStr + "-" + name1 + "_count" + lb1Sig,
		bucket1: typeHistogram + "-" + nameStr + "-" + name1 + "_bucket" + "-" + "le-" +
			strconv.FormatFloat(floatVal1, 'f', -1, 64) + lb1Sig,
		bucket2: typeHistogram + "-" + nameStr + "-" + name1 + "_bucket" + "-" + "le-" +
			strconv.FormatFloat(floatVal2, 'f', -1, 64) + lb1Sig,
		bucketInf: typeHistogram + "-" + nameStr + "-" + name1 + "_bucket" + "-" + "le-" +
			"+Inf" + lb1Sig,
	}
	labels := map[string][]prompb.Label{
		sum:   append(promLbs1, getPromLabels(nameStr, name1+"_sum")...),
		count: append(promLbs1, getPromLabels(nameStr, name1+"_count")...),
		bucket1: append(promLbs1, getPromLabels(nameStr, name1+"_bucket", "le",
			strconv.FormatFloat(floatVal1, 'f', -1, 64))...),
		bucket2: append(promLbs1, getPromLabels(nameStr, name1+"_bucket", "le",
			strconv.FormatFloat(floatVal2, 'f', -1, 64))...),
		bucketInf: append(promLbs1, getPromLabels(nameStr, name1+"_bucket", "le",
			"+Inf")...),
	}
	tests := []struct {
		name        string
		m           otlp.Metric
		returnError bool
		want        map[string]*prompb.TimeSeries
	}{
		{
			"invalid_type_array",
			otlp.Metric{
				MetricDescriptor:    getDescriptor("invalid_type_array", histogramComb, validCombinations),
				Int64DataPoints:     nil,
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			true,
			map[string]*prompb.TimeSeries{},
		},
		{
			"hist_nil_pt",
			otlp.Metric{
				MetricDescriptor:    getDescriptor("hist_nil_pt", histogramComb, validCombinations),
				Int64DataPoints:     nil,
				DoubleDataPoints:    nil,
				HistogramDataPoints: []*otlp.HistogramDataPoint{nil},
				SummaryDataPoints:   nil,
			},
			false,
			map[string]*prompb.TimeSeries{},
		},
		{
			"single_histogram_point",
			otlp.Metric{
				MetricDescriptor:    getDescriptor(name1+"", histogramComb, validCombinations),
				Int64DataPoints:     nil,
				DoubleDataPoints:    nil,
				HistogramDataPoints: []*otlp.HistogramDataPoint{histPoint},
				SummaryDataPoints:   nil,
			},
			false,
			map[string]*prompb.TimeSeries{
				sigs[sum]:       getTimeSeries(labels[sum], getSample(floatVal2, msTime1)),
				sigs[count]:     getTimeSeries(labels[count], getSample(float64(intVal2), msTime1)),
				sigs[bucket1]:   getTimeSeries(labels[bucket1], getSample(float64(intVal1), msTime1)),
				sigs[bucket2]:   getTimeSeries(labels[bucket2], getSample(float64(intVal1), msTime1)),
				sigs[bucketInf]: getTimeSeries(labels[bucketInf], getSample(float64(intVal2), msTime1)),
			},
		},
	}

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tsMap := map[string]*prompb.TimeSeries{}
			prw := &PrwExporter{}
			ok := prw.handleHistogramMetric(tsMap, &tt.m)
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

// Test_ NewPrwExporter checks that a new exporter instance with non-nil fields is initialized
func Test_NewPrwExporter(t *testing.T) {
	config := &Config{
		ExporterSettings:   configmodels.ExporterSettings{},
		TimeoutSettings:    exporterhelper.TimeoutSettings{},
		QueueSettings:      exporterhelper.QueueSettings{},
		RetrySettings:      exporterhelper.RetrySettings{},
		Namespace:          "",
		HTTPClientSettings: confighttp.HTTPClientSettings{Endpoint: ""},
	}
	tests := []struct {
		name        string
		config      *Config
		namespace   string
		endpoint    string
		client      *http.Client
		returnError bool
	}{
		{
			"invalid_URL",
			config,
			"test",
			"invalid URL",
			http.DefaultClient,
			true,
		},
		{
			"nil_client",
			config,
			"test",
			"http://some.url:9411/api/prom/push",
			nil,
			true,
		},
		{
			"success_case",
			config,
			"test",
			"http://some.url:9411/api/prom/push",
			http.DefaultClient,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prwe, err := NewPrwExporter(tt.namespace, tt.endpoint, tt.client)
			if tt.returnError {
				assert.Error(t, err)
				return
			}
			require.NotNil(t, prwe)
			assert.NotNil(t, prwe.namespace)
			assert.NotNil(t, prwe.endpointURL)
			assert.NotNil(t, prwe.client)
			assert.NotNil(t, prwe.closeChan)
			assert.NotNil(t, prwe.wg)
		})
	}
}

// Test_Shutdown checks after Shutdown is called, incoming calls to PushMetrics return error.
func Test_Shutdown(t *testing.T) {
	prwe := &PrwExporter{
		wg:        new(sync.WaitGroup),
		closeChan: make(chan struct{}),
	}
	wg := new(sync.WaitGroup)
	errChan := make(chan error, 5)
	err := prwe.Shutdown(context.Background())
	require.NoError(t, err)
	errChan = make(chan error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok := prwe.PushMetrics(context.Background(),
				pdatautil.MetricsFromOldInternalMetrics(testdataold.GenerateMetricDataEmpty()))
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
//Much akin to Test_PushMetrics
func Test_export(t *testing.T) {
	//First we will instantiate a dummy TimeSeries instance to pass into both the export call and compare the http request
	labels := getPromLabels(label11, value11, label12, value12, label21, value21, label22, value22)
	sample1 := getSample(floatVal1, msTime1)
	sample2 := getSample(floatVal2, msTime2)
	ts1 := getTimeSeries(labels, sample1, sample2)
	handleFunc := func(w http.ResponseWriter, r *http.Request, code int) {
		//The following is a handler function that reads the sent httpRequest, unmarshals, and checks if the WriteRequest
		//preserves the TimeSeries data correctly
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		require.NotNil(t, body)
		//Receives the http requests and unzip, unmarshals, and extracts TimeSeries
		assert.Equal(t, "0.1.0", r.Header.Get("X-Prometheus-Remote-Write-Version"))
		assert.Equal(t, "snappy", r.Header.Get("Content-Encoding"))
		writeReq := &prompb.WriteRequest{}
		unzipped := []byte{}

		dest, err := snappy.Decode(unzipped, body)
		require.NoError(t, err)

		ok := proto.Unmarshal(dest, writeReq)
		require.NoError(t, ok)

		assert.EqualValues(t, 1, len(writeReq.Timeseries))
		require.NotNil(t, writeReq.GetTimeseries())
		assert.Equal(t, *ts1, writeReq.GetTimeseries()[0])
		w.WriteHeader(code)
	}

	// Create in test table format to check if different HTTP response codes or server errors
	// are properly identified
	tests := []struct {
		name             string
		ts               prompb.TimeSeries
		serverUp         bool
		httpResponseCode int
		returnError      bool
	}{
		{"success_case",
			*ts1,
			true,
			http.StatusAccepted,
			false,
		},
		{
			"server_no_response_case",
			*ts1,
			false,
			http.StatusAccepted,
			true,
		}, {
			"error_status_code_case",
			*ts1,
			true,
			http.StatusForbidden,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if handleFunc != nil {
					handleFunc(w, r, tt.httpResponseCode)
				}
			}))
			defer server.Close()
			serverURL, uErr := url.Parse(server.URL)
			assert.NoError(t, uErr)
			if !tt.serverUp {
				server.Close()
			}
			err := runExportPipeline(t, ts1, serverURL)
			if tt.returnError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func runExportPipeline(t *testing.T, ts *prompb.TimeSeries, endpoint *url.URL) error {
	//First we will construct a TimeSeries array from the testutils package
	testmap := make(map[string]*prompb.TimeSeries)
	testmap["test"] = ts

	HTTPClient := http.DefaultClient
	//after this, instantiate a CortexExporter with the current HTTP client and endpoint set to passed in endpoint
	prwe, err := NewPrwExporter("test", endpoint.String(), HTTPClient)
	if err != nil {
		return err
	}
	err = prwe.export(context.Background(), testmap)
	return err
}

// Test_PushMetrics checks the number of TimeSeries received by server and the number of metrics dropped is the same as
// expected
func Test_PushMetrics(t *testing.T) {

	noTempBatch := pdatautil.MetricsFromOldInternalMetrics((testdataold.GenerateMetricDataManyMetricsSameResource(10)))
	invalidTypeBatch := pdatautil.MetricsFromOldInternalMetrics((testdataold.GenerateMetricDataMetricTypeInvalid()))
	nilDescBatch := pdatautil.MetricsFromOldInternalMetrics((testdataold.GenerateMetricDataNilMetricDescriptor()))

	// 10 counter metrics, 2 points in each. Two TimeSeries in total
	batch := testdataold.GenerateMetricDataManyMetricsSameResource(10)
	setCumulative(&batch)
	scalarBatch := pdatautil.MetricsFromOldInternalMetrics((batch))

	nilBatch1 := testdataold.GenerateMetricDataManyMetricsSameResource(10)
	nilBatch2 := testdataold.GenerateMetricDataManyMetricsSameResource(10)
	nilBatch3 := testdataold.GenerateMetricDataManyMetricsSameResource(10)
	nilBatch4 := testdataold.GenerateMetricDataManyMetricsSameResource(10)
	nilBatch5 := testdataold.GenerateMetricDataOneEmptyResourceMetrics()
	nilBatch6 := testdataold.GenerateMetricDataOneEmptyInstrumentationLibrary()
	nilBatch7 := testdataold.GenerateMetricDataOneMetric()

	nilResource := dataold.MetricDataToOtlp(nilBatch5)
	nilResource[0] = nil
	nilResourceBatch := pdatautil.MetricsFromOldInternalMetrics(dataold.MetricDataFromOtlp(nilResource))

	nilInstrumentation := dataold.MetricDataToOtlp(nilBatch6)
	nilInstrumentation[0].InstrumentationLibraryMetrics[0] = nil
	nilInstrumentationBatch := pdatautil.MetricsFromOldInternalMetrics(dataold.MetricDataFromOtlp(nilInstrumentation))

	nilMetric := dataold.MetricDataToOtlp(nilBatch7)
	nilMetric[0].InstrumentationLibraryMetrics[0].Metrics[0] = nil
	nilMetricBatch := pdatautil.MetricsFromOldInternalMetrics(dataold.MetricDataFromOtlp(nilMetric))

	setCumulative(&nilBatch1)
	setCumulative(&nilBatch2)
	setCumulative(&nilBatch3)
	setCumulative(&nilBatch4)

	setDataPointToNil(&nilBatch1, typeMonotonicInt64)
	setType(&nilBatch2, typeMonotonicDouble)
	setType(&nilBatch3, typeHistogram)
	setType(&nilBatch4, typeSummary)

	nilIntDataPointsBatch := pdatautil.MetricsFromOldInternalMetrics((nilBatch1))
	nilDoubleDataPointsBatch := pdatautil.MetricsFromOldInternalMetrics((nilBatch2))
	nilHistogramDataPointsBatch := pdatautil.MetricsFromOldInternalMetrics((nilBatch3))

	hist := dataold.MetricDataToOtlp(testdataold.GenerateMetricDataOneMetric())
	hist[0].InstrumentationLibraryMetrics[0].Metrics[0] = &otlp.Metric{
		MetricDescriptor: getDescriptor("hist_test", histogramComb, validCombinations),
		HistogramDataPoints: []*otlp.HistogramDataPoint{getHistogramDataPoint(
			lbs1,
			time1,
			floatVal1,
			uint64(intVal1),
			[]float64{floatVal1},
			[]uint64{uint64(intVal1)},
		),
		},
	}

	histBatch := pdatautil.MetricsFromOldInternalMetrics((dataold.MetricDataFromOtlp(hist)))
	checkFunc := func(t *testing.T, r *http.Request, expected int) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		buf := make([]byte, len(body))
		dest, err := snappy.Decode(buf, body)
		assert.Equal(t, "0.1.0", r.Header.Get("x-prometheus-remote-write-version"))
		assert.Equal(t, "snappy", r.Header.Get("content-encoding"))
		assert.NotNil(t, r.Header.Get("tenant-id"))
		require.NoError(t, err)
		wr := &prompb.WriteRequest{}
		ok := proto.Unmarshal(dest, wr)
		require.Nil(t, ok)
		assert.EqualValues(t, expected, len(wr.Timeseries))
	}

	summary := dataold.MetricDataToOtlp(testdataold.GenerateMetricDataOneMetric())
	summary[0].InstrumentationLibraryMetrics[0].Metrics[0] = &otlp.Metric{
		MetricDescriptor:  getDescriptor("summary_test", summaryComb, validCombinations),
		SummaryDataPoints: []*otlp.SummaryDataPoint{},
	}
	summaryBatch := pdatautil.MetricsFromOldInternalMetrics(dataold.MetricDataFromOtlp(summary))

	tests := []struct {
		name                 string
		md                   *pdata.Metrics
		reqTestFunc          func(t *testing.T, r *http.Request, expected int)
		expectedTimeSeries   int
		httpResponseCode     int
		numDroppedTimeSeries int
		returnErr            bool
	}{
		{
			"invalid_type_case",
			&invalidTypeBatch,
			nil,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(invalidTypeBatch),
			true,
		},
		{
			"nil_desc_case",
			&nilDescBatch,
			nil,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(nilDescBatch),
			true,
		},
		{
			"nil_resourece_case",
			&nilResourceBatch,
			nil,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(nilResourceBatch),
			false,
		},
		{
			"nil_instrumentation_case",
			&nilInstrumentationBatch,
			nil,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(nilInstrumentationBatch),
			false,
		},
		{
			"nil_metric_case",
			&nilMetricBatch,
			nil,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(nilMetricBatch),
			true,
		},
		{
			"nil_int_point_case",
			&nilIntDataPointsBatch,
			nil,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(nilIntDataPointsBatch),
			true,
		},
		{
			"nil_double_point_case",
			&nilDoubleDataPointsBatch,
			nil,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(nilDoubleDataPointsBatch),
			true,
		},
		{
			"nil_histogram_point_case",
			&nilHistogramDataPointsBatch,
			nil,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(nilHistogramDataPointsBatch),
			true,
		},
		{
			"nil_histogram_point_case",
			&nilHistogramDataPointsBatch,
			nil,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(nilHistogramDataPointsBatch),
			true,
		},
		{
			"no_temp_case",
			&noTempBatch,
			nil,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(noTempBatch),
			true,
		},
		{
			"http_error_case",
			&noTempBatch,
			nil,
			0,
			http.StatusForbidden,
			pdatautil.MetricCount(noTempBatch),
			true,
		},
		{
			"scalar_case",
			&scalarBatch,
			checkFunc,
			2,
			http.StatusAccepted,
			0,
			false,
		},
		{"histogram_case",
			&histBatch,
			checkFunc,
			4,
			http.StatusAccepted,
			0,
			false,
		},
		{"summary_case",
			&summaryBatch,
			checkFunc,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(summaryBatch),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.reqTestFunc != nil {
					tt.reqTestFunc(t, r, tt.expectedTimeSeries)
				}
				w.WriteHeader(tt.httpResponseCode)
			}))

			defer server.Close()

			serverURL, uErr := url.Parse(server.URL)
			assert.NoError(t, uErr)

			config := &Config{
				ExporterSettings: configmodels.ExporterSettings{
					TypeVal: "prometheusremotewrite",
					NameVal: "prometheusremotewrite",
				},
				Namespace: "",
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint: "http://some.url:9411/api/prom/push",
					// We almost read 0 bytes, so no need to tune ReadBufferSize.
					ReadBufferSize:  0,
					WriteBufferSize: 512 * 1024,
				},
			}
			assert.NotNil(t, config)
			// c, err := config.HTTPClientSettings.ToClient()
			// assert.Nil(t, err)
			c := http.DefaultClient
			prwe, nErr := NewPrwExporter(config.Namespace, serverURL.String(), c)
			require.NoError(t, nErr)
			numDroppedTimeSeries, err := prwe.PushMetrics(context.Background(), *tt.md)
			assert.Equal(t, tt.numDroppedTimeSeries, numDroppedTimeSeries)
			if tt.returnErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
