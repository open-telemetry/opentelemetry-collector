package processinfo

import (
	"context"
	"fmt"
	"github.com/shirou/gopsutil/process"
	"go.opentelemetry.io/collector/myip"
	"go.uber.org/zap"
	"net"
	"strconv"
	"strings"

	netutil "github.com/shirou/gopsutil/v3/net"
)

type ClientInfo struct {
	Cc  bool // 是否集中式采集
	Ip  string
	Pid int32
	Cmd string
}

const (
	processInfo string = "EASYOPS_PROCESS_INFO"
)

func SetClientInfo(ctx context.Context, cc bool, ip string, pid int32, cmd string) context.Context {
	return context.WithValue(ctx, processInfo, ClientInfo{
		Cc:  cc,
		Ip:  ip,
		Pid: pid,
		Cmd: cmd,
	})
}

func GetClientInfo(ctx context.Context) (ClientInfo, error) {
	if c, ok := ctx.Value(processInfo).(ClientInfo); !ok {
		return ClientInfo{}, fmt.Errorf("can not find clientInfo")
	} else {
		return c, nil
	}
}

func handleAddr(ctx context.Context, netAddr net.Addr, logger *zap.Logger) context.Context {
	addr := netAddr.String()
	l := strings.LastIndexByte(addr, ':')
	if l == -1 {
		logger.Error(fmt.Sprintf("can not get ip: %s", addr))
		context.WithCancel(ctx)
		return ctx
	}
	ip := addr[:l]
	portStr := addr[l+1:]
	port, err := strconv.ParseUint(portStr, 0, 64)
	if err != nil {
		logger.Error(fmt.Sprintf("parse port to int fail:%s;port %s", err, portStr))
		context.WithCancel(ctx)
		return ctx
	}
	if isLocalIp(ip) {
		cmd, pid, err := getCommandCmd(ctx, uint32(port))
		if err != nil {
			logger.Error(fmt.Sprintf("get command cmd fail:%s;port %s", err, portStr))
			context.WithCancel(ctx)
			return ctx
		}
		newCtx := SetClientInfo(ctx, true, ip, pid, cmd)
		return newCtx
	}
	return SetClientInfo(ctx, false, "", 0, "")
}

func isLocalIp(remoteIp string) bool {
	ips := myip.Get()
	for _, ip := range ips {
		if ip == remoteIp {
			return true
		}
	}
	return false
}

func getCommandCmd(ctx context.Context, port uint32) (cmd string, id int32, err error) {
	conns, err := netutil.ConnectionsPidWithContext(ctx, "tcp4", 0)
	if err != nil {
		return
	}
	var p *process.Process
	for _, conn := range conns {
		if conn.Raddr.Port == port {
			p, err = process.NewProcess(conn.Pid)
			if err != nil {
				return
			}
			cmd, err = p.Cmdline()
			if err != nil {
				return
			}
			id = conn.Pid
			return
		}
	}
	err = fmt.Errorf("can not find the prpcess;port %d", port)
	return
}
