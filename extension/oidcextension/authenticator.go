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

package oidcextension

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
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configauth"

	"github.com/coreos/go-oidc"
	"google.golang.org/grpc"
)

type oidcAuthenticator struct {
	attribute string
	config    Config
	provider  *oidc.Provider
	verifier  *oidc.IDTokenVerifier

	unaryInterceptor  configauth.UnaryInterceptorFunc
	streamInterceptor configauth.StreamInterceptorFunc
	host              component.Host
}

var (
	errNoClientIDProvided                = errors.New("no ClientID provided for the Config configuration")
	errNoIssuerURL                       = errors.New("no IssuerURL provided for the Config configuration")
	errInvalidAuthenticationHeaderFormat = errors.New("invalid authorization header format")
	errFailedToObtainClaimsFromToken     = errors.New("failed to get the subject from the token issued by the Config provider")
	errClaimNotFound                     = errors.New("username claim from the Config configuration not found on the token returned by the Config provider")
	errUsernameNotString                 = errors.New("the username returned by the Config provider isn't a regular string")
	errGroupsClaimNotFound               = errors.New("groups claim from the Config configuration not found on the token returned by the Config provider")
	errNotAuthenticated                  = errors.New("authentication didn't succeed")
	defaultAttribute                     = "authorization"
)

func newOIDCAuthenticator(cfg *Config) (*oidcAuthenticator, error) {
	if cfg.Audience == "" {
		return nil, errNoClientIDProvided
	}
	if cfg.IssuerURL == "" {
		return nil, errNoIssuerURL
	}

	if cfg.Attribute == "" {
		cfg.Attribute = defaultAttribute
	}

	return &oidcAuthenticator{
		attribute:         cfg.Attribute,
		config:            *cfg,
		unaryInterceptor:  configauth.DefaultUnaryInterceptor,
		streamInterceptor: configauth.DefaultStreamInterceptor,
	}, nil
}

func (o *oidcAuthenticator) Authenticate(ctx context.Context, headers map[string][]string) (context.Context, error) {
	authHeaders := headers[o.attribute]
	if len(authHeaders) == 0 {
		return ctx, errNotAuthenticated
	}

	// we only use the first header, if multiple values exist
	parts := strings.Split(authHeaders[0], " ")
	if len(parts) != 2 {
		return ctx, errInvalidAuthenticationHeaderFormat
	}

	idToken, err := o.verifier.Verify(ctx, parts[1])
	if err != nil {
		return ctx, fmt.Errorf("failed to verify token: %w", err)
	}

	claims := map[string]interface{}{}
	if err = idToken.Claims(&claims); err != nil {
		// currently, this isn't a valid condition, the Verify call a few lines above
		// will already attempt to parse the payload as a json and set it as the claims
		// for the token. As we are using a map to hold the claims, there's no way to fail
		// to read the claims. It could fail if we were using a custom struct. Instead of
		// swalling the error, it's better to make this future-proof, in case the underlying
		// code changes
		return ctx, errFailedToObtainClaimsFromToken
	}

	sub, err := getSubjectFromClaims(claims, o.config.UsernameClaim, idToken.Subject)
	if err != nil {
		return ctx, fmt.Errorf("failed to get subject from claims in the token: %w", err)
	}
	ctx = context.WithValue(ctx, subjectKey, sub)

	gr, err := getGroupsFromClaims(claims, o.config.GroupsClaim)
	if err != nil {
		return ctx, fmt.Errorf("failed to get groups from claims in the token: %w", err)
	}
	ctx = context.WithValue(ctx, groupsKey, gr)

	return ctx, nil
}

func (o *oidcAuthenticator) Start(ctx context.Context, host component.Host) error {
	provider, err := getProviderForConfig(o.config)
	if err != nil {
		return fmt.Errorf("failed to get configuration from the auth server: %w", err)
	}
	o.provider = provider

	o.verifier = o.provider.Verifier(&oidc.Config{
		ClientID: o.config.Audience,
	})

	o.host = host

	return nil
}

func (o *oidcAuthenticator) Shutdown(ctx context.Context) error {
	// no-op at the moment
	// once we implement caching of the tokens we might need this
	return nil
}

func (o *oidcAuthenticator) UnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return o.unaryInterceptor(ctx, req, info, handler, o.Authenticate)
}

func (o *oidcAuthenticator) StreamInterceptor(srv interface{}, str grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return o.streamInterceptor(srv, str, info, handler, o.Authenticate)
}

// ToServerOptions builds a set of server options ready to be used by the gRPC server
func (o *oidcAuthenticator) ToServerOptions() ([]grpc.ServerOption, error) {
	// perhaps we should use a timeout here?
	// TODO: we need a hook to call auth.Close()
	if err := o.Start(context.Background(), o.host); err != nil {
		return nil, err
	}

	return []grpc.ServerOption{
		grpc.UnaryInterceptor(o.UnaryInterceptor),
		grpc.StreamInterceptor(o.StreamInterceptor),
	}, nil
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

func getProviderForConfig(config Config) (*oidc.Provider, error) {
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

	rawCA, err := ioutil.ReadFile(path)
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
