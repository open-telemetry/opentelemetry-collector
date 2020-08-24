// Copyright The OpenTelemetry Authors
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

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_formatIdentifier(t *testing.T) {
	var tests = []struct {
		input    string
		want     string
		exported bool
		wantErr  string
	}{
		// Unexported.
		{input: "max.cpu", want: "maxCPU"},
		{input: "max.foo", want: "maxFoo"},
		{input: "cpu.utilization", want: "cpuUtilization"},
		{input: "cpu", want: "cpu"},
		{input: "max.ip.addr", want: "maxIPAddr"},
		{input: "some_metric", want: "someMetric"},
		{input: "some-metric", want: "someMetric"},
		{input: "Upper.Case", want: "upperCase"},
		{input: "max.ip6", want: "maxIP6"},
		{input: "max.ip6.idle", want: "maxIP6Idle"},
		{input: "node_netstat_IpExt_OutOctets", want: "nodeNetstatIPExtOutOctets"},

		// Exported.
		{input: "cpu.state", want: "CPUState", exported: true},

		// Errors
		{input: "", want: "", wantErr: "string cannot be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := formatIdentifier(tt.input, tt.exported)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
