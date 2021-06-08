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

package oidcauthextension

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configauth"
)

func TestOIDCAuthenticationSucceeded(t *testing.T) {
	// prepare
	oidcServer, err := newOIDCServer()
	require.NoError(t, err)
	oidcServer.Start()
	defer oidcServer.Close()

	config := &Config{
		IssuerURL:   oidcServer.URL,
		Audience:    "unit-test",
		GroupsClaim: "memberships",
	}
	p, err := newExtension(config, zap.NewNop())
	require.NoError(t, err)

	err = p.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	payload, _ := json.Marshal(map[string]interface{}{
		"sub":         "jdoe@example.com",
		"name":        "jdoe",
		"iss":         oidcServer.URL,
		"aud":         "unit-test",
		"exp":         time.Now().Add(time.Minute).Unix(),
		"memberships": []string{"department-1", "department-2"},
	})
	token, err := oidcServer.token(payload)
	require.NoError(t, err)

	// test
	err = p.Authenticate(context.Background(), map[string][]string{"authorization": {fmt.Sprintf("Bearer %s", token)}})

	// verify
	assert.NoError(t, err)

	// TODO(jpkroehling): assert that the authentication routine set the subject/membership to the resource
}

func TestOIDCProviderForConfigWithTLS(t *testing.T) {
	// prepare the CA cert for the TLS handler
	cert := x509.Certificate{
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(10 * time.Second),
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		SerialNumber: big.NewInt(9447457), // some number
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	x509Cert, err := x509.CreateCertificate(rand.Reader, &cert, &cert, &priv.PublicKey, priv)
	require.NoError(t, err)

	caFile, err := ioutil.TempFile(os.TempDir(), "cert")
	require.NoError(t, err)
	defer os.Remove(caFile.Name())

	err = pem.Encode(caFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: x509Cert,
	})
	require.NoError(t, err)

	oidcServer, err := newOIDCServer()
	require.NoError(t, err)
	defer oidcServer.Close()

	tlsCert := tls.Certificate{
		Certificate: [][]byte{x509Cert},
		PrivateKey:  priv,
	}
	oidcServer.TLS = &tls.Config{Certificates: []tls.Certificate{tlsCert}}
	oidcServer.StartTLS()

	// prepare the processor configuration
	config := &Config{
		IssuerURL:    oidcServer.URL,
		IssuerCAPath: caFile.Name(),
		Audience:     "unit-test",
	}

	// test
	provider, err := getProviderForConfig(config)

	// verify
	assert.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestOIDCLoadIssuerCAFromPath(t *testing.T) {
	// prepare
	cert := x509.Certificate{
		SerialNumber: big.NewInt(9447457), // some number
		IsCA:         true,
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	x509Cert, err := x509.CreateCertificate(rand.Reader, &cert, &cert, &priv.PublicKey, priv)
	require.NoError(t, err)

	file, err := ioutil.TempFile(os.TempDir(), "cert")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	err = pem.Encode(file, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: x509Cert,
	})
	require.NoError(t, err)

	// test
	loaded, err := getIssuerCACertFromPath(file.Name())

	// verify
	assert.NoError(t, err)
	assert.Equal(t, cert.SerialNumber, loaded.SerialNumber)
}

func TestOIDCFailedToLoadIssuerCAFromPathEmptyCert(t *testing.T) {
	// prepare
	file, err := ioutil.TempFile(os.TempDir(), "cert")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	// test
	loaded, err := getIssuerCACertFromPath(file.Name()) // the file exists, but the contents isn't a cert

	// verify
	assert.Error(t, err)
	assert.Nil(t, loaded)
}

func TestOIDCFailedToLoadIssuerCAFromPathMissingFile(t *testing.T) {
	// test
	loaded, err := getIssuerCACertFromPath("some-non-existing-file")

	// verify
	assert.Error(t, err)
	assert.Nil(t, loaded)
}

func TestOIDCFailedToLoadIssuerCAFromPathInvalidContent(t *testing.T) {
	// prepare
	file, err := ioutil.TempFile(os.TempDir(), "cert")
	require.NoError(t, err)
	defer os.Remove(file.Name())
	_, err = file.Write([]byte("foobar"))
	require.NoError(t, err)

	config := &Config{
		IssuerCAPath: file.Name(),
	}

	// test
	provider, err := getProviderForConfig(config) // cross test with getIssuerCACertFromPath

	// verify
	assert.Error(t, err)
	assert.Nil(t, provider)
}

func TestOIDCInvalidAuthHeader(t *testing.T) {
	// prepare
	p, err := newExtension(&Config{
		Audience:  "some-audience",
		IssuerURL: "http://example.com",
	}, zap.NewNop())
	require.NoError(t, err)

	// test
	err = p.Authenticate(context.Background(), map[string][]string{"authorization": {"some-value"}})

	// verify
	assert.Equal(t, errInvalidAuthenticationHeaderFormat, err)
}

func TestOIDCNotAuthenticated(t *testing.T) {
	// prepare
	p, err := newExtension(&Config{
		Audience:  "some-audience",
		IssuerURL: "http://example.com",
	}, zap.NewNop())
	require.NoError(t, err)

	// test
	err = p.Authenticate(context.Background(), make(map[string][]string))

	// verify
	assert.Equal(t, errNotAuthenticated, err)
}

func TestProviderNotReacheable(t *testing.T) {
	// prepare
	p, err := newExtension(&Config{
		Audience:  "some-audience",
		IssuerURL: "http://example.com",
	}, zap.NewNop())
	require.NoError(t, err)

	// test
	err = p.Start(context.Background(), componenttest.NewNopHost())

	// verify
	assert.Error(t, err)
}

func TestFailedToVerifyToken(t *testing.T) {
	// prepare
	oidcServer, err := newOIDCServer()
	require.NoError(t, err)
	oidcServer.Start()
	defer oidcServer.Close()

	p, err := newExtension(&Config{
		IssuerURL: oidcServer.URL,
		Audience:  "unit-test",
	}, zap.NewNop())
	require.NoError(t, err)

	err = p.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	// test
	err = p.Authenticate(context.Background(), map[string][]string{"authorization": {"Bearer some-token"}})

	// verify
	assert.Error(t, err)
}

func TestFailedToGetGroupsClaimFromToken(t *testing.T) {
	// prepare
	oidcServer, err := newOIDCServer()
	require.NoError(t, err)
	oidcServer.Start()
	defer oidcServer.Close()

	for _, tt := range []struct {
		casename      string
		config        *Config
		expectedError error
	}{
		{
			"groupsClaimNonExisting",
			&Config{
				IssuerURL:   oidcServer.URL,
				Audience:    "unit-test",
				GroupsClaim: "non-existing-claim",
			},
			errGroupsClaimNotFound,
		},
		{
			"usernameClaimNonExisting",
			&Config{
				IssuerURL:     oidcServer.URL,
				Audience:      "unit-test",
				UsernameClaim: "non-existing-claim",
			},
			errClaimNotFound,
		},
		{
			"usernameNotString",
			&Config{
				IssuerURL:     oidcServer.URL,
				Audience:      "unit-test",
				UsernameClaim: "some-non-string-field",
			},
			errUsernameNotString,
		},
	} {
		t.Run(tt.casename, func(t *testing.T) {
			p, err := newExtension(tt.config, zap.NewNop())
			require.NoError(t, err)

			err = p.Start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err)

			payload, _ := json.Marshal(map[string]interface{}{
				"iss":                   oidcServer.URL,
				"some-non-string-field": 123,
				"aud":                   "unit-test",
				"exp":                   time.Now().Add(time.Minute).Unix(),
			})
			token, err := oidcServer.token(payload)
			require.NoError(t, err)

			// test
			err = p.Authenticate(context.Background(), map[string][]string{"authorization": {fmt.Sprintf("Bearer %s", token)}})

			// verify
			assert.ErrorIs(t, err, tt.expectedError)
		})
	}
}

