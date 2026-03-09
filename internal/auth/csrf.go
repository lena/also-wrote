package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"
)

const (
	CSRFCookieName = "csrf_token"
	csrfTokenLen   = 32 // 32 bytes = 64 hex chars
)

var validTokenRE = regexp.MustCompile(`^[a-f0-9]{64}$`)

// GenerateToken returns a new random CSRF token (64 hex chars).
func GenerateToken() (string, error) {
	b := make([]byte, csrfTokenLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SetCookie sets the CSRF token in a cookie (SameSite=Strict so it is not sent on cross-site requests).
func SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   24 * 3600, // 24 hours
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   isProduction(),
	})
}

// TokenFromRequest returns the CSRF token from the request cookie, or "" if missing/invalid.
func TokenFromRequest(r *http.Request) string {
	c, err := r.Cookie(CSRFCookieName)
	if err != nil || c.Value == "" {
		return ""
	}
	if !validTokenRE.MatchString(c.Value) {
		return ""
	}
	return c.Value
}

// ValidToken reports whether s is a valid token format (64 hex chars).
func ValidToken(s string) bool {
	return validTokenRE.MatchString(s)
}

// Verify checks that the token from the form or header matches the cookie and is valid.
func VerifyCSRF(r *http.Request, tokenFromFormOrHeader string) bool {
	cookieToken := TokenFromRequest(r)
	if cookieToken == "" || !ValidToken(tokenFromFormOrHeader) {
		return false
	}
	return cookieToken == tokenFromFormOrHeader
}
