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

func jsonError(status int, code, message string) *pb.PluginHTTPResponse {
	return jsonResponse(status, map[string]string{
		"code":    code,
		"message": message,
	})
}
