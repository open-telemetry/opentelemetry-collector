// Code generated - DO NOT EDIT.
package paas_observability

import (
	"context"

	rpcprotocol "go.avito.ru/gl/rpc-protocol"
)

const Name = "paas-observability"

type Server struct{}

func (Server) Name__() string                       { return Name }
func (Server) Client__(c rpcprotocol.Client) Client { return New(c) }

const (
	RPC_ListLogsRules = "listLogsRules"
)

type Client interface {
	ListLogsRules(context.Context, *ListLogsRulesIn) (*ListLogsRulesOut, error)
}

type client struct {
	listLogsRules func(ctx context.Context, in interface{}) (out interface{}, err error)
}

func New(c rpcprotocol.Client) Client {
	return &client{
		listLogsRules: c.Method("listLogsRules", true, (*ListLogsRulesOut)(nil), nil, nil),
	}
}

func (c *client) ListLogsRules(ctx context.Context, in *ListLogsRulesIn) (*ListLogsRulesOut, error) {
	o, err := c.listLogsRules(ctx, in)
	out, _ := o.(*ListLogsRulesOut)
	return out, err
}
