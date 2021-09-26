package processinfo

import (
	"context"
	"go.uber.org/zap"
	"google.golang.org/grpc/stats"
)

var _ stats.Handler = (*GrpcStatsHandler)(nil)

type GrpcStatsHandler struct {
	logger *zap.Logger
}

func NewGrpcStatsHandler(logger *zap.Logger) GrpcStatsHandler {
	return GrpcStatsHandler{
		logger: logger,
	}
}

func (g GrpcStatsHandler) TagRPC(c context.Context, _ *stats.RPCTagInfo) context.Context {
	return c
}

func (g GrpcStatsHandler) HandleRPC(context.Context, stats.RPCStats) {}

func (g GrpcStatsHandler) TagConn(ctx context.Context, c *stats.ConnTagInfo) context.Context {
	return handleAddr(ctx, c.RemoteAddr, g.logger)
}

func (g GrpcStatsHandler) HandleConn(context.Context, stats.ConnStats) {}
