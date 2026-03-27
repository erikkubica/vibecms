package scripting

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/d5/tengo/v2"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// fetchModule returns the cms/fetch built-in module for outbound HTTP requests.
func (e *ScriptEngine) fetchModule() map[string]tengo.Object {
	return map[string]tengo.Object{
		"get":    &tengo.UserFunction{Name: "get", Value: e.fetchRequest("GET")},
		"post":   &tengo.UserFunction{Name: "post", Value: e.fetchRequest("POST")},
		"put":    &tengo.UserFunction{Name: "put", Value: e.fetchRequest("PUT")},
		"patch":  &tengo.UserFunction{Name: "patch", Value: e.fetchRequest("PATCH")},
		"delete": &tengo.UserFunction{Name: "delete", Value: e.fetchRequest("DELETE")},
	}
}

// fetchRequest returns a Tengo function that makes an HTTP request with the given method.
// Usage: fetch.post(url, {headers: {}, body: ""})
// Returns: {status_code: int, body: string, error: string}
func (e *ScriptEngine) fetchRequest(method string) tengo.CallableFunc {
	return func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 1 {
			return nil, tengo.ErrWrongNumArguments
		}

		// Arg 0: URL (string)
		url, ok := tengo.ToString(args[0])
		if !ok {
			return nil, tengo.ErrInvalidArgumentType{Name: "url", Expected: "string", Found: args[0].TypeName()}
		}

		// Arg 1: Options (map, optional)
		var body string
		headers := make(map[string]string)

		if len(args) > 1 {
			opts, ok := args[1].(*tengo.Map)
			if ok {
				// Extract headers
				if h, exists := opts.Value["headers"]; exists {
					if hMap, ok := h.(*tengo.Map); ok {
						for k, v := range hMap.Value {
							if s, ok := tengo.ToString(v); ok {
								headers[k] = s
							}
						}
					}
				}
				// Extract body
				if b, exists := opts.Value["body"]; exists {
					if s, ok := tengo.ToString(b); ok {
						body = s
					}
				}
			}
		}

		// Build request
		var bodyReader io.Reader
		if body != "" {
			bodyReader = bytes.NewBufferString(body)
		}

		req, err := http.NewRequest(method, url, bodyReader)
		if err != nil {
			return makeHTTPResult(0, "", err.Error()), nil
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}

		// Execute
		resp, err := httpClient.Do(req)
		if err != nil {
			return makeHTTPResult(0, "", err.Error()), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
		if err != nil {
			return makeHTTPResult(resp.StatusCode, "", err.Error()), nil
		}

		return makeHTTPResult(resp.StatusCode, string(respBody), ""), nil
	}
}

// makeHTTPResult creates a Tengo map with the HTTP response.
func makeHTTPResult(statusCode int, body string, errMsg string) *tengo.Map {
	return &tengo.Map{
		Value: map[string]tengo.Object{
			"status_code": &tengo.Int{Value: int64(statusCode)},
			"body":        &tengo.String{Value: body},
			"error":       &tengo.String{Value: errMsg},
		},
	}
}
