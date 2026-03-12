package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidToken(t *testing.T) {
	valid64 := "a0b1c2d3e4f5678901234567890abcdef1234567890abcdef1234567890abcde" // 64 hex chars
	tests := []struct {
		token string
		want  bool
	}{
		{valid64, true},
		{"0000000000000000000000000000000000000000000000000000000000000000", true},
		{"", false},
		{"abc", false},
		{"G0b1c2d3e4f5678901234567890abcdef1234567890abcdef1234567890abcdef", false}, // G not hex
		{valid64 + "1", false}, // too long
	}
	for _, tt := range tests {
		got := ValidToken(tt.token)
		if got != tt.want {
			t.Errorf("ValidToken(%q) = %v, want %v", tt.token, got, tt.want)
		}
	}
}

func TestVerifyCSRF_match(t *testing.T) {
	token := "a0b1c2d3e4f5678901234567890abcdef1234567890abcdef1234567890abcde"
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: token})
	if !VerifyCSRF(req, token) {
		t.Error("VerifyCSRF: expected true when cookie and header match")
	}
}

func TestVerifyCSRF_mismatch(t *testing.T) {
	cookieToken := "a0b1c2d3e4f5678901234567890abcdef1234567890abcdef1234567890abcde"
	headerToken := "b0b1c2d3e4f5678901234567890abcdef1234567890abcdef1234567890abcde"
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: cookieToken})
	if VerifyCSRF(req, headerToken) {
		t.Error("VerifyCSRF: expected false when cookie and header differ")
	}
}

func TestVerifyCSRF_noCookie(t *testing.T) {
	token := "a0b1c2d3e4f5678901234567890abcdef1234567890abcdef1234567890abcde"
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	if VerifyCSRF(req, token) {
		t.Error("VerifyCSRF: expected false when cookie missing")
	}
}
