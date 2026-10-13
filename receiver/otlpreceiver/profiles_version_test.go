// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/profiles"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestHTTPProfilesDevelopmentVersion(t *testing.T) {
	t.Parallel()
	for _, contentType := range []string{pbContentType, jsonContentType} {
		for _, tt := range []struct {
			name   string
			values []string
			valid  bool
		}{
			{name: "absent", valid: pprofileotlp.DevelopmentVersion == "1"},
			{name: "supported", values: []string{pprofileotlp.DevelopmentVersion}, valid: true},
			{name: "unsupported", values: []string{pprofileotlp.DevelopmentVersion + "0"}},
			{name: "repeated", values: []string{pprofileotlp.DevelopmentVersion, pprofileotlp.DevelopmentVersion}},
		} {
			t.Run(contentType+"/"+tt.name, func(t *testing.T) {
				t.Parallel()
				sink := new(consumertest.ProfilesSink)
				set := receivertest.NewNopSettings(metadata.Type)
				obs, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
					ReceiverID: set.ID, Transport: "http", ReceiverCreateSettings: set,
				})
				require.NoError(t, err)
				receiver := profiles.New(sink, obs)

				// A malformed body distinguishes a decoding attempt from rejection
				// based on version metadata alone.
				body := []byte("invalid payload")
				if tt.valid {
					request := pprofileotlp.NewExportRequestFromProfiles(testdata.GenerateProfiles(1))
					if contentType == pbContentType {
						body, err = request.MarshalProto()
					} else {
						body, err = request.MarshalJSON()
					}
					require.NoError(t, err)
				}
				req := httptest.NewRequest(http.MethodPost, defaultProfilesURLPath, bytes.NewReader(body))
				req.Header.Set("Content-Type", contentType)
				for _, value := range tt.values {
					req.Header.Add("OTLP-Profiles-Development-Version", value)
				}
				resp := httptest.NewRecorder()
				handleProfiles(resp, req, receiver)
				assert.Equal(t, contentType, resp.Header().Get("Content-Type"))
				if tt.valid {
					assert.Equal(t, http.StatusOK, resp.Code)
					require.Len(t, sink.AllProfiles(), 1)
					return
				}
				assert.Equal(t, http.StatusBadRequest, resp.Code)
				assert.Empty(t, sink.AllProfiles())
				st := &spb.Status{}
				if contentType == pbContentType {
					require.NoError(t, proto.Unmarshal(resp.Body.Bytes(), st))
				} else {
					require.NoError(t, protojson.Unmarshal(resp.Body.Bytes(), st))
				}
				assert.Equal(t, int32(codes.InvalidArgument), st.Code)
				assert.Contains(t, st.Message, pprofileotlp.DevelopmentVersionHeader)
			})
		}
	}
}
