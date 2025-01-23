// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalText(t *testing.T) {
	id := NewIDWithName(MustNewType("test"), "name")
	got, err := id.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, id.String(), string(got))
}

func TestUnmarshalText(t *testing.T) {
	validType := MustNewType("valid_type")
	testCases := []struct {
		name        string
		expectedErr bool
		expectedID  ID
	}{
		{
			name:       "valid_type",
			expectedID: ID{typeVal: validType, nameVal: ""},
		},
		{
			name:       "valid_type/valid_name",
			expectedID: ID{typeVal: validType, nameVal: "valid_name"},
		},
		{
			name:       "   valid_type   /   valid_name  ",
			expectedID: ID{typeVal: validType, nameVal: "valid_name"},
		},
		{
			name:       "valid_type/中文好",
			expectedID: ID{typeVal: validType, nameVal: "中文好"},
		},
		{
			name:       "valid_type/name-with-dashes",
			expectedID: ID{typeVal: validType, nameVal: "name-with-dashes"},
		},
		// issue 10816
		{
			name:       "valid_type/Linux-Messages-File_01J49HCH3SWFXRVASWFZFRT3J2__processor0__logs",
			expectedID: ID{typeVal: validType, nameVal: "Linux-Messages-File_01J49HCH3SWFXRVASWFZFRT3J2__processor0__logs"},
		},
		{
			name:       "valid_type/1",
			expectedID: ID{typeVal: validType, nameVal: "1"},
		},
		{
			name:        "/valid_name",
			expectedErr: true,
		},
		{
			name:        "     /valid_name",
			expectedErr: true,
		},
		{
			name:        "valid_type/",
			expectedErr: true,
		},
		{
			name:        "valid_type/      ",
			expectedErr: true,
		},
		{
			name:        "      ",
			expectedErr: true,
		},
		{
			name:        "valid_type/invalid name",
			expectedErr: true,
		},
		{
			name:        "valid_type/" + strings.Repeat("a", 1025),
			expectedErr: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			id := ID{}
			err := id.UnmarshalText([]byte(tt.name))
			if tt.expectedErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedID.Type(), id.Type())
			assert.Equal(t, tt.expectedID.Name(), id.Name())
			assert.Equal(t, tt.expectedID.String(), id.String())
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
		{name: "logicmonitor"},
		{name: "logstransform"},
		{name: "logzio"},
		{name: "loki"},
		{name: "mdatagen"},
		{name: "memcached"},
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
