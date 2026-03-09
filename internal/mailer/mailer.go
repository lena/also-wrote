package mailer

import (
	"log"
	"net/smtp"
	"os"
	"strings"
)

// SendMagicLink sends the magic link to email. If SMTP is not configured, it logs the link (for local dev).
func SendMagicLink(toEmail, link string) error {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		log.Printf("[Mailer] Magic link for %s: %s", toEmail, link)
		return nil
	}
	from := os.Getenv("MAILER_FROM")
	if from == "" {
		from = "noreply@localhost"
	}
	subject := "Sign in to Also Wrote"
	body := "Click to sign in:\n\n" + link + "\n\nThis link expires in 15 minutes."
	msg := "From: " + from + "\r\n" +
		"To: " + toEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" + body

	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	addr := host + ":" + port
	auth := smtp.PlainAuth("", os.Getenv("SMTP_USER"), os.Getenv("SMTP_PASS"), host)
	return smtp.SendMail(addr, auth, from, []string{toEmail}, []byte(msg))
}

// MagicLinkURL builds the full verify URL given the app base URL and token param.
func MagicLinkURL(baseURL, tokenParam string) string {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return baseURL + "/auth/verify?token=" + tokenParam
}
