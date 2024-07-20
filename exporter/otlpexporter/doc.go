// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate env GITHUB_PROJECT=open-telemetry/opentelemetry-collector mdatagen metadata.yaml

// Package otlpexporter exports data by using the OTLP format to a gPRC endpoint.
package otlpexporter // import "go.opentelemetry.io/collector/exporter/otlpexporter"
