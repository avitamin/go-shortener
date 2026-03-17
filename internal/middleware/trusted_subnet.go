package middleware

import (
	"net"
	"net/http"
	"strings"
)

// RequireTrustedSubnet ограничивает доступ к endpoint'ам запросами из доверенной подсети.
// Ожидает IP клиента в заголовке X-Real-IP.
func RequireTrustedSubnet(trustedSubnet string) func(next http.Handler) http.Handler {
	var subnet *net.IPNet
	if trustedSubnet != "" {
		_, parsedSubnet, err := net.ParseCIDR(trustedSubnet)
		if err == nil {
			subnet = parsedSubnet
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if subnet == nil {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			clientIP := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP")))
			if clientIP == nil || !subnet.Contains(clientIP) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