func TestSubjectFromClaims(t *testing.T) {
	// prepare
	claims := map[string]interface{}{
		"username": "jdoe",
	}

	// test
	username, err := getSubjectFromClaims(claims, "username", "")

	// verify
	assert.NoError(t, err)
	assert.Equal(t, "jdoe", username)
}

func TestSubjectFallback(t *testing.T) {
	// prepare
	claims := map[string]interface{}{
		"sub": "jdoe",
	}

	// test
	username, err := getSubjectFromClaims(claims, "", "jdoe")

	// verify
	assert.NoError(t, err)
	assert.Equal(t, "jdoe", username)
}

func TestGroupsFromClaim(t *testing.T) {
	// prepare
	for _, tt := range []struct {
		casename string
		input    interface{}
		expected []string
	}{
		{
			"single-string",
			"department-1",
			[]string{"department-1"},
		},
		{
			"multiple-strings",
			[]string{"department-1", "department-2"},
			[]string{"department-1", "department-2"},
		},
		{
			"multiple-things",
			[]interface{}{"department-1", 123},
			[]string{"department-1", "123"},
		},
	} {
		t.Run(tt.casename, func(t *testing.T) {
			claims := map[string]interface{}{
				"sub":         "jdoe",
				"memberships": tt.input,
			}

			// test
			groups, err := getGroupsFromClaims(claims, "memberships")
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, groups)
		})
	}
}

