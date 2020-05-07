// Copyright 2020, OpenTelemetry Authors
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

package networkscraper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/net"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdatautil"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
)

// Scraper for Network Metrics
type Scraper struct {
	config    *Config
	consumer  consumer.MetricsConsumer
	startTime pdata.TimestampUnixNano
	cancel    context.CancelFunc
}

// NewNetworkScraper creates a set of Network related metrics
func NewNetworkScraper(ctx context.Context, cfg *Config, consumer consumer.MetricsConsumer) (*Scraper, error) {
	return &Scraper{config: cfg, consumer: consumer}, nil
}

// Start
func (c *Scraper) Start(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)

	bootTime, err := host.BootTime()
	if err != nil {
		return err
	}

	c.startTime = pdata.TimestampUnixNano(bootTime)

	go func() {
		ticker := time.NewTicker(c.config.CollectionInterval())
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.scrapeMetrics(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Shutdown
func (c *Scraper) Shutdown(ctx context.Context) error {
	c.cancel()
	return nil
}

func (c *Scraper) scrapeMetrics(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "networkscraper.scrapeMetrics")
	defer span.End()

	metricData := data.NewMetricData()
	metrics := internal.InitializeMetricSlice(metricData)

	err := c.scrapeAndAppendMetrics(metrics)
	if err != nil {
		span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Error(s) when scraping network metrics: %v", err)})
		return
	}

	if metrics.Len() > 0 {
		err := c.consumer.ConsumeMetrics(ctx, pdatautil.MetricsFromInternalMetrics(metricData))
		if err != nil {
			span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Unable to process metrics: %v", err)})
			return
		}
	}
}

// scrapeAndAppendMetrics appends scraped metrics to the provided metric slice.
// If errors occur scraping some metrics, an error will be returned, but the
// successfully scraped metrics will still be appended to the provided slice
func (c *Scraper) scrapeAndAppendMetrics(metrics pdata.MetricSlice) error {
	var errors []error

	err := scrapeAndAppendNetworkCounterMetrics(metrics, c.startTime)
	if err != nil {
		errors = append(errors, err)
	}

	err = scrapeAndAppendNetworkTCPConnectionsMetric(metrics)
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return componenterror.CombineErrors(errors)
	}

	return nil
}

func scrapeAndAppendNetworkCounterMetrics(metrics pdata.MetricSlice, startTime pdata.TimestampUnixNano) error {
	// get total stats only
	perNetworkInterfaceController := false

	networkStatsSlice, err := net.IOCounters(perNetworkInterfaceController)
	if err != nil {
		return err
	}

	networkStats := networkStatsSlice[0]

	startIdx := metrics.Len()
	metrics.Resize(startIdx + 4)
	initializeNetworkMetric(metrics.At(startIdx+0), metricNetworkPacketsDescriptor, startTime, networkStats.PacketsSent, networkStats.PacketsRecv)
	initializeNetworkMetric(metrics.At(startIdx+1), metricNetworkDroppedPacketsDescriptor, startTime, networkStats.Dropout, networkStats.Dropin)
	initializeNetworkMetric(metrics.At(startIdx+2), metricNetworkErrorsDescriptor, startTime, networkStats.Errout, networkStats.Errin)
	initializeNetworkMetric(metrics.At(startIdx+3), metricNetworkBytesDescriptor, startTime, networkStats.BytesSent, networkStats.BytesRecv)
	return nil
}

func initializeNetworkMetric(metric pdata.Metric, metricDescriptor pdata.MetricDescriptor, startTime pdata.TimestampUnixNano, transmitValue, receiveValue uint64) {
	metricDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(2)
	initializeNetworkDataPoint(idps.At(0), startTime, transmitDirectionLabelValue, int64(transmitValue))
	initializeNetworkDataPoint(idps.At(1), startTime, receiveDirectionLabelValue, int64(receiveValue))
}

func initializeNetworkDataPoint(dataPoint pdata.Int64DataPoint, startTime pdata.TimestampUnixNano, directionLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(directionLabelName, directionLabel)
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}

func scrapeAndAppendNetworkTCPConnectionsMetric(metrics pdata.MetricSlice) error {
	connections, err := net.Connections("tcp")
	if err != nil {
		return err
	}

	connectionStatusCounts := getConnectionStatusCounts(connections)

	metric := internal.AddNewMetric(metrics)
	initializeNetworkTCPConnectionsMetric(metric, connectionStatusCounts)
	return nil
}

func getConnectionStatusCounts(connections []net.ConnectionStat) map[string]int64 {
	connectionStatuses := make(map[string]int64, len(connections))
	for _, connection := range connections {
		connectionStatuses[strings.ToLower(connection.Status)]++
	}
	return connectionStatuses
}

func initializeNetworkTCPConnectionsMetric(metric pdata.Metric, connectionStateCounts map[string]int64) {
	metricNetworkTCPConnectionDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(len(connectionStateCounts))

	i := 0
	for connectionState, count := range connectionStateCounts {
		initializeTCPConnectionsDataPoint(idps.At(i), connectionState, count)
		i++
	}
}

func initializeTCPConnectionsDataPoint(dataPoint pdata.Int64DataPoint, stateLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(stateLabelName, stateLabel)
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(value)
}
