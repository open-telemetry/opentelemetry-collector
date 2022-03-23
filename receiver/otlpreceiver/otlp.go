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

package otlpreceiver // import "go.opentelemetry.io/collector/receiver/otlpreceiver"

import (
	"compress/gzip"
	"context"
	"encoding/binary"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/otlpgrpc"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/logs"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metrics"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/trace"
)

// otlpReceiver is the type that exposes Trace and Metrics reception.
type otlpReceiver struct {
	cfg        *Config
	serverGRPC *grpc.Server
	httpMux    *mux.Router
	serverHTTP *http.Server

	traceReceiver   *trace.Receiver
	metricsReceiver *metrics.Receiver
	logReceiver     *logs.Receiver
	shutdownWG      sync.WaitGroup

	settings component.ReceiverCreateSettings
}

// newOtlpReceiver just creates the OpenTelemetry receiver services. It is the caller's
// responsibility to invoke the respective Start*Reception methods as well
// as the various Stop*Reception methods to end it.
func newOtlpReceiver(cfg *Config, settings component.ReceiverCreateSettings) *otlpReceiver {
	r := &otlpReceiver{
		cfg:      cfg,
		settings: settings,
	}
	if cfg.HTTP != nil {
		r.httpMux = mux.NewRouter()
	}

	return r
}

func (r *otlpReceiver) startGRPCServer(cfg *configgrpc.GRPCServerSettings, host component.Host) error {
	r.settings.Logger.Info("Starting GRPC server on endpoint " + cfg.NetAddr.Endpoint)

	gln, err := cfg.ToListener()
	if err != nil {
		return err
	}
	r.shutdownWG.Add(1)
	go func() {
		defer r.shutdownWG.Done()

		if errGrpc := r.serverGRPC.Serve(gln); errGrpc != nil && errGrpc != grpc.ErrServerStopped {
			host.ReportFatalError(errGrpc)
		}
	}()
	return nil
}

func (r *otlpReceiver) startHTTPServer(cfg *confighttp.HTTPServerSettings, host component.Host) error {
	r.settings.Logger.Info("Starting HTTP server on endpoint " + cfg.Endpoint)
	var hln net.Listener
	hln, err := cfg.ToListener()
	if err != nil {
		return err
	}
	r.shutdownWG.Add(1)
	go func() {
		defer r.shutdownWG.Done()

		if errHTTP := r.serverHTTP.Serve(hln); errHTTP != http.ErrServerClosed {
			host.ReportFatalError(errHTTP)
		}
	}()
	return nil
}

func (r *otlpReceiver) startProtocolServers(host component.Host) error {
	var err error
	if r.cfg.GRPC != nil {
		var opts []grpc.ServerOption
		opts, err = r.cfg.GRPC.ToServerOption(host, r.settings.TelemetrySettings)
		if err != nil {
			return err
		}
		r.serverGRPC = grpc.NewServer(opts...)

		if r.traceReceiver != nil {
			otlpgrpc.RegisterTracesServer(r.serverGRPC, r.traceReceiver)
		}

		if r.metricsReceiver != nil {
			otlpgrpc.RegisterMetricsServer(r.serverGRPC, r.metricsReceiver)
		}

		if r.logReceiver != nil {
			otlpgrpc.RegisterLogsServer(r.serverGRPC, r.logReceiver)
		}

		err = r.startGRPCServer(r.cfg.GRPC, host)
		if err != nil {
			return err
		}
	}
	if r.cfg.HTTP != nil {
		r.serverHTTP, err = r.cfg.HTTP.ToServer(
			host,
			r.settings.TelemetrySettings,
			r.httpMux,
			confighttp.WithErrorHandler(errorHandler),
		)
		if err != nil {
			return err
		}

		err = r.startHTTPServer(r.cfg.HTTP, host)
		if err != nil {
			return err
		}
	}

	return err
}

// Start runs the trace receiver on the gRPC server. Currently
// it also enables the metrics receiver too.
func (r *otlpReceiver) Start(_ context.Context, host component.Host) error {
	return r.startProtocolServers(host)
}

// Shutdown is a method to turn off receiving.
func (r *otlpReceiver) Shutdown(ctx context.Context) error {
	var err error

	if r.serverHTTP != nil {
		err = r.serverHTTP.Shutdown(ctx)
	}

	if r.serverGRPC != nil {
		r.serverGRPC.GracefulStop()
	}

	r.shutdownWG.Wait()
	return err
}

