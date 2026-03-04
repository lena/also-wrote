package auth

import (
	"also-wrote/internal/db"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	CookieName   = "session"
	MaxAgeSecs   = 30 * 24 * 3600 // 30 days
	TokenExpiry  = 15 * time.Minute
	TokenBytes   = 32
)

// Secret returns the HMAC secret (must be set for secure sessions).
func Secret() string {
	return getEnv("SESSION_SECRET", "")
}

// SignCookie creates a signed value: base64(id:email:hmac(id:email)).
func SignCookie(userID int64, email, secret string) string {
	if secret == "" {
		return ""
	}
	payload := fmt.Sprintf("%d:%s", userID, email)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return base64.URLEncoding.EncodeToString([]byte(payload + ":" + sig))
}

// VerifyCookie parses and verifies the cookie; returns userID and email or 0, "".
func VerifyCookie(value, secret string) (int64, string) {
	if value == "" || secret == "" {
		return 0, ""
	}
	raw, err := base64.URLEncoding.DecodeString(value)
	if err != nil {
		return 0, ""
	}
	parts := strings.SplitN(string(raw), ":", 3)
	if len(parts) != 3 {
		return 0, ""
	}
	payload := parts[0] + ":" + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	if parts[2] != expected {
		return 0, ""
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, ""
	}
	return id, parts[1]
}

// SetSession sets the session cookie for the user.
func SetSession(w http.ResponseWriter, user *db.User, secret string) {
	v := SignCookie(user.ID, user.Email, secret)
	if v == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    v,
		Path:     "/",
		MaxAge:   MaxAgeSecs,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   getEnv("APP_ENV", "development") == "production",
	})
}

// ClearSession removes the session cookie.
func ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// TokenHash returns SHA256 hex of the token for storing in DB.
func TokenHash(token []byte) string {
	h := sha256.Sum256(token)
	return hex.EncodeToString(h[:])
}

// NewMagicLinkToken generates a cryptographically random token.
func NewMagicLinkToken() (raw []byte, hash string, err error) {
	raw = make([]byte, TokenBytes)
	if _, err = rand.Read(raw); err != nil {
		return nil, "", err
	}
	hash = TokenHash(raw)
	return raw, hash, nil
}

// RawTokenToURLParam returns base64url-encoded token for use in link.
func RawTokenToURLParam(raw []byte) string {
	return base64.URLEncoding.EncodeToString(raw)
}

// URLParamToRaw decodes the token from the URL (for verify).
func URLParamToRaw(param string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(param)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
