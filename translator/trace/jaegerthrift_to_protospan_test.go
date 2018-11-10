// Copyright 2018, OpenCensus Authors
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

package tracetranslator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/census-instrumentation/opencensus-service/internal/testutils"

	"github.com/jaegertracing/jaeger/thrift-gen/jaeger"
)

func TestJaegerThriftBatchToOCProto(t *testing.T) {
	const numOfFiles = 2
	for i := 1; i <= 2; i++ {
		thriftInFile := fmt.Sprintf("./testdata/thrift_batch_%02d.json", i)
		jb := &jaeger.Batch{}
		if err := loadFromJSON(thriftInFile, jb); err != nil {
			t.Errorf("Failed load Jaeger Thrift from %q. Error: %v", thriftInFile, err)
			continue
		}

		octrace, err := JaegerThriftBatchToOCProto(jb)
		if err != nil {
			t.Errorf("Failed to handled Jaeger Thrift Batch from %q. Error: %v", thriftInFile, err)
			continue
		}

		wantSpanCount, gotSpanCount := len(jb.Spans), len(octrace.Spans)
		if wantSpanCount != gotSpanCount {
			t.Errorf("Different number of spans in the batches on pass #%d (want %d, got %d)", i, wantSpanCount, gotSpanCount)
			continue
		}

		gb, err := json.MarshalIndent(octrace, "", "  ")
		if err != nil {
			t.Errorf("Failed to convert received OC proto to json. Error: %v", err)
			continue
		}

		protoFile := fmt.Sprintf("./testdata/ocproto_batch_%02d.json", i)
		wb, err := ioutil.ReadFile(protoFile)
		if err != nil {
			t.Errorf("Failed to read file %q with expected OC proto in JSON format. Error: %v", protoFile, err)
			continue
		}

		gj, wj := testutils.GenerateNormalizedJSON(string(gb)), testutils.GenerateNormalizedJSON(string(wb))
		if gj != wj {
			t.Errorf("The roundtrip JSON doesn't match the JSON that we want\nGot:\n%s\nWant:\n%s", gj, wj)
		}
	}
}

func loadFromJSON(file string, obj interface{}) error {
	blob, err := ioutil.ReadFile(file)
	if err == nil {
		err = json.Unmarshal(blob, obj)
	}

	return err
}