func (r *otlpReceiver) registerTraceConsumer(tc consumer.Traces) error {
	if tc == nil {
		return componenterror.ErrNilNextConsumer
	}
	r.traceReceiver = trace.New(r.cfg.ID(), tc, r.settings)
	if r.httpMux != nil {
		r.httpMux.HandleFunc("/v1/traces", func(resp http.ResponseWriter, req *http.Request) {
			handleTraces(resp, req, r.traceReceiver, pbEncoder)
		}).Methods(http.MethodPost).Headers("Content-Type", pbContentType)
		r.httpMux.HandleFunc("/v1/traces", func(resp http.ResponseWriter, req *http.Request) {
			handleTraces(resp, req, r.traceReceiver, jsEncoder)
		}).Methods(http.MethodPost).Headers("Content-Type", jsonContentType)
		r.httpMux.HandleFunc("/v1/traces", func(resp http.ResponseWriter, req *http.Request) {
			handleUnmatchedRequests(resp, req)
		})

		r.httpMux.HandleFunc(
			"/opentelemetry.proto.collector.trace.v1.TraceService/Export",
			func(resp http.ResponseWriter, req *http.Request) {
				prepareGrpcResponse(resp)
				body, err := deframeGrpc(req)
				if err != nil {
					writeGrpcError(resp, err)
					return
				}
				otlpReq, err := pbEncoder.unmarshalTracesRequest(body)
				if err != nil {
					writeGrpcError(resp, err)
					return
				}
				otlpResp, err := r.traceReceiver.Export(req.Context(), otlpReq)
				if err != nil {
					writeGrpcError(resp, err)
					return
				}
				msg, err := pbEncoder.marshalTracesResponse(otlpResp)
				if err != nil {
					writeGrpcError(resp, err)
					return
				}
				writeGrpcMessage(resp, msg)
			}).Methods(http.MethodPost)
	}
	return nil
}

func writeGrpcMessage(resp http.ResponseWriter, msg []byte) {
	_, err := resp.Write([]byte{0})
	if err != nil {
		writeGrpcError(resp, err)
	}
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(msg)))
	_, err = resp.Write(length)
	if err != nil {
		writeGrpcError(resp, err)
		return
	}
	_, err = resp.Write(msg)
	if err != nil {
		writeGrpcError(resp, err)
		return
	}
	writeGrpcStatus(resp, status.New(codes.OK, ""))
}

func (r *otlpReceiver) registerMetricsConsumer(mc consumer.Metrics) error {
	if mc == nil {
		return componenterror.ErrNilNextConsumer
	}
	r.metricsReceiver = metrics.New(r.cfg.ID(), mc, r.settings)
	if r.httpMux != nil {
		r.httpMux.HandleFunc("/v1/metrics", func(resp http.ResponseWriter, req *http.Request) {
			handleMetrics(resp, req, r.metricsReceiver, pbEncoder)
		}).Methods(http.MethodPost).Headers("Content-Type", pbContentType)
		r.httpMux.HandleFunc("/v1/metrics", func(resp http.ResponseWriter, req *http.Request) {
			handleMetrics(resp, req, r.metricsReceiver, jsEncoder)
		}).Methods(http.MethodPost).Headers("Content-Type", jsonContentType)
		r.httpMux.HandleFunc("/v1/metrics", func(resp http.ResponseWriter, req *http.Request) {
			handleUnmatchedRequests(resp, req)
		})

		r.httpMux.HandleFunc(
			"/opentelemetry.proto.collector.metrics.v1.MetricsService/Export",
			func(resp http.ResponseWriter, req *http.Request) {
				prepareGrpcResponse(resp)
				body, err := deframeGrpc(req)
				if err != nil {
					writeGrpcError(resp, err)
					return
				}
				otlpReq, err := pbEncoder.unmarshalMetricsRequest(body)
				if err != nil {
					writeGrpcError(resp, err)
					return
				}
				otlpResp, err := r.metricsReceiver.Export(req.Context(), otlpReq)
				if err != nil {
					writeGrpcError(resp, err)
					return
				}
				msg, err := pbEncoder.marshalMetricsResponse(otlpResp)
				if err != nil {
					writeGrpcError(resp, err)
					return
				}
				writeGrpcMessage(resp, msg)
			}).Methods(http.MethodPost)
	}
	return nil
}

