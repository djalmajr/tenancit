package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Resolver interface {
	LookupIP(context.Context, string, string) ([]net.IP, error)
}

type ResolvedEndpoint struct {
	URL       *url.URL
	Addresses []net.IP
}

func ValidateEndpoint(ctx context.Context, rawURL string, allowLoopbackHTTP bool, resolver Resolver) (*url.URL, error) {
	resolved, err := ResolveEndpoint(ctx, rawURL, allowLoopbackHTTP, resolver)
	if err != nil {
		return nil, err
	}
	return resolved.URL, nil
}

func ResolveEndpoint(ctx context.Context, rawURL string, allowLoopbackHTTP bool, resolver Resolver) (ResolvedEndpoint, error) {
	endpoint, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.Fragment != "" {
		return ResolvedEndpoint{}, errors.New("invalid webhook URL")
	}
	if endpoint.Scheme != "https" && !(allowLoopbackHTTP && endpoint.Scheme == "http") {
		return ResolvedEndpoint{}, errors.New("webhook URL must use HTTPS")
	}
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	addresses, err := resolver.LookupIP(ctx, "ip", endpoint.Hostname())
	if err != nil || len(addresses) == 0 {
		return ResolvedEndpoint{}, errors.New("webhook hostname cannot be resolved")
	}
	for _, address := range addresses {
		if allowLoopbackHTTP && endpoint.Scheme == "http" && address.IsLoopback() {
			continue
		}
		if !isPublicAddress(address) {
			return ResolvedEndpoint{}, fmt.Errorf("webhook hostname resolves to a non-public address")
		}
	}
	return ResolvedEndpoint{URL: endpoint, Addresses: addresses}, nil
}

func ClientFor(resolved ResolvedEndpoint, timeout time.Duration) *http.Client {
	port := resolved.URL.Port()
	if port == "" {
		if resolved.URL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	addresses := append([]net.IP(nil), resolved.Addresses...)
	transport := &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
		var last error
		for _, address := range addresses {
			connection, err := (&net.Dialer{Timeout: timeout}).DialContext(ctx, network, net.JoinHostPort(address.String(), port))
			if err == nil {
				return connection, nil
			}
			last = err
		}
		return nil, last
	}}
	return &http.Client{Transport: transport, Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func isPublicAddress(address net.IP) bool {
	return address != nil && !address.IsUnspecified() && !address.IsLoopback() &&
		!address.IsPrivate() && !address.IsLinkLocalUnicast() && !address.IsLinkLocalMulticast() &&
		!address.IsMulticast()
}

func Signature(secret []byte, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}

func VerifySignature(secret []byte, timestamp string, body []byte, presented string) bool {
	expected := Signature(secret, timestamp, body)
	return hmac.Equal([]byte(expected), []byte(presented))
}
