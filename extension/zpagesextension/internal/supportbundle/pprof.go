// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package supportbundle // import "go.opentelemetry.io/collector/extension/zpagesextension/internal/supportbundle"

import (
	"bytes"
	"context"
	"fmt"
	"runtime/pprof"
)

// PProfName is the name of the pprof gatherer.
const PProfName = "pprof"

// snapshotProfiles are the profiles that need no collection window.
var snapshotProfiles = []string{"heap", "goroutine"}

// PProfGatherer collects point-in-time runtime profiles in the pprof format.
type PProfGatherer struct{}

func (PProfGatherer) Name() string { return PProfName }

func (PProfGatherer) Gather(ctx context.Context) ([]File, error) {
	files := make([]File, 0, len(snapshotProfiles))
	for _, name := range snapshotProfiles {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		// The runtime always defines these profiles, so Lookup cannot return nil.
		// A debug value of 0 writes the gzipped protobuf format that "go tool pprof" reads.
		if err := pprof.Lookup(name).WriteTo(&buf, 0); err != nil {
			return nil, fmt.Errorf("write %s profile: %w", name, err)
		}
		files = append(files, File{Path: "pprof/" + name + ".pprof", Content: buf.Bytes()})
	}
	return files, nil
}
