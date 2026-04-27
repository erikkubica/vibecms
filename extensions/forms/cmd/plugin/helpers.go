package main

import (
	"encoding/json"

	pb "vibecms/pkg/plugin/proto"
)

func jsonResponse(status int, data any) *pb.PluginHTTPResponse {
	b, _ := json.Marshal(data)
	return &pb.PluginHTTPResponse{
		StatusCode: int32(status),
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       b,
	}
}

// jsonError returns a {"error": "<CODE>", "message": "..."} response. The
// shape matches the inline error responses in handlers_submit.go (rate limit,
// captcha, validation) so every admin/public client can read `data.error` for
// the machine-readable code and `data.message` for the human-readable text.
func jsonError(status int, code, message string) *pb.PluginHTTPResponse {
	return jsonResponse(status, map[string]string{
		"error":   code,
		"message": message,
	})
}
