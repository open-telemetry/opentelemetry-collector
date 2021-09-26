package processinfo

import (
	"context"
	"go.uber.org/zap"
	"net"
)

func Handler(ctx context.Context, n net.Conn) context.Context {
	return handleAddr(ctx, n.RemoteAddr(), zap.NewNop())
}
