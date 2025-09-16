package validator

import (
	"fmt"
	"net"

	"github.com/aredoff/proxygun/internal/proxy"
)

func (v *Validator) checkTCPConnectivity(p *proxy.Proxy) bool {
	address := net.JoinHostPort(p.Host, fmt.Sprintf("%d", p.Port))

	conn, err := net.DialTimeout("tcp", address, v.tcpTimeout)
	if err != nil {
		return false
	}
	defer conn.Close()

	return true
}
