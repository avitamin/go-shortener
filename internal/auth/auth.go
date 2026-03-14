package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// ErrAuthorizationMissing indicates missing authorization value.
var ErrAuthorizationMissing = errors.New("authorization metadata is missing")

// ErrAuthorizationInvalid indicates invalid authorization token.
var ErrAuthorizationInvalid = errors.New("authorization metadata is invalid")

// BuildSignedUserID signs userID and returns token in "userID|signature" format.
func BuildSignedUserID(userID, secret string) string {
	sig := ComputeHMAC(userID, secret)
	return fmt.Sprintf("%s|%s", userID, sig)
}

// VerifySignature checks if HMAC signature is valid for message.
func VerifySignature(msg, sig, secret string) bool {
	expected := ComputeHMAC(msg, secret)
	return hmac.Equal([]byte(expected), []byte(sig))
}

// ComputeHMAC returns lowercase hex-encoded HMAC-SHA256 for msg.
func ComputeHMAC(msg, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// UserIDFromAuthorization extracts and verifies signed user ID from authorization header value.
// Supports both raw token and "Bearer <token>" formats.
func UserIDFromAuthorization(value string, secret string) (string, error) {
	token := normalizeAuthorizationValue(value)
	if token == "" {
		return "", ErrAuthorizationMissing
	}

	parts := strings.SplitN(token, "|", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return "", ErrAuthorizationInvalid
	}

	userID := parts[0]
	sig := parts[1]
	if !VerifySignature(userID, sig, secret) {
		return "", ErrAuthorizationInvalid
	}

	return userID, nil
}

func normalizeAuthorizationValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	if len(trimmed) >= 7 && strings.EqualFold(trimmed[:7], "bearer ") {
		return strings.TrimSpace(trimmed[7:])
	}

	return trimmed
}
