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

package oidcauthextension

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configauth"
)

type oidcExtension struct {
	cfg               *Config
	unaryInterceptor  configauth.GRPCUnaryInterceptorFunc
	streamInterceptor configauth.GRPCStreamInterceptorFunc

	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier

	logger *zap.Logger
}

var (
	_ configauth.ServerAuthenticator = (*oidcExtension)(nil)

	errNoAudienceProvided                = errors.New("no Audience provided for the OIDC configuration")
	errNoIssuerURL                       = errors.New("no IssuerURL provided for the OIDC configuration")
	errInvalidAuthenticationHeaderFormat = errors.New("invalid authorization header format")
	errFailedToObtainClaimsFromToken     = errors.New("failed to get the subject from the token issued by the OIDC provider")
	errClaimNotFound                     = errors.New("username claim from the OIDC configuration not found on the token returned by the OIDC provider")
	errUsernameNotString                 = errors.New("the username returned by the OIDC provider isn't a regular string")
	errGroupsClaimNotFound               = errors.New("groups claim from the OIDC configuration not found on the token returned by the OIDC provider")
	errNotAuthenticated                  = errors.New("authentication didn't succeed")
)

func newExtension(cfg *Config, logger *zap.Logger) (*oidcExtension, error) {
	if cfg.Audience == "" {
		return nil, errNoAudienceProvided
	}
	if cfg.IssuerURL == "" {
		return nil, errNoIssuerURL
	}

	if cfg.Attribute == "" {
		cfg.Attribute = defaultAttribute
	}

	return &oidcExtension{
		cfg:               cfg,
		logger:            logger,
		unaryInterceptor:  configauth.DefaultGRPCUnaryServerInterceptor,
		streamInterceptor: configauth.DefaultGRPCStreamServerInterceptor,
	}, nil
}

func (e *oidcExtension) Start(ctx context.Context, _ component.Host) error {
	provider, err := getProviderForConfig(e.cfg)
	if err != nil {
		return fmt.Errorf("failed to get configuration from the auth server: %w", err)
	}
	e.provider = provider

	e.verifier = e.provider.Verifier(&oidc.Config{
		ClientID: e.cfg.Audience,
	})

	return nil
}

// Shutdown is invoked during service shutdown.
func (e *oidcExtension) Shutdown(context.Context) error {
	return nil
}

// Authenticate checks whether the given context contains valid auth data. Successfully authenticated calls will always return a nil error and a context with the auth data.
func (e *oidcExtension) Authenticate(ctx context.Context, headers map[string][]string) error {
	authHeaders := headers[e.cfg.Attribute]
	if len(authHeaders) == 0 {
		return errNotAuthenticated
	}

	// we only use the first header, if multiple values exist
	parts := strings.Split(authHeaders[0], " ")
	if len(parts) != 2 {
		return errInvalidAuthenticationHeaderFormat
	}

	idToken, err := e.verifier.Verify(ctx, parts[1])
	if err != nil {
		return fmt.Errorf("failed to verify token: %w", err)
	}

	claims := map[string]interface{}{}
	if err = idToken.Claims(&claims); err != nil {
		// currently, this isn't a valid condition, the Verify call a few lines above
		// will already attempt to parse the payload as a json and set it as the claims
		// for the token. As we are using a map to hold the claims, there's no way to fail
		// to read the claims. It could fail if we were using a custom struct. Instead of
		// swalling the error, it's better to make this future-proof, in case the underlying
		// code changes
		return errFailedToObtainClaimsFromToken
	}

	_, err = getSubjectFromClaims(claims, e.cfg.UsernameClaim, idToken.Subject)
	if err != nil {
		return fmt.Errorf("failed to get subject from claims in the token: %w", err)
	}

	_, err = getGroupsFromClaims(claims, e.cfg.GroupsClaim)
	if err != nil {
		return fmt.Errorf("failed to get groups from claims in the token: %w", err)
	}

	return nil
}

// GRPCUnaryServerInterceptor is a helper method to provide a gRPC-compatible UnaryInterceptor, typically calling the authenticator's Authenticate method.
func (e *oidcExtension) GRPCUnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return e.unaryInterceptor(ctx, req, info, handler, e.Authenticate)
}

// GRPCStreamServerInterceptor is a helper method to provide a gRPC-compatible StreamInterceptor, typically calling the authenticator's Authenticate method.
func (e *oidcExtension) GRPCStreamServerInterceptor(srv interface{}, str grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return e.streamInterceptor(srv, str, info, handler, e.Authenticate)
}

func getSubjectFromClaims(claims map[string]interface{}, usernameClaim string, fallback string) (string, error) {
	if len(usernameClaim) > 0 {
		username, found := claims[usernameClaim]
		if !found {
			return "", errClaimNotFound
		}

		sUsername, ok := username.(string)
		if !ok {
			return "", errUsernameNotString
		}

		return sUsername, nil
	}

	return fallback, nil
}

func getGroupsFromClaims(claims map[string]interface{}, groupsClaim string) ([]string, error) {
	if len(groupsClaim) > 0 {
		var groups []string
		rawGroup, ok := claims[groupsClaim]
		if !ok {
			return nil, errGroupsClaimNotFound
		}
		switch v := rawGroup.(type) {
		case string:
			groups = append(groups, v)
		case []string:
			groups = v
		case []interface{}:
			groups = make([]string, 0, len(v))
			for i := range v {
				groups = append(groups, fmt.Sprintf("%v", v[i]))
			}
		}

		return groups, nil
	}

	return []string{}, nil
}

func getProviderForConfig(config *Config) (*oidc.Provider, error) {
	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 10 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	cert, err := getIssuerCACertFromPath(config.IssuerCAPath)
	if err != nil {
		return nil, err // the errors from this path have enough context already
	}

	if cert != nil {
		t.TLSClientConfig = &tls.Config{
			RootCAs: x509.NewCertPool(),
		}
		t.TLSClientConfig.RootCAs.AddCert(cert)
	}

	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: t,
	}
	oidcContext := oidc.ClientContext(context.Background(), client)
	return oidc.NewProvider(oidcContext, config.IssuerURL)
}

func getIssuerCACertFromPath(path string) (*x509.Certificate, error) {
	if path == "" {
		return nil, nil
	}

	rawCA, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("could not read the CA file %q: %w", path, err)
	}

	if len(rawCA) == 0 {
		return nil, fmt.Errorf("could not read the CA file %q: empty file", path)
	}

	block, _ := pem.Decode(rawCA)
	if block == nil {
		return nil, fmt.Errorf("cannot decode the contents of the CA file %q: %w", path, err)
	}

	return x509.ParseCertificate(block.Bytes)
}
