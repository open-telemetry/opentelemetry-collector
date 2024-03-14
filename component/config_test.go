// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
)

var _ fmt.Stringer = Type{}

type configChildStruct struct {
	Child    errConfig
	ChildPtr *errConfig
}

type configChildSlice struct {
	Child    []errConfig
	ChildPtr []*errConfig
}

type configChildMapValue struct {
	Child    map[string]errConfig
	ChildPtr map[string]*errConfig
}

type configChildMapKey struct {
	Child    map[errType]string
	ChildPtr map[*errType]string
}

type configChildTypeDef struct {
	Child    errType
	ChildPtr *errType
}

type errConfig struct {
	err error
}

func (e *errConfig) Validate() error {
	return e.err
}

type errType string

func (e errType) Validate() error {
	if e == "" {
		return nil
	}
	return errors.New(string(e))
}

func newErrType(etStr string) *errType {
	et := errType(etStr)
	return &et
}

type errMapType map[string]string

func (e errMapType) Validate() error {
	return errors.New(e["err"])
}

func newErrMapType() *errMapType {
	et := errMapType(nil)
	return &et
}

func TestValidateConfig(t *testing.T) {
	var tests = []struct {
		name     string
		cfg      any
		expected error
	}{
		{
			name:     "struct",
			cfg:      errConfig{err: errors.New("struct")},
			expected: errors.New("struct"),
		},
		{
			name:     "pointer struct",
			cfg:      &errConfig{err: errors.New("pointer struct")},
			expected: errors.New("pointer struct"),
		},
		{
			name:     "type",
			cfg:      errType("type"),
			expected: errors.New("type"),
		},
		{
			name:     "pointer child",
			cfg:      newErrType("pointer type"),
			expected: errors.New("pointer type"),
		},
		{
			name:     "nil",
			cfg:      nil,
			expected: nil,
		},
		{
			name:     "nil map type",
			cfg:      errMapType(nil),
			expected: errors.New(""),
		},
		{
			name:     "nil pointer map type",
			cfg:      newErrMapType(),
			expected: errors.New(""),
		},
		{
			name:     "child struct",
			cfg:      configChildStruct{Child: errConfig{err: errors.New("child struct")}},
			expected: errors.New("child struct"),
		},
		{
			name:     "pointer child struct",
			cfg:      &configChildStruct{Child: errConfig{err: errors.New("pointer child struct")}},
			expected: errors.New("pointer child struct"),
		},
		{
			name:     "child struct pointer",
			cfg:      &configChildStruct{ChildPtr: &errConfig{err: errors.New("child struct pointer")}},
			expected: errors.New("child struct pointer"),
		},
		{
			name:     "child slice",
			cfg:      configChildSlice{Child: []errConfig{{}, {err: errors.New("child slice")}}},
			expected: errors.New("child slice"),
		},
		{
			name:     "pointer child slice",
			cfg:      &configChildSlice{Child: []errConfig{{}, {err: errors.New("pointer child slice")}}},
			expected: errors.New("pointer child slice"),
		},
		{
			name:     "child slice pointer",
			cfg:      &configChildSlice{ChildPtr: []*errConfig{{}, {err: errors.New("child slice pointer")}}},
			expected: errors.New("child slice pointer"),
		},
		{
			name:     "child map value",
			cfg:      configChildMapValue{Child: map[string]errConfig{"test": {err: errors.New("child map")}}},
			expected: errors.New("child map"),
		},
		{
			name:     "pointer child map value",
			cfg:      &configChildMapValue{Child: map[string]errConfig{"test": {err: errors.New("pointer child map")}}},
			expected: errors.New("pointer child map"),
		},
		{
			name:     "child map value pointer",
			cfg:      &configChildMapValue{ChildPtr: map[string]*errConfig{"test": {err: errors.New("child map pointer")}}},
			expected: errors.New("child map pointer"),
		},
		{
			name:     "child map key",
			cfg:      configChildMapKey{Child: map[errType]string{"child map key": ""}},
			expected: errors.New("child map key"),
		},
		{
			name:     "pointer child map key",
			cfg:      &configChildMapKey{Child: map[errType]string{"pointer child map key": ""}},
			expected: errors.New("pointer child map key"),
		},
		{
			name:     "child map key pointer",
			cfg:      &configChildMapKey{ChildPtr: map[*errType]string{newErrType("child map key pointer"): ""}},
			expected: errors.New("child map key pointer"),
		},
		{
			name:     "child type",
			cfg:      configChildTypeDef{Child: "child type"},
			expected: errors.New("child type"),
		},
		{
			name:     "pointer child type",
			cfg:      &configChildTypeDef{Child: "pointer child type"},
			expected: errors.New("pointer child type"),
		},
		{
			name:     "child type pointer",
			cfg:      &configChildTypeDef{ChildPtr: newErrType("child type pointer")},
			expected: errors.New("child type pointer"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(reflect.ValueOf(tt.cfg))
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestNewType(t *testing.T) {
	tests := []struct {
		name      string
		shouldErr bool
	}{
		{name: "active_directory_ds"},
		{name: "aerospike"},
		{name: "alertmanager"},
		{name: "alibabacloud_logservice"},
		{name: "apache"},
		{name: "apachespark"},
		{name: "asapclient"},
		{name: "attributes"},
		{name: "awscloudwatch"},
		{name: "awscloudwatchlogs"},
		{name: "awscloudwatchmetrics"},
		{name: "awscontainerinsightreceiver"},
		{name: "awsecscontainermetrics"},
		{name: "awsemf"},
		{name: "awsfirehose"},
		{name: "awskinesis"},
		{name: "awsproxy"},
		{name: "awss3"},
		{name: "awsxray"},
		{name: "azureblob"},
		{name: "azuredataexplorer"},
		{name: "azureeventhub"},
		{name: "azuremonitor"},
		{name: "basicauth"},
		{name: "batch"},
		{name: "bearertokenauth"},
		{name: "bigip"},
		{name: "carbon"},
		{name: "cassandra"},
		{name: "chrony"},
		{name: "clickhouse"},
		{name: "cloudflare"},
		{name: "cloudfoundry"},
		{name: "collectd"},
		{name: "configschema"},
		{name: "coralogix"},
		{name: "couchdb"},
		{name: "count"},
		{name: "cumulativetodelta"},
		{name: "datadog"},
		{name: "dataset"},
		{name: "db_storage"},
		{name: "debug"},
		{name: "deltatorate"},
		{name: "demo"},
		{name: "docker_observer"},
		{name: "docker_stats"},
		{name: "dynatrace"},
		{name: "ecs_observer"},
		{name: "ecs_task_observer"},
		{name: "elasticsearch"},
		{name: "exceptions"},
		{name: "experimental_metricsgeneration"},
		{name: "expvar"},
		{name: "f5cloud"},
		{name: "failover"},
		{name: "file"},
		{name: "filelog"},
		{name: "filestats"},
		{name: "file_storage"},
		{name: "filter"},
		{name: "flinkmetrics"},
		{name: "fluentforward"},
		{name: "forward"},
		{name: "githubgen"},
		{name: "gitprovider"},
		{name: "golden"},
		{name: "googlecloud"},
		{name: "googlecloudpubsub"},
		{name: "googlecloudspanner"},
		{name: "googlemanagedprometheus"},
		{name: "groupbyattrs"},
		{name: "groupbytrace"},
		{name: "haproxy"},
		{name: "headers_setter"},
		{name: "health_check"},
		{name: "honeycombmarker"},
		{name: "hostmetrics"},
		{name: "host_observer"},
		{name: "httpcheck"},
		{name: "http_forwarder"},
		{name: "iis"},
		{name: "influxdb"},
		{name: "instana"},
		{name: "interval"},
		{name: "jaeger"},
		{name: "jaeger_encoding"},
		{name: "jaegerremotesampling"},
		{name: "jmx"},
		{name: "journald"},
		{name: "json_log_encoding"},
		{name: "k8sattributes"},
		{name: "k8s_cluster"},
		{name: "k8s_events"},
		{name: "k8sobjects"},
		{name: "k8s_observer"},
		{name: "kafka"},
		{name: "kafkametrics"},
		{name: "kinetica"},
		{name: "kubeletstats"},
		{name: "loadbalancing"},
		{name: "logging"},
		{name: "logicmonitor"},
		{name: "logstransform"},
		{name: "logzio"},
		{name: "loki"},
		{name: "mdatagen"},
		{name: "memcached"},
		{name: "memory_ballast"},
		{name: "memory_limiter"},
		{name: "metricstransform"},
		{name: "mezmo"},
		{name: "mongodb"},
		{name: "mongodbatlas"},
		{name: "mysql"},
		{name: "namedpipe"},
		{name: "nginx"},
		{name: "nsxt"},
		{name: "oauth2client"},
		{name: "oidc"},
		{name: "opamp"},
		{name: "opampsupervisor"},
		{name: "opencensus"},
		{name: "opensearch"},
		{name: "oracledb"},
		{name: "osquery"},
		{name: "otelarrow"},
		{name: "otelcontribcol"},
		{name: "oteltestbedcol"},
		{name: "otlp"},
		{name: "otlp_encoding"},
		{name: "otlphttp"},
		{name: "otlpjsonfile"},
		{name: "ottl"},
		{name: "podman_stats"},
		{name: "postgresql"},
		{name: "pprof"},
		{name: "probabilistic_sampler"},
		{name: "prometheus"},
		{name: "prometheusremotewrite"},
		{name: "prometheus_simple"},
		{name: "pulsar"},
		{name: "purefa"},
		{name: "purefb"},
		{name: "rabbitmq"},
		{name: "receiver_creator"},
		{name: "redaction"},
		{name: "redis"},
		{name: "remotetap"},
		{name: "resource"},
		{name: "resourcedetection"},
		{name: "riak"},
		{name: "routing"},
		{name: "saphana"},
		{name: "sapm"},
		{name: "schema"},
		{name: "sentry"},
		{name: "servicegraph"},
		{name: "signalfx"},
		{name: "sigv4auth"},
		{name: "skywalking"},
		{name: "snmp"},
		{name: "snowflake"},
		{name: "solace"},
		{name: "solarwindsapmsettings"},
		{name: "span"},
		{name: "spanmetrics"},
		{name: "splunkenterprise"},
		{name: "splunk_hec"},
		{name: "sqlquery"},
		{name: "sqlserver"},
		{name: "sshcheck"},
		{name: "statsd"},
		{name: "sumologic"},
		{name: "syslog"},
		{name: "tail_sampling"},
		{name: "tcplog"},
		{name: "telemetrygen"},
		{name: "tencentcloud_logservice"},
		{name: "text_encoding"},
		{name: "transform"},
		{name: "udplog"},
		{name: "vcenter"},
		{name: "wavefront"},
		{name: "webhookevent"},
		{name: "windowseventlog"},
		{name: "windowsperfcounters"},
		{name: "zipkin"},
		{name: "zipkin_encoding"},
		{name: "zookeeper"},
		{name: "zpages"},
		{name: strings.Repeat("a", 63)},

		{name: "", shouldErr: true},
		{name: "contains spaces", shouldErr: true},
		{name: "contains-dashes", shouldErr: true},
		{name: "0startswithnumber", shouldErr: true},
		{name: "contains/slash", shouldErr: true},
		{name: "contains:colon", shouldErr: true},
		{name: "contains#hash", shouldErr: true},
		{name: strings.Repeat("a", 64), shouldErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ty, err := NewType(tt.name)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.name, ty.String())
			}
		})
	}
}

type configWithEmbeddedStruct struct {
	String string `mapstructure:"string"`
	Num    int    `mapstructure:"num"`
	embeddedUnmarshallingConfig
}

type embeddedUnmarshallingConfig struct {
}

func (euc *embeddedUnmarshallingConfig) Unmarshal(_ *confmap.Conf) error {
	return nil // do nothing.
}
func TestStructWithEmbeddedUnmarshaling(t *testing.T) {
	t.Skip("Skipping, to be fixed with https://github.com/open-telemetry/opentelemetry-collector/issues/7102")
	cfgMap := confmap.NewFromStringMap(map[string]any{
		"string": "foo",
		"num":    123,
	})
	tc := &configWithEmbeddedStruct{}
	err := UnmarshalConfig(cfgMap, tc)
	require.NoError(t, err)
	assert.Equal(t, "foo", tc.String)
	assert.Equal(t, 123, tc.Num)
}
