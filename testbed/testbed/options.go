// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package tests contains test cases. To run the tests go to tests directory and run:
// RUN_TESTBED=1 go test -v

package testbed

// TestCaseOption defines a TestCase option.
type TestCaseOption struct {
	option func(t *TestCase)
}

// Apply takes a TestCase and runs the option function on it.
func (o TestCaseOption) Apply(t *TestCase) {
	o.option(t)
}

// WithSkipResults option disables writing out results file for a TestCase.
func WithSkipResults() TestCaseOption {
	return TestCaseOption{func(t *TestCase) {
		t.skipResults = true
	}}
}

// WithConfigFile allows a custom configuration file for TestCase.
func WithConfigFile(file string) TestCaseOption {
	return TestCaseOption{func(t *TestCase) {
		t.agentConfigFile = file
	}}
}
