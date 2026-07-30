// Package response provides the only JSON response encoding helpers used by HTTP handlers.
package response

import (
	"encoding/json"
	"net/http"
)

// JSON writes a JSON response with headers committed in the correct order.
func JSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
