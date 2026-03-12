package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLimiter_Allow_underLimit(t *testing.T) {
	l := NewLimiter(3, 10*time.Second)
	ip := "192.168.1.1"
	for i := 0; i < 3; i++ {
		if !l.Allow(ip) {
			t.Errorf("request %d: expected Allow=true", i+1)
		}
	}
}

func TestLimiter_Allow_overLimit(t *testing.T) {
	l := NewLimiter(2, 10*time.Second)
	ip := "10.0.0.1"
	l.Allow(ip)
	l.Allow(ip)
	if l.Allow(ip) {
		t.Error("third request: expected Allow=false")
	}
}

func TestLimiter_Allow_emptyIP(t *testing.T) {
	l := NewLimiter(0, time.Minute)
	// Empty IP is always allowed (e.g. no RemoteAddr in tests)
	if !l.Allow("") {
		t.Error("Allow(\"\"): expected true")
	}
}

func TestClientIP_xForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18")
	req.RemoteAddr = "127.0.0.1:12345"
	got := ClientIP(req)
	if got != "203.0.113.195" {
		t.Errorf("ClientIP (X-Forwarded-For): got %q, want 203.0.113.195", got)
	}
}

func TestClientIP_xRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "198.51.100.1")
	req.RemoteAddr = "127.0.0.1:12345"
	got := ClientIP(req)
	if got != "198.51.100.1" {
		t.Errorf("ClientIP (X-Real-IP): got %q, want 198.51.100.1", got)
	}
}

func TestClientIP_remoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.100:45678"
	got := ClientIP(req)
	if got != "192.168.1.100" {
		t.Errorf("ClientIP (RemoteAddr): got %q, want 192.168.1.100", got)
	}
}
