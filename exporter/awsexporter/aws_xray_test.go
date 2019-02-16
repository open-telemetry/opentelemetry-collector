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
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	xray "contrib.go.opencensus.io/exporter/aws"
)

func TestTransformConfigToXRayOptions(t *testing.T) {
	testCases := []struct {
		config  *awsXRayConfig
		want    []xray.Option
		wantErr string
	}{
		{
			config: nil,
			want:   nil,
		},
		{
			config: &awsXRayConfig{
				Version:           "0.12.9",
				BlacklistRegexes:  []string{"foo.+", "[a-z]+"},
				BufferSize:        10,
				BufferPeriod:      379 * time.Millisecond,
				DestinationRegion: "us-west-2",
			},
			want: []xray.Option{
				xray.WithBlacklist([]*regexp.Regexp{
					regexp.MustCompile("foo.+"), regexp.MustCompile("[a-z]+"),
				}),
				xray.WithBufferSize(10),
				xray.WithInterval(379 * time.Millisecond),
				xray.WithRegion("us-west-2"),
				xray.WithVersion("0.12.9"),
			},
		},
		{
			config: &awsXRayConfig{
				Version:          "0.12.9",
				BlacklistRegexes: []string{"+"},
			},
			wantErr: "error parsing regexp: missing argument to repetition operator: `+`",
		},
	}

	for i, tt := range testCases {
		got, err := transformConfigToXRayOptions(tt.config)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("#%d:\nGot: %q\nWant substring: %q\n", i, err, tt.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("#%d: Unexpected error: %v", i, err)
			continue
		}

		ng, nw := len(got), len(tt.want)
		if ng != nw {
			t.Errorf("#%d: got len(options)=%d want len(options)=%d", i, ng, nw)
			continue
		}

		// Comparing the slice of options. This is a horrendous
		// comparison but the closest that we can do for now.
		addrStr := func(v interface{}) string { return fmt.Sprintf("%p", v) }

		for j := 0; j < ng; j++ {
			sg, sw := addrStr(got[j]), addrStr(tt.want[j])
			// Since these are unexported options that could be anything,
			// the best that we could do is compare pointer addresses.
			if sg != sw {
				t.Errorf("#%d:\nGot:\n%v\nWant:\n%v\n", i, sg, sw)
			}
		}
	}
}
