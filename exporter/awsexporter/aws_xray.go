// Copyright 2019, OpenCensus Authors
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

package awsexporter

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	xray "contrib.go.opencensus.io/exporter/aws"
	"go.opencensus.io/trace"

	"github.com/spf13/viper"

	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/exporter/exporterwrapper"
)

const defaultVersionForAWSXRayApplications = "latest"

type awsXRayConfig struct {
	Version            string        `mapstructure:"version"`
	BlacklistRegexes   []string      `mapstructure:"blacklist_regexes"`
	BufferSize         int           `mapstructure:"buffer_size"`
	BufferPeriod       time.Duration `mapstructure:"buffer_period"`
	DefaultServiceName string        `mapstructure:"default_service_name"`

	// DestinationRegion is an optional field that if set defines
	// the region to which the X-Ray payloads will be sent.
	DestinationRegion string `mapstructure:"destination_region"`
}

type awsXRayExporter struct {
	mu sync.RWMutex

	// exportersByServiceName shards AWS X-Ray OpenCensus-Go
	// Trace exporters by serviceName that's derived
	// from each Node of spans that this exporter receives.
	exportersByServiceName map[string]*xray.Exporter

	defaultServiceName string
	defaultOptions     []xray.Option
}

var _ consumer.TraceConsumer = (*awsXRayExporter)(nil)

// AWSXRayTraceExportersFromViper unmarshals the viper and returns an consumer.TraceConsumer targeting
// AWS X-Ray according to the configuration settings.
func AWSXRayTraceExportersFromViper(v *viper.Viper) (tps []consumer.TraceConsumer, mps []consumer.MetricsConsumer, doneFns []func() error, err error) {
	var cfg struct {
		AWSXRay *awsXRayConfig `mapstructure:"aws-xray"`
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, nil, nil, err
	}
	xc := cfg.AWSXRay
	if xc == nil {
		return nil, nil, nil, nil
	}

	defaultOptions, err := transformConfigToXRayOptions(xc)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("AWS-Xray: converting configuration to options: %v", err)
	}

	axe := &awsXRayExporter{
		exportersByServiceName: make(map[string]*xray.Exporter),
		defaultOptions:         defaultOptions,
		defaultServiceName:     xc.DefaultServiceName,
	}

	tps = append(tps, axe)
	doneFns = append(doneFns, func() error {
		axe.Flush()
		return nil
	})
	return
}

// Flush invokes .Flush() for every one of its underlying exporters.
func (axe *awsXRayExporter) Flush() {
	axe.mu.RLock()
	defer axe.mu.RUnlock()

	for _, exp := range axe.exportersByServiceName {
		if exp != nil {
			exp.Flush()
		}
	}
}

func transformConfigToXRayOptions(axrCfg *awsXRayConfig) (xopts []xray.Option, err error) {
	if axrCfg == nil {
		return nil, nil
	}

	// Compile any blacklist regexes.
	var blacklistRegexes []*regexp.Regexp
	for _, blacklistRegexStr := range axrCfg.BlacklistRegexes {
		blacklistRegex, err := regexp.Compile(blacklistRegexStr)
		if err != nil {
			return nil, fmt.Errorf("compiling %q error: %v", blacklistRegexStr, err)
		}
		blacklistRegexes = append(blacklistRegexes, blacklistRegex)
	}
	if len(blacklistRegexes) > 0 {
		xopts = append(xopts, xray.WithBlacklist(blacklistRegexes))
	}

	// Handle the buffer-size option.
	if axrCfg.BufferSize > 0 {
		xopts = append(xopts, xray.WithBufferSize(axrCfg.BufferSize))
	}

	if axrCfg.BufferPeriod > 0 {
		xopts = append(xopts, xray.WithInterval(axrCfg.BufferPeriod))
	}

	if axrCfg.DestinationRegion != "" {
		xopts = append(xopts, xray.WithRegion(axrCfg.DestinationRegion))
	}

	version := axrCfg.Version
	if version == "" {
		version = defaultVersionForAWSXRayApplications
	}
	xopts = append(xopts, xray.WithVersion(version))

	return xopts, nil
}

// ExportSpans is the method that translates OpenCensus-Proto Traces into AWS X-Ray spans.
// It uniquely maintains
func (axe *awsXRayExporter) ConsumeTraceData(ctx context.Context, td data.TraceData) (xerr error) {
	ctx, span := trace.StartSpan(ctx,
		"opencensus.service.exporter.aws_xray.ExportSpans",
		trace.WithSampler(trace.NeverSample()))

	defer func() {
		if xerr != nil && span.IsRecordingEvents() {
			span.SetStatus(trace.Status{
				Code:    int32(trace.StatusCodeUnknown),
				Message: xerr.Error(),
			})
		}
		span.End()
	}()

	serviceName := td.Node.GetServiceInfo().GetName()
	if serviceName == "" {
		serviceName = axe.defaultServiceName
	}
	if span.IsRecordingEvents() {
		span.Annotate([]trace.Attribute{
			trace.StringAttribute("service_name", serviceName),
		}, "")
	}

	exp, err := axe.getOrMakeExporterByServiceName(serviceName)
	if err != nil {
		return err
	}
	return exporterwrapper.PushOcProtoSpansToOCTraceExporter(exp, td)
}

func (axe *awsXRayExporter) getOrMakeExporterByServiceName(serviceName string) (*xray.Exporter, error) {
	axe.mu.Lock()
	defer axe.mu.Unlock()

	exp, ok := axe.exportersByServiceName[serviceName]
	if ok && exp != nil {
		return exp, nil
	}

	// Otherwise, this is the our first time creating this exporter,
	// so create it with the default options but finally the prescribed serviceName.
	opts := append(axe.defaultOptions, xray.WithServiceName(serviceName))
	exp, err := xray.NewExporter(opts...)
	if err != nil {
		return nil, err
	}

	// Now memoize the newly created AWS X-Ray exporter, for later lookups.
	axe.exportersByServiceName[serviceName] = exp

	return exp, nil
}