func TestEmptyGroupsClaim(t *testing.T) {
	// prepare
	claims := map[string]interface{}{
		"sub": "jdoe",
	}

	// test
	groups, err := getGroupsFromClaims(claims, "")
	assert.NoError(t, err)
	assert.Equal(t, []string{}, groups)
}

func TestMissingClient(t *testing.T) {
	// prepare
	config := &Config{
		IssuerURL: "http://example.com/",
	}

	// test
	p, err := newExtension(config, zap.NewNop())

	// verify
	assert.Nil(t, p)
	assert.Equal(t, errNoAudienceProvided, err)
}

func TestMissingIssuerURL(t *testing.T) {
	// prepare
	config := &Config{
		Audience: "some-audience",
	}

	// test
	p, err := newExtension(config, zap.NewNop())

	// verify
	assert.Nil(t, p)
	assert.Equal(t, errNoIssuerURL, err)
}

func TestShutdown(t *testing.T) {
	// prepare
	config := &Config{
		Audience:  "some-audience",
		IssuerURL: "http://example.com/",
	}
	p, err := newExtension(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, p)

	// test
	err = p.Shutdown(context.Background()) // for now, we never fail

	// verify
	assert.NoError(t, err)
}

func TestUnaryInterceptor(t *testing.T) {
	// prepare
	config := &Config{
		Audience:  "some-audience",
		IssuerURL: "http://example.com/",
	}
	p, err := newExtension(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, p)

	interceptorCalled := false
	p.unaryInterceptor = func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler, authenticate configauth.AuthenticateFunc) (interface{}, error) {
		interceptorCalled = true
		return nil, nil
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	// test
	res, err := p.GRPCUnaryServerInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)

	// verify
	assert.NoError(t, err)
	assert.Nil(t, res)
	assert.True(t, interceptorCalled)
}

func TestStreamInterceptor(t *testing.T) {
	// prepare
	config := &Config{
		Audience:  "some-audience",
		IssuerURL: "http://example.com/",
	}
	p, err := newExtension(config, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, p)

	interceptorCalled := false
	p.streamInterceptor = func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler, authenticate configauth.AuthenticateFunc) error {
		interceptorCalled = true
		return nil
	}
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}
	streamServer := &mockServerStream{
		ctx: context.Background(),
	}

	// test
	err = p.GRPCStreamServerInterceptor(nil, streamServer, &grpc.StreamServerInfo{}, handler)

	// verify
	assert.NoError(t, err)
	assert.True(t, interceptorCalled)
}

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}
