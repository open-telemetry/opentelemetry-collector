// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	measureCount = stats.Int64("fooCount", "testing count metrics", stats.UnitBytes)
	measureSum   = stats.Int64("fooSum", "testing sum metrics", stats.UnitBytes)
	measureLast  = stats.Int64("fooLast", "testing LastValueData metrics", stats.UnitBytes)
	measureDist  = stats.Int64("fooHisto", "testing histogram metrics", stats.UnitDimensionless)
	testTags     []tag.Key
)

func newView(agg *view.Aggregation) *view.View {
	return &view.View{
		Name:        "fooCount",
		Description: "fooDesc",
		Measure:     measureCount,
		Aggregation: agg,
	}
}

func newCustomView(measureName string, agg *view.Aggregation, tags []tag.Key, measure *stats.Int64Measure) *view.View {
	return &view.View{
		Name:        measureName,
		Description: "fooDesc",
		Measure:     measureCount,
		TagKeys:     tags,
		Aggregation: agg,
	}
}

func TestExportView(t *testing.T) {
	reportPeriod := time.Millisecond
	exporter := testExporter(Options{})

	vd := newCustomView("fooCount", view.Count(), testTags, measureCount)
	if err := view.Register(vd); err != nil {
		t.Fatalf("Register error occurred: %v\n", err)
	}
	defer view.Unregister(vd)
	// Wait for exporter to process metrics
	<-time.After(10 * reportPeriod)

	ctx := context.Background()
	stats.Record(ctx, measureCount.M(1))
	<-time.After(10 * time.Millisecond)

	actual := exporter.view("fooCount")
	if actual != vd {
		t.Errorf("Expected: %v, Got: %v\n", vd, actual)
	}
}

func TestSanitizeString(t *testing.T) {
	testCases := []struct {
		input string
		want  string
	}{
		{"data-234_123!doge", "data_234_123_doge"},
		{"hello!good@morn#ing$test%", "hello_good_morn_ing_test_"},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			got := sanitizeString(tc.input)
			if got != tc.want {
				t.Errorf("Expected: %v, Got: %v\n", tc.want, got)
			}
		})
	}
}

func TestSanitizeMetricName(t *testing.T) {
	vd1 := newCustomView("fooGauge", view.Count(), testTags, measureCount)
	vd2 := newCustomView("bar-Sum", view.Sum(), testTags, measureSum)

	testCases := []struct {
		namespace string
		view      *view.View
		want      string
	}{
		{"opencensus", vd1, "opencensus.fooGauge"},
		{"data!doge", vd2, "data_doge.bar_Sum"},
	}
	for _, tc := range testCases {
		t.Run("Testing sanitizeMetricName", func(t *testing.T) {
			got := sanitizeMetricName(tc.namespace, tc.view)
			if got != tc.want {
				t.Errorf("Expected: %v, Got: %v\n", tc.want, got)
			}
		})
	}
}

func TestSignature(t *testing.T) {
	key, _ := tag.NewKey("signature")
	namespace := "opencensus"
	tags := append(testTags, key)
	vd := newCustomView("fooGauge", view.Count(), tags, measureCount)

	res := viewSignature(namespace, vd)
	exp := "opencensus.fooGauge_signature"
	if res != exp {
		t.Errorf("Expected: %v, Got: %v\n", exp, res)
	}
}

func TestTagMetrics(t *testing.T) {
	o := Options{}
	key, _ := tag.NewKey("testTags")
	tags := []tag.Tag{tag.Tag{Key: key, Value: "Metrics"}}
	customTag := []string{"program_name:main"}
	result := o.tagMetrics(tags, customTag)
	expected := []string{"testTags:Metrics", "program_name:main"}

	if n := len(expected); n == 0 {
		t.Fatal("got 0")
	}

	if !reflect.DeepEqual(expected, result) {
		t.Fatalf("Expected: %v, Got: %v\n", expected, result)
	}
}

func TestOnErrorNil(t *testing.T) {
	var buf bytes.Buffer
	opt := &Options{}
	testError := errors.New("Testing error")

	testCases := []struct {
		input error
		want  string
	}{
		{nil, fmt.Sprintf("Failed to export to Datadog: %v", nil)},
		{testError, "Testing error"},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing error: %v\n", tc.input), func(t *testing.T) {
			log.SetOutput(&buf)
			defer func() {
				log.SetOutput(os.Stderr)
			}()
			opt.onError(tc.input)
			got := buf.String()
			if !strings.Contains(got, tc.want) {
				t.Errorf("expected: %v, got: %v\n", tc.want, got)
			}
		})
	}
}

func TestCountData(t *testing.T) {
	reportPeriod := time.Millisecond
	exporter := testExporter(Options{})

	vd := newCustomView("fooCount", view.Count(), testTags, measureCount)
	if err := view.Register(vd); err != nil {
		t.Fatalf("Register error occurred: %v\n", err)
	}
	defer view.Unregister(vd)
	// Wait for exporter to process metrics
	<-time.After(10 * reportPeriod)

	ctx := context.Background()
	stats.Record(ctx, measureCount.M(1))
	<-time.After(10 * time.Millisecond)

	actual := exporter.view("fooCount")
	if actual != vd {
		t.Errorf("Expected: %v, Got: %v\n", vd, actual)
	}
}

func TestSumData(t *testing.T) {
	reportPeriod := time.Millisecond
	exporter := testExporter(Options{})

	vd := newCustomView("fooSum", view.Sum(), testTags, measureSum)
	if err := view.Register(vd); err != nil {
		t.Fatalf("Register error occurred: %v\n", err)
	}
	defer view.Unregister(vd)
	// Wait for exporter to process metrics
	<-time.After(10 * reportPeriod)

	ctx := context.Background()
	stats.Record(ctx, measureCount.M(1))
	<-time.After(10 * time.Millisecond)

	actual := exporter.view("fooSum")
	if actual != vd {
		t.Errorf("Expected: %v, Got: %v\n", vd, actual)
	}
}

func TestLastValueData(t *testing.T) {
	reportPeriod := time.Millisecond
	exporter := testExporter(Options{})

	vd := newCustomView("fooLast", view.LastValue(), testTags, measureLast)
	if err := view.Register(vd); err != nil {
		t.Fatalf("Register error occurred: %v\n", err)
	}
	defer view.Unregister(vd)
	// Wait for exporter to process metrics
	<-time.After(10 * reportPeriod)

	ctx := context.Background()
	stats.Record(ctx, measureCount.M(1))
	<-time.After(10 * time.Millisecond)

	actual := exporter.view("fooLast")
	if actual != vd {
		t.Errorf("Expected: %v, Got: %v\n", vd, actual)
	}
}

func TestHistogram(t *testing.T) {
	reportPeriod := time.Millisecond
	exporter := testExporter(Options{})

	vd := newCustomView("fooHisto", view.Distribution(), testTags, measureDist)
	if err := view.Register(vd); err != nil {
		t.Fatalf("Register error occurred: %v\n", err)
	}
	defer view.Unregister(vd)
	// Wait for exporter to process metrics
	<-time.After(10 * reportPeriod)

	ctx := context.Background()
	stats.Record(ctx, measureCount.M(1))
	<-time.After(10 * time.Millisecond)

	actual := exporter.view("fooHisto")
	if actual != vd {
		t.Errorf("Expected: %v, Got: %v\n", vd, actual)
	}
}
