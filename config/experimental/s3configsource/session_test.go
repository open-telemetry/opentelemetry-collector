package s3configsource

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestS3SessionForKVRetrieval(t *testing.T) {

	logger := zap.NewNop()

	config := Config{
		Region: "us-west-2",
		Bucket: "hasconfig",
		Key:    "retest_config.yaml",
	}

	cs, err := newConfigSource(logger, &config)
	require.NoError(t, err)
	require.NotNil(t, cs)

	s, err := cs.NewSession(context.Background())
	require.NoError(t, err)
	require.NotNil(t, s)

	retreived, err := s.Retrieve(context.Background(), "exporters::logging::loglevel", nil)
	require.NoError(t, err)
	require.Equal(t, "debug", retreived.Value().(string))

}
