package webhook

import (
	"context"
	"net"
	"strings"
	"testing"
)

type resolverStub map[string][]net.IP

func (r resolverStub) LookupIP(_ context.Context, _, host string) ([]net.IP, error) {
	return r[host], nil
}

func TestValidateEndpointRejectsSSRFAndAllowsExplicitLoopbackFixture(t *testing.T) {
	resolver := resolverStub{
		"public.example":     {net.ParseIP("203.0.113.10")},
		"private.example":    {net.ParseIP("10.0.0.1")},
		"link-local.example": {net.ParseIP("169.254.169.254")},
		"local.test":         {net.ParseIP("127.0.0.1")},
	}
	for _, rawURL := range []string{
		"http://public.example/hook", "https://private.example/hook", "https://user:pass@public.example/hook", "https://public.example/hook#fragment",
	} {
		if _, err := ValidateEndpoint(context.Background(), rawURL, false, resolver); err == nil {
			t.Fatalf("accepted unsafe URL %q", rawURL)
		}
	}
	if _, err := ValidateEndpoint(context.Background(), "https://public.example/hook", false, resolver); err != nil {
		t.Fatalf("public HTTPS rejected: %v", err)
	}
	if _, err := ValidateEndpoint(context.Background(), "http://local.test/hook", true, resolver); err != nil {
		t.Fatalf("explicit loopback fixture rejected: %v", err)
	}
	for _, rawURL := range []string{"http://private.example/hook", "http://link-local.example/hook"} {
		if _, err := ValidateEndpoint(context.Background(), rawURL, true, resolver); err == nil {
			t.Fatalf("development loopback exception accepted non-loopback URL %q", rawURL)
		}
	}
}

func TestSignatureBindsTimestampAndBody(t *testing.T) {
	secret := []byte("test signing secret")
	body := []byte(`{"type":"tenant.created"}`)
	signature := Signature(secret, "1783780000", body)
	if !strings.HasPrefix(signature, "v1=") || !VerifySignature(secret, "1783780000", body, signature) {
		t.Fatalf("signature rejected: %s", signature)
	}
	if VerifySignature(secret, "1783780001", body, signature) || VerifySignature(secret, "1783780000", append(body, ' '), signature) {
		t.Fatal("signature did not bind timestamp and body")
	}
}
