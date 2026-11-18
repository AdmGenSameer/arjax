package icons

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

var reservedIPRanges []*net.IPNet

func init() {
	ranges := []string{
		"0.0.0.0/8",          // Current network (only valid as source address)
		"10.0.0.0/8",         // Private-Use
		"100.64.0.0/10",      // Shared Address Space (Carrier-grade NAT)
		"127.0.0.0/8",        // Loopback
		"169.254.0.0/16",     // Link Local & Cloud metadata
		"172.16.0.0/12",      // Private-Use
		"192.0.0.0/24",       // IETF Protocol Assignments
		"192.0.2.0/24",       // Documentation (TEST-NET-1)
		"192.88.99.0/24",     // 6to4 Relay Anycast
		"192.168.0.0/16",     // Private-Use
		"198.18.0.0/15",      // Benchmarking
		"198.51.100.0/24",    // Documentation (TEST-NET-2)
		"203.0.113.0/24",     // Documentation (TEST-NET-3)
		"224.0.0.0/4",        // Multicast
		"240.0.0.0/4",        // Reserved for Future Use
		"255.255.255.255/32", // Limited Broadcast

		// IPv6 reserved subnets
		"::/128",       // Unspecified
		"::1/128",      // Loopback
		"100::/64",     // Discard prefix
		"2001:db8::/32",// Documentation
		"fc00::/7",     // Unique Local Addresses (ULA)
		"fe80::/10",    // Link-Local Unicast
		"ff00::/8",     // Multicast
	}

	for _, cidr := range ranges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			reservedIPRanges = append(reservedIPRanges, ipNet)
		}
	}
}

// IsPrivateOrReservedIP checks if an IP is private, loopback, multicast, or cloud metadata
func IsPrivateOrReservedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	// Unwrap IPv4-mapped IPv6 addresses (e.g. ::ffff:127.0.0.1)
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}

	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}

	for _, block := range reservedIPRanges {
		if block.Contains(ip) {
			return true
		}
	}

	// Specific AWS/GCP/Azure link-local metadata addresses
	ipStr := ip.String()
	if ipStr == "169.254.169.254" || ipStr == "fd00:ec2::254" {
		return true
	}

	return false
}

// ValidateRemoteURL performs pre-flight URL verification (HTTPS only, valid host, non-private DNS resolution)
func ValidateRemoteURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL syntax: %w", err)
	}

	if strings.ToLower(u.Scheme) != "https" {
		return fmt.Errorf("insecure scheme '%s': HTTPS is strictly required for remote icon retrieval", u.Scheme)
	}

	if u.User != nil {
		return fmt.Errorf("credentials in URL are prohibited")
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty hostname")
	}

	lowHost := strings.ToLower(host)
	if lowHost == "localhost" || strings.HasSuffix(lowHost, ".localhost") ||
		strings.HasSuffix(lowHost, ".local") || strings.HasSuffix(lowHost, ".internal") ||
		strings.HasSuffix(lowHost, ".lan") {
		return fmt.Errorf("prohibited local domain: %s", host)
	}

	// Check if hostname is an IP directly
	if ip := net.ParseIP(host); ip != nil {
		if IsPrivateOrReservedIP(ip) {
			return fmt.Errorf("prohibited private/reserved IP: %s", host)
		}
		return nil
	}

	// Resolve hostname to ensure none of the IPs point to private/reserved ranges
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("DNS resolution failed for %s: %w", host, err)
	}

	if len(ips) == 0 {
		return fmt.Errorf("no IP addresses resolved for %s", host)
	}

	for _, ip := range ips {
		if IsPrivateOrReservedIP(ip) {
			return fmt.Errorf("DNS resolution for %s returned blocked IP %s (SSRF / DNS rebinding protection)", host, ip.String())
		}
	}

	return nil
}

// NewSecureHTTPClient returns an HTTP client hardened against SSRF, DNS rebinding, redirect loops, and slowloris attacks
func NewSecureHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 6 * time.Second
	}

	dialer := &net.Dialer{
		Timeout:   3 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(network, address string, c syscall.RawConn) error {
			// Extract host and port
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				host = address
			}
			ip := net.ParseIP(host)
			if ip != nil && IsPrivateOrReservedIP(ip) {
				return fmt.Errorf("dial blocked to reserved address: %s", ip.String())
			}
			return nil
		},
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			// Pre-dial DNS verification
			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, fmt.Errorf("pre-dial DNS lookup failed for %s: %w", host, err)
			}

			var dialIP net.IP
			for _, ip := range ips {
				if !IsPrivateOrReservedIP(ip) {
					dialIP = ip
					break
				}
			}

			if dialIP == nil {
				return nil, fmt.Errorf("all resolved IPs for %s are private or reserved", host)
			}

			targetAddr := net.JoinHostPort(dialIP.String(), port)
			return dialer.DialContext(ctx, network, targetAddr)
		},
		MaxIdleConns:          50,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		DisableKeepAlives:     false,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("stopped after 3 redirects")
			}
			// Validate redirect destination against SSRF
			if err := ValidateRemoteURL(req.URL.String()); err != nil {
				return fmt.Errorf("redirect blocked: %w", err)
			}
			return nil
		},
	}
}

// SafeDecompressGzip decompresses gzip bytes with strict bounds against decompression bombs
func SafeDecompressGzip(data []byte, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = MaxIconSizeBytes
	}

	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("invalid gzip stream: %w", err)
	}
	defer gr.Close()

	// Read up to maxBytes + 1 to detect overflow
	limitedReader := io.LimitReader(gr, maxBytes+1)
	decompressed, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("gzip decompression error: %w", err)
	}

	if int64(len(decompressed)) > maxBytes {
		return nil, fmt.Errorf("gzip payload exceeds safe size limit (%d bytes), decompression bomb rejected", maxBytes)
	}

	return decompressed, nil
}
