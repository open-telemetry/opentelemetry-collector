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
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

// network metric constants

const (
	directionLabelName = "direction"
	stateLabelName     = "state"
)

const (
	receiveDirectionLabelValue  = "receive"
	transmitDirectionLabelValue = "transmit"
)

var metricNetworkPacketsDescriptor = createMetricNetworkPacketsDescriptor()

func createMetricNetworkPacketsDescriptor() pdata.MetricDescriptor {
	descriptor := pdata.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("host/network/packets")
	descriptor.SetDescription("The number of packets transferred.")
	descriptor.SetUnit("1")
	descriptor.SetType(pdata.MetricTypeCounterInt64)
	return descriptor
}

var metricNetworkDroppedPacketsDescriptor = createMetricNetworkDroppedPacketsDescriptor()

func createMetricNetworkDroppedPacketsDescriptor() pdata.MetricDescriptor {
	descriptor := pdata.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("host/network/dropped_packets")
	descriptor.SetDescription("The number of packets dropped.")
	descriptor.SetUnit("1")
	descriptor.SetType(pdata.MetricTypeCounterInt64)
	return descriptor
}

var metricNetworkErrorsDescriptor = createMetricNetworkErrorsDescriptor()

func createMetricNetworkErrorsDescriptor() pdata.MetricDescriptor {
	descriptor := pdata.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("host/network/errors")
	descriptor.SetDescription("The number of errors encountered")
	descriptor.SetUnit("1")
	descriptor.SetType(pdata.MetricTypeCounterInt64)
	return descriptor
}

var metricNetworkBytesDescriptor = createMetricNetworkBytesDescriptor()

func createMetricNetworkBytesDescriptor() pdata.MetricDescriptor {
	descriptor := pdata.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("host/network/bytes")
	descriptor.SetDescription("The number of bytes transmitted and received")
	descriptor.SetUnit("bytes")
	descriptor.SetType(pdata.MetricTypeCounterInt64)
	return descriptor
}

var metricNetworkTCPConnectionDescriptor = createMetricNetworkTCPConnectionDescriptor()

func createMetricNetworkTCPConnectionDescriptor() pdata.MetricDescriptor {
	descriptor := pdata.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("host/network/tcp_connections")
	descriptor.SetDescription("The number of tcp connections")
	descriptor.SetUnit("bytes")
	descriptor.SetType(pdata.MetricTypeGaugeInt64)
	return descriptor
}
