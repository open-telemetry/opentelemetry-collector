// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package remotetapextension // import "go.opentelemetry.io/collector/extension/remotetapextension"

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/pdata"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

//go:embed html/*
var httpFS embed.FS

type remoteTapExtension struct {
	config   *Config
	settings extension.Settings
	server   *http.Server

	callbackManager *CallbackManager
	publisher       *Publisher
}

var _ pdata.Publisher = &remoteTapExtension{}

func (s *remoteTapExtension) Start(ctx context.Context, host component.Host) error {
	htmlContent, err := fs.Sub(httpFS, "html")
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(htmlContent)))
	mux.HandleFunc("/tap/", s.tapComponent)
	mux.HandleFunc("/components/", s.listComponents)

	ln, err := s.config.ServerConfig.ToListener(ctx)
	if err != nil {
		return fmt.Errorf("failed to bind to address %s: %w", s.config.Endpoint, err)
	}

	s.server, err = s.config.ServerConfig.ToServer(ctx, host, s.settings.TelemetrySettings, mux)
	if err != nil {
		return err
	}

	go func() {
		if errHTTP := s.server.Serve(ln); !errors.Is(errHTTP, http.ErrServerClosed) && errHTTP != nil {
			//s.settings.TelemetrySettings.ReportStatus(component.NewFatalErrorEvent(err))
		}
	}()
	return nil
}

func (s *remoteTapExtension) tapComponent(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/tap/")
	if id == "" {
		http.Error(w, "Missing component ID", http.StatusBadRequest)
		return
	}

	// Buffer of 1000 entries to handle load spikes and prevent this functionality from eating up too much memory.
	dataCh := make(chan string, 1000)
	ctx := r.Context()
	componentID := pdata.ComponentID(id)

	callbackID := CallbackID(uuid.New().String())

	err := s.callbackManager.Add(componentID, callbackID, func(data string) {
		select {
		case <-ctx.Done():
			return
		default:
			// Avoid blocking the channel when the channel is full
			select {
			case dataCh <- data:
			default:
			}
		}
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer func() {
		close(dataCh)
		s.callbackManager.Delete(componentID, callbackID)
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		select {
		case data := <-dataCh:
			jsonData, jsonErr := json.Marshal(data)
			if jsonErr != nil {
				http.Error(w, jsonErr.Error(), http.StatusInternalServerError)
				return
			}

			_, writeErr := w.Write([]byte("data: " + string(jsonData) + "\n\n"))
			if writeErr != nil {
				return
			}
			w.(http.Flusher).Flush()
		case <-ctx.Done():
			return
		}
	}
}

func (s *remoteTapExtension) listComponents(w http.ResponseWriter, r *http.Request) {
	jsonData, err := json.Marshal(s.callbackManager.GetRegisteredComponents())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// IsActive returns true when at least one connection is open for the given componentID.
func (s *remoteTapExtension) IsActive(componentID pdata.ComponentID) bool {
	return s.callbackManager.IsActive(componentID)
}

// PublishMetrics sends metrics for a given componentID.
func (s *remoteTapExtension) PublishMetrics(componentID pdata.ComponentID, md pmetric.Metrics) {
	s.publisher.PublishMetrics(componentID, md)
}

// PublishTraces sends traces for a given componentID.
func (s *remoteTapExtension) PublishTraces(componentID pdata.ComponentID, td ptrace.Traces) {
	s.publisher.PublishTraces(componentID, td)
}

// PublishLogs sends logs for a given componentID.
func (s *remoteTapExtension) PublishLogs(componentID pdata.ComponentID, ld plog.Logs) {
	s.publisher.PublishLogs(componentID, ld)
}

func (s *remoteTapExtension) Shutdown(_ context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Close()
}
