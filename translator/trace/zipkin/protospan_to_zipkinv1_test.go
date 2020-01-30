// Copyright 2019 OpenTelemetry Authors
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

package zipkin

import (
	"net"
	"testing"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/stretchr/testify/assert"
)

func TestZipkinEndpointFromNode(t *testing.T) {
	type args struct {
		attributes   map[string]interface{}
		serviceName  string
		endpointType zipkinDirection
	}
	type want struct {
		endpoint      *zipkinmodel.Endpoint
		redundantKeys map[string]bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Nil attributes",
			args: args{attributes: nil, serviceName: "", endpointType: isLocalEndpoint},
			want: want{
				redundantKeys: make(map[string]bool),
			},
		},
		{
			name: "Only svc name",
			args: args{
				attributes:   make(map[string]interface{}),
				serviceName:  "test",
				endpointType: isLocalEndpoint,
			},
			want: want{
				endpoint:      &zipkinmodel.Endpoint{ServiceName: "test"},
				redundantKeys: make(map[string]bool),
			},
		},
		{
			name: "Only ipv4",
			args: args{
				attributes:   map[string]interface{}{"ipv4": "1.2.3.4"},
				serviceName:  "",
				endpointType: isLocalEndpoint,
			},
			want: want{
				endpoint:      &zipkinmodel.Endpoint{IPv4: net.ParseIP("1.2.3.4")},
				redundantKeys: map[string]bool{"ipv4": true},
			},
		},
		{
			name: "Only ipv6 remote",
			args: args{
				attributes:   map[string]interface{}{RemoteEndpointIPv6: "2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
				serviceName:  "",
				endpointType: isRemoteEndpoint,
			},
			want: want{
				endpoint:      &zipkinmodel.Endpoint{IPv6: net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334")},
				redundantKeys: map[string]bool{RemoteEndpointIPv6: true},
			},
		},
		{
			name: "Only port",
			args: args{
				attributes:   map[string]interface{}{"port": "42"},
				serviceName:  "",
				endpointType: isLocalEndpoint,
			},
			want: want{
				endpoint:      &zipkinmodel.Endpoint{Port: 42},
				redundantKeys: map[string]bool{"port": true},
			},
		},
		{
			name: "Service name, ipv4, and port",
			args: args{
				attributes:   map[string]interface{}{"ipv4": "4.3.2.1", "port": "2"},
				serviceName:  "test-svc",
				endpointType: isLocalEndpoint,
			},
			want: want{
				endpoint:      &zipkinmodel.Endpoint{ServiceName: "test-svc", IPv4: net.ParseIP("4.3.2.1"), Port: 2},
				redundantKeys: map[string]bool{"ipv4": true, "port": true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redundantKeys := make(map[string]bool)
			endpoint := zipkinEndpointFromAttributes(
				tt.args.attributes,
				tt.args.serviceName,
				tt.args.endpointType,
				redundantKeys)
			assert.Equal(t, tt.want.endpoint, endpoint)
			assert.Equal(t, tt.want.redundantKeys, redundantKeys)
		})
	}
}
