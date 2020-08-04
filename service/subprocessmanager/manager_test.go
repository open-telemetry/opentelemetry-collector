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

package subprocessmanager

import (
	"reflect"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestGetDelay(t *testing.T) {
	var (
		getDelayTests = []struct {
			name               string
			elapsed            time.Duration
			healthyProcessTime time.Duration
			crashCount         int
			healthyCrashCount  int
			want               time.Duration
		}{
			{
				name:               "healthy process 1",
				elapsed:            15 * time.Second,
				healthyProcessTime: 30 * time.Minute,
				crashCount:         2,
				healthyCrashCount:  3,
				want:               1 * time.Second,
			},
			{
				name:               "healthy process 2",
				elapsed:            15 * time.Hour,
				healthyProcessTime: 20 * time.Minute,
				crashCount:         6,
				healthyCrashCount:  2,
				want:               1 * time.Second,
			},
			{
				name:               "unhealthy process 1",
				elapsed:            15 * time.Second,
				healthyProcessTime: 45 * time.Minute,
				crashCount:         4,
				healthyCrashCount:  3,
			},
			{
				name:               "unhealthy process 2",
				elapsed:            15 * time.Second,
				healthyProcessTime: 75 * time.Second,
				crashCount:         5,
				healthyCrashCount:  3,
			},
			{
				name:               "unhealthy process 3",
				elapsed:            15 * time.Second,
				healthyProcessTime: 30 * time.Minute,
				crashCount:         6,
				healthyCrashCount:  3,
			},
			{
				name:               "unhealthy process 4",
				elapsed:            15 * time.Second,
				healthyProcessTime: 10 * time.Minute,
				crashCount:         7,
				healthyCrashCount:  3,
			},
		}
		previousResult time.Duration
	)

	for _, test := range getDelayTests {
		t.Run(test.name, func(t *testing.T) {
			got := GetDelay(test.elapsed, test.healthyProcessTime, test.crashCount, test.healthyCrashCount)
			if test.name == "healthy process" {
				if !reflect.DeepEqual(got, test.want) {
					t.Errorf("GetDelay() got = %v, want %v", got, test.want)
					return
				}
			}
			if previousResult > got {
				t.Errorf("GetDelay() got = %v, want something larger than the previous result %v", got, previousResult)
			}
			previousResult = got
		})
	}
}

func TestFormatEnvSlice(t *testing.T) {
	var formatEnvSliceTests = []struct {
		name     string
		envSlice *[]EnvSettings
		want     []string
		wantNil  bool
	}{
		{
			name:     "empty slice",
			envSlice: &[]EnvSettings{},
			want:     nil,
			wantNil:  true,
		},
		{
			name: "one entry",
			envSlice: &[]EnvSettings{
				{
					Name:  "DATA_SOURCE",
					Value: "password:username",
				},
			},
			want: []string{
				"DATA_SOURCE=password:username",
			},
			wantNil: false,
		},
		{
			name: "three entries",
			envSlice: &[]EnvSettings{
				{
					Name:  "DATA_SOURCE",
					Value: "password:username",
				},
				{
					Name:  "",
					Value: "",
				},
				{
					Name:  "john",
					Value: "doe",
				},
			},
			want: []string{
				"DATA_SOURCE=password:username",
				"=",
				"john=doe",
			},
			wantNil: false,
		},
	}

	for _, test := range formatEnvSliceTests {
		t.Run(test.name, func(t *testing.T) {
			got := formatEnvSlice(test.envSlice)
			if test.wantNil && got != nil {
				t.Errorf("formatEnvSlice() got = %v, wantNil %v", got, test.wantNil)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("formatEnvSlice() got = %v, want %v", got, test.want)
			}
		})
	}
}

func TestRun(t *testing.T) {
	var runTests = []struct {
		name        string
		process     *SubprocessSettings
		wantElapsed time.Duration
		wantErr     bool
	}{
		{
			name: "normal process 1, error process exit",
			process: &SubprocessSettings{
				Command: "go run testdata/test_crasher.go",
				Env: []EnvSettings{
					{
						Name:  "DATA_SOURCE",
						Value: "username:password@(url:port)/dbname",
					},
				},
			},
			wantElapsed: 4 * time.Millisecond,
			wantErr:     false,
		},
		{
			name: "normal process 2, normal process exit",
			process: &SubprocessSettings{
				Command: "go version",
				Env: []EnvSettings{
					{
						Name:  "DATA_SOURCE",
						Value: "username:password@(url:port)/dbname",
					},
				},
			},
			wantElapsed: 0 * time.Nanosecond,
			wantErr:     false,
		},
		{
			name: "shellquote error",
			process: &SubprocessSettings{
				Command: "command flag='something",
				Env:     []EnvSettings{},
			},
			wantElapsed: 0,
			wantErr:     true,
		},
	}

	for _, test := range runTests {
		t.Run(test.name, func(t *testing.T) {
			logger, _ := zap.NewProduction()
			got, err := test.process.Run(logger)
			if test.wantErr && err == nil {
				t.Errorf("Run() got = %v, wantErr %v", got, test.wantErr)
				return
			}
			if got < test.wantElapsed {
				t.Errorf("Run() got = %v, want larger than %v", got, test.wantElapsed)
			}
		})
	}
}
