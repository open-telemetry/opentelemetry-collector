package pprofile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/jzelinskie/must"
	"github.com/pyroscope-io/pyroscope/pkg/convert/jfr"
	"github.com/pyroscope-io/pyroscope/pkg/storage"
	"github.com/pyroscope-io/pyroscope/pkg/storage/segment"
	"github.com/pyroscope-io/pyroscope/pkg/storage/tree"

	"go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1/alternatives/pprof"
)

type putter struct {
	profiles []*pprof.Profile
}

func (p *putter) Put(_ context.Context, pi *storage.PutInput) error {
	converted := pi.Val.Pprof(&tree.PprofMetadata{
		Type: "cpu",
		Unit: "samples",
		// PeriodType: "cpu",
		// PeriodUnit: "ms",
		// Period:     100,
		StartTime: time.Now(),
		Duration:  10 * time.Second,
	})
	b := must.NotError(proto.Marshal(converted))
	var res pprof.Profile
	err := proto.Unmarshal(b, &res)
	if err != nil {
		panic(err)
	}

	p.profiles = append(p.profiles, &res)
	return nil
}

func readJfr(filename string) []*pprof.Profile {
	b, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	b = ungzipIfNeeded(b)

	putter := putter{}

	pi := &storage.PutInput{
		Key: must.NotError(segment.ParseKey("test{foo=\"bar\"}")),
	}
	err = jfr.ParseJFR(context.TODO(), &putter, bytes.NewReader(b), pi, &jfr.LabelsSnapshot{})
	if err != nil {
		return nil
	}
	// chunks := must.NotError(parser.Parse(bytes.NewReader(b)))
	// for _, c := range chunks {
	// 	if pErr := parseChunk(c); pErr != nil {
	// 		panic(pErr)
	// 	}
	// }

	// // var p pprof.Profile
	// // err := proto.Unmarshal(b, &p)
	// // if err != nil {
	// // 	panic(err)
	// // }
	// return p
	fmt.Printf("jfr %s length: %d\n", filename, len(putter.profiles))
	return putter.profiles
}
