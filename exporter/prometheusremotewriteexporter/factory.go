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

package prometheusremotewriteexporter

import (
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/pkg/errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	// The value of "type" key in configuration.
	typeStr    = "prometheusremotewrite"
	serviceStr = "servicname"
)

// Custom RoundTripper
type SigningRoundTripper struct {
	transport http.RoundTripper
	signer    *v4.Signer
	cfg       *aws.Config
}

// Custom RoundTrip
func (si *SigningRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Sign the request
	headers, err := si.signer.Sign(req, nil, serviceStr, *si.cfg.Region, time.Now())
	if err != nil {
		// might need a response here
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v[0])
	}
	// Send the request to Cortex
	response, err := si.transport.RoundTrip(req)

	return response, err
}

// the following are methods we would reimplement
func createClient(origClient *http.Client, region string) (*http.Client, error) {
	if origClient == nil {
		return nil, errors.Errorf("invalid http configuration")
	}
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if err != nil {
		return nil, err
	}
	// Get Credentials, either from ./aws or from environmental variables
	creds := sess.Config.Credentials
	signer := v4.NewSigner(creds)
	// Initialize Client with interceptor
	client := &http.Client{
		Transport: &SigningRoundTripper{
			transport: origClient.Transport,
			signer:    signer,
			cfg:       sess.Config,
		},
	}
	return client, nil

}

func NewFactory() component.ExporterFactory {
	return exporterhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		exporterhelper.WithMetrics(createMetricsExporter))
}

// Instantiates a pseudo-Cortex Exporter that adheres to the component MetricsExporter interface
func createMetricsExporter(_ context.Context, _ component.ExporterCreateParams,
	cfg configmodels.Exporter) (component.MetricsExporter, error) {

	cCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errors.Errorf("invalid configuration")
	}
	client, _ := cCfg.HTTPClientSettings.ToClient()
	client, err := createClient(client, cCfg.Region)

	if err != nil {
		return nil, err
	}

	prwe, err := newPrwExporter(cCfg.Namespace, cCfg.HTTPClientSettings.Endpoint, client, cCfg.Headers)

	if err != nil {
		return nil, err
	}

	prwexp, err := exporterhelper.NewMetricsExporter(
		cfg,
		prwe.pushMetrics,
		exporterhelper.WithTimeout(cCfg.TimeoutSettings),
		exporterhelper.WithQueue(cCfg.QueueSettings),
		exporterhelper.WithRetry(cCfg.RetrySettings),
		exporterhelper.WithShutdown(prwe.shutdown),
	)

	if err != nil {
		return nil, err
	}

	return prwexp, nil
}

func createDefaultConfig() configmodels.Exporter {
	// TODO: Enable the queued settings.
	qs := exporterhelper.CreateDefaultQueueSettings()
	qs.Enabled = false

	return &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		Namespace: "",
		Headers:   map[string]string{},

		TimeoutSettings: exporterhelper.CreateDefaultTimeoutSettings(),
		RetrySettings:   exporterhelper.CreateDefaultRetrySettings(),
		QueueSettings:   qs,
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: "http://some.url:9411/api/prom/push",
			// We almost read 0 bytes, so no need to tune ReadBufferSize.
			ReadBufferSize:  0,
			WriteBufferSize: 512 * 1024,
		},
	}
}
