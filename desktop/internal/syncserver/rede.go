package syncserver

import (
	"net"
	"net/netip"
)

// IPsLocaisPrivados lista os IPv4 privados (RFC 1918) das interfaces ativas.
func IPsLocaisPrivados() []net.IP {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var ips []net.IP
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok {
				if ip4 := n.IP.To4(); ip4 != nil && ip4.IsPrivate() {
					ips = append(ips, ip4)
				}
			}
		}
	}
	return ips
}

// EnderecoPermitido aceita apenas clientes de rede local: privados (RFC 1918),
// link-local e loopback. Qualquer outro é recusado.
func EnderecoPermitido(remoteAddr string) bool {
	ap, err := netip.ParseAddrPort(remoteAddr)
	if err != nil {
		return false
	}
	ip := ap.Addr().Unmap()
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()
}
