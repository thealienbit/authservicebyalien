package http

import (
	"net/http/httptest"
	"testing"
)

func TestClientIPIgnoresForwardedWhenNoTrustedProxies(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "203.0.113.5:1234"
	r.Header.Set("X-Forwarded-For", "1.2.3.4")

	if got := ClientIP(r, nil); got != "203.0.113.5" {
		t.Fatalf("got %q, want %q", got, "203.0.113.5")
	}
}

func TestClientIPUsesForwardedFromTrustedProxy(t *testing.T) {
	trusted, err := ParseTrustedProxies("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.2:1234"
	r.Header.Set("X-Forwarded-For", "198.51.100.7")

	if got := ClientIP(r, trusted); got != "198.51.100.7" {
		t.Fatalf("got %q, want %q", got, "198.51.100.7")
	}
}

func TestClientIPSkipsTrustedHopsRightToLeft(t *testing.T) {
	trusted, err := ParseTrustedProxies("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.2:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.5, 10.0.0.6")

	if got := ClientIP(r, trusted); got != "203.0.113.9" {
		t.Fatalf("got %q, want %q", got, "203.0.113.9")
	}
}

func TestClientIPIgnoresSpoofedLeftmost(t *testing.T) {
	trusted, err := ParseTrustedProxies("10.0.0.2/32")
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.2:1234"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 198.51.100.20")

	if got := ClientIP(r, trusted); got != "198.51.100.20" {
		t.Fatalf("got %q, want %q", got, "198.51.100.20")
	}
}

func TestParseTrustedProxies(t *testing.T) {
	networks, err := ParseTrustedProxies("10.0.0.0/8, 192.168.1.1, ")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(networks) != 2 {
		t.Fatalf("expected 2 networks, got %d", len(networks))
	}

	if _, err := ParseTrustedProxies("not-an-ip"); err == nil {
		t.Fatal("expected an error for an invalid proxy")
	}
}