func (r *otlpReceiver) registerLogsConsumer(lc consumer.Logs) error {
	if lc == nil {
		return componenterror.ErrNilNextConsumer
	}
	r.logReceiver = logs.New(r.cfg.ID(), lc, r.settings)
	if r.httpMux != nil {
		r.httpMux.HandleFunc("/v1/logs", func(w http.ResponseWriter, req *http.Request) {
			handleLogs(w, req, r.logReceiver, pbEncoder)
		}).Methods(http.MethodPost).Headers("Content-Type", pbContentType)
		r.httpMux.HandleFunc("/v1/logs", func(w http.ResponseWriter, req *http.Request) {
			handleLogs(w, req, r.logReceiver, jsEncoder)
		}).Methods(http.MethodPost).Headers("Content-Type", jsonContentType)
		r.httpMux.HandleFunc("/v1/logs", func(resp http.ResponseWriter, req *http.Request) {
			handleUnmatchedRequests(resp, req)
		})

		r.httpMux.HandleFunc(
			"/opentelemetry.proto.collector.logs.v1.LogsService/Export",
			func(resp http.ResponseWriter, req *http.Request) {
				prepareGrpcResponse(resp)
				body, err := deframeGrpc(req)
				if err != nil {
					writeGrpcError(resp, err)
					return
				}
				otlpReq, err := pbEncoder.unmarshalLogsRequest(body)
				if err != nil {
					writeGrpcError(resp, err)
					return
				}
				otlpResp, err := r.logReceiver.Export(req.Context(), otlpReq)
				if err != nil {
					writeGrpcError(resp, err)
					return
				}
				msg, err := pbEncoder.marshalLogsResponse(otlpResp)
				if err != nil {
					writeGrpcError(resp, err)
					return
				}
				writeGrpcMessage(resp, msg)
			}).Methods(http.MethodPost)
	}
	return nil
}

func prepareGrpcResponse(resp http.ResponseWriter) {
	resp.Header().Set("Trailer", "grpc-status, grpc-message")
	resp.Header().Set("Content-Type", "application/grpc")
	resp.WriteHeader(http.StatusOK)
}

func handleUnmatchedRequests(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		status := http.StatusMethodNotAllowed
		writeResponse(resp, "text/plain", status, []byte(fmt.Sprintf("%v method not allowed, supported: [POST]", status)))
		return
	}
	if req.Header.Get("Content-Type") == "" {
		status := http.StatusUnsupportedMediaType
		writeResponse(resp, "text/plain", status, []byte(fmt.Sprintf("%v unsupported media type, supported: [%s, %s]", status, jsonContentType, pbContentType)))
		return
	}
}

func deframeGrpc(req *http.Request) ([]byte, error) {
	defer func() {
		_ = req.Body.Close()
	}()
	header := make([]byte, 5)
	_, err := io.ReadFull(req.Body, header)
	if err != nil {
		return nil, err
	}
	flag := header[0]
	length := binary.BigEndian.Uint32(header[1:])
	if flag == 0 {
		payload, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		if uint32(len(payload)) != length {
			return nil, fmt.Errorf("unexpected payload length, expected %v got %v", length, len(payload))
		}
		return payload, nil
	}

	// Compressed body
	encoding := req.Header.Get("grpc-encoding")
	if encoding != "gzip" {
		return nil, fmt.Errorf("unsupported compression format %v", encoding)
	}

	lr := io.LimitReader(req.Body, int64(length))
	gz, err := gzip.NewReader(lr)
	if err != nil {
		return nil, err
	}
	payload, err := io.ReadAll(gz)
	if err != nil {
		return nil, err
	}
	rem, err := io.ReadAll(req.Body)
	if len(rem) != 0 {
		return nil, fmt.Errorf("unexpected payload length, expected %v got %v", length, int(length)+len(rem))
	}
	return payload, nil
}

func writeGrpcError(resp http.ResponseWriter, err error) {
	s, ok := status.FromError(err)
	if !ok {
		s = errorMsgToStatus(err.Error(), http.StatusBadRequest)
	}
	writeGrpcStatus(resp, s)
}

func writeGrpcStatus(resp http.ResponseWriter, s *status.Status) {
	resp.Header().Set("grpc-status", strconv.Itoa(int(s.Code())))
	if len(s.Message()) != 0 {
		resp.Header().Set("grpc-message", s.Message())
	}
}
