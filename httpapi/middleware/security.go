package middleware

import "net/http"

// SecurityHeaders applies defensive defaults suitable for a JSON API.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("Content-Security-Policy", "default-src 'none'")
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
		next.ServeHTTP(writer, request)
	})
}
