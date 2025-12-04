package avitoquotaprocessor

import (
	"time"

	"go.avito.ru/gl/json-http-protocol/v3/client"
	"go.avito.ru/gl/transport-http/v2"
	"go.avito.ru/gl/transport-http/v2/zstd"
)

// rpcClient create RPC client
func rpcClient(destination string, cfg *ClientTls) *client.Client {
	opts := []transport.Option{
		transport.WithTimeout(2 * time.Second),
		transport.WithMW(zstd.ClientMW()),
		transport.WithDestinationServiceOptions(destination),
	}

	if cfg != nil && cfg.Enabled {
		opts = append(opts, transport.WithTLSCertificateFiles(cfg.CrtPath, cfg.KeyPath))
	}

	return client.New(
		destination,
		client.WithTransportCreator(transport.New(opts...)),
	)
}
