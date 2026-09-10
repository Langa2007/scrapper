package crawler

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

func validatePublicURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("URL must be an absolute http(s) URL")
	}
	if u.User != nil {
		return nil, fmt.Errorf("URLs with embedded credentials are not allowed")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && !isPublicIP(ip) {
		return nil, fmt.Errorf("private or reserved network targets are not allowed")
	}
	return u, nil
}

func isPublicIP(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	if addr.Is4() {
		v := addr.As4()
		if v[0] == 0 || v[0] == 127 || (v[0] == 100 && v[1] >= 64 && v[1] <= 127) || (v[0] == 169 && v[1] == 254) || (v[0] == 192 && v[1] == 0) || (v[0] == 198 && (v[1] == 18 || v[1] == 19)) || v[0] >= 224 {
			return false
		}
	}
	return true
}

func safeDialAddress(ctx context.Context, address string) (string, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return "", err
	}
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return "", fmt.Errorf("blocked private or reserved network target")
		}
		return address, nil
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", strings.Trim(host, "[]"))
	if err != nil {
		return "", err
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("host did not resolve")
	}
	for _, ip := range ips {
		if !isPublicIP(net.IP(ip.AsSlice())) {
			return "", fmt.Errorf("host resolves to a private or reserved address")
		}
	}
	_, port, _ := net.SplitHostPort(address)
	return net.JoinHostPort(ips[0].String(), port), nil
}
