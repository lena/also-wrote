package auth

import (
	"also-wrote/internal/db"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignCookie_VerifyCookie_roundTrip(t *testing.T) {
	secret := "test-secret-do-not-use-in-production"
	userID := int64(42)
	email := "user@example.com"

	signed := SignCookie(userID, email, secret)
	if signed == "" {
		t.Fatal("SignCookie returned empty")
	}

	gotID, gotEmail := VerifyCookie(signed, secret)
	if gotID != userID || gotEmail != email {
		t.Errorf("VerifyCookie: got id=%v email=%q, want id=%v email=%q", gotID, gotEmail, userID, email)
	}
}

func TestVerifyCookie_rejectsTampered(t *testing.T) {
	secret := "secret"
	signed := SignCookie(1, "a@b.co", secret)
	// Tamper: change a character in the base64 payload
	tampered := signed[:len(signed)-2] + "xx"

	id, email := VerifyCookie(tampered, secret)
	if id != 0 || email != "" {
		t.Errorf("VerifyCookie(tampered): got id=%v email=%q, want 0, \"\"", id, email)
	}
}

func TestVerifyCookie_wrongSecret(t *testing.T) {
	signed := SignCookie(1, "a@b.co", "secret1")
	id, email := VerifyCookie(signed, "secret2")
	if id != 0 || email != "" {
		t.Errorf("VerifyCookie(wrong secret): got id=%v email=%q, want 0, \"\"", id, email)
	}
}

func TestVerifyCookie_emptyInputs(t *testing.T) {
	tests := []struct {
		value, secret string
	}{
		{"", "x"},
		{"x", ""},
		{"", ""},
	}
	for _, tt := range tests {
		id, email := VerifyCookie(tt.value, tt.secret)
		if id != 0 || email != "" {
			t.Errorf("VerifyCookie(%q, %q): got id=%v email=%q", tt.value, tt.secret, id, email)
		}
	}
}

func TestTokenHash(t *testing.T) {
	// TokenHash should be deterministic
	in := []byte("same-input")
	h1 := TokenHash(in)
	h2 := TokenHash(in)
	if h1 != h2 {
		t.Errorf("TokenHash not deterministic: %q vs %q", h1, h2)
	}
	if len(h1) != 64 {
		t.Errorf("TokenHash (sha256 hex) len: got %d, want 64", len(h1))
	}
}

func TestSetSession_setsCookie(t *testing.T) {
	secret := "test-secret"
	user := &db.User{ID: 1, Email: "test@example.com"}
	w := httptest.NewRecorder()
	SetSession(w, user, secret)
	res := w.Result()
	cookies := res.Cookies()
	var session *http.Cookie
	for _, c := range cookies {
		if c.Name == CookieName {
			session = c
			break
		}
	}
	if session == nil {
		t.Fatal("no session cookie set")
	}
	id, email := VerifyCookie(session.Value, secret)
	if id != user.ID || email != user.Email {
		t.Errorf("cookie value: got id=%v email=%q, want id=%v email=%q", id, email, user.ID, user.Email)
	}
}
