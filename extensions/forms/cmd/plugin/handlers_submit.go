package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"strings"
	"time"

	"vibecms/internal/coreapi"
	pb "vibecms/pkg/plugin/proto"
)

func (p *FormsPlugin) handleSubmit(ctx context.Context, slug string, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	// Find form by slug
	res, err := p.host.DataQuery(ctx, formsTable, coreapi.DataStoreQuery{
		Where: map[string]any{"slug": slug},
		Limit: 1,
	})
	if err != nil || res.Total == 0 {
		return jsonError(404, "FORM_NOT_FOUND", "Form not found"), nil
	}
	form := normalizeForm(res.Rows[0])
	formID, _ := toUint(form["id"])
	if formID == 0 {
		return jsonError(500, "INVALID_FORM", "Form has no usable id"), nil
	}

	// --- Rate limiting ---
	settings := getFormSettings(form)
	ip := req.GetHeaders()["X-Forwarded-For"]
	rateLimit := 10
	if v, ok := settings["rate_limit"]; ok {
		switch n := v.(type) {
		case float64:
			rateLimit = int(n)
		case int:
			rateLimit = n
		}
	}
	if !p.rateLimiter.Allow(ip, rateLimit) {
		return jsonResponse(429, map[string]any{
			"error":   "RATE_LIMITED",
			"message": "Too many submissions. Try again later.",
		}), nil
	}

	// Parse submission data from either JSON or multipart/form-data
	submissionData, pendingFiles, err := parseSubmissionBody(req)
	if err != nil {
		return jsonError(400, "INVALID_BODY", err.Error()), nil
	}

	// --- Honeypot spam check ---
	honeypotEnabled := true
	if v, ok := settings["honeypot_enabled"]; ok {
		if b, ok := v.(bool); ok {
			honeypotEnabled = b
		}
	}
	if honeypotEnabled {
		if hpVal, ok := submissionData["website_url"]; ok {
			if s, ok := hpVal.(string); ok && s != "" {
				// Bot detected — silently discard
				p.host.Log(ctx, "warn", fmt.Sprintf("Honeypot triggered for form %s (slug: %s), discarding submission", fmt.Sprintf("%v", formID), slug), map[string]any{"form_id": formID, "slug": slug})
				// Return 200 so bots don't realize they were blocked
				return jsonResponse(200, map[string]any{"success": true, "message": "Submission received"}), nil
			}
		}
	}

	// --- CAPTCHA verification ---
	provider, _ := settings["captcha_provider"].(string)
	if provider != "" && provider != "none" {
		secret, _ := settings["captcha_secret_key"].(string)
		token := extractCaptchaToken(submissionData)
		if captchaErr := p.verifyCAPTCHA(ctx, provider, secret, token, ip); captchaErr != nil {
			p.host.Log(ctx, "warn", "captcha failed: "+captchaErr.Error(), map[string]any{"form_id": formID})
			return jsonResponse(422, map[string]any{"error": "CAPTCHA_FAILED", "message": "CAPTCHA verification failed"}), nil
		}
	}

	// --- Input validation (text + file fields) ---
	fields := getFormFields(form)

	// Build field-by-ID map and group pending files by field name.
	fieldsByID := make(map[string]map[string]any, len(fields))
	for _, f := range fields {
		if id, ok := f["id"].(string); ok && id != "" {
			fieldsByID[id] = f
		}
	}
	filesByField := make(map[string][]uploadedFile)
	for _, pf := range pendingFiles {
		filesByField[pf.FieldName] = append(filesByField[pf.FieldName], pf)
	}

	fieldErrors := validateSubmission(submissionData, fields, filesByField)

	if len(fieldErrors) > 0 {
		return jsonResponse(422, map[string]any{
			"error":  "VALIDATION_FAILED",
			"fields": fieldErrors,
		}), nil
	}

	// --- Store uploaded files ---
	for fieldName, files := range filesByField {
		fd := fieldsByID[fieldName]
		multiple, _ := fd["multiple"].(bool)
		if multiple || len(files) > 1 {
			var arr []map[string]any
			for _, f := range files {
				meta, err := p.storeFile(ctx, formID, f)
				if err != nil {
					return jsonError(500, "STORE_FAILED", "file storage error: "+err.Error()), nil
				}
				arr = append(arr, meta)
			}
			submissionData[fieldName] = arr
		} else {
			meta, err := p.storeFile(ctx, formID, files[0])
			if err != nil {
				return jsonError(500, "STORE_FAILED", "file storage error: "+err.Error()), nil
			}
			submissionData[fieldName] = meta
		}
	}

	// --- Build metadata (Phase 1.5: respect store_ip) ---
	meta := map[string]any{
		"user_agent": req.GetHeaders()["User-Agent"],
		"referer":    req.GetHeaders()["Referer"],
	}
	// Default: store IP unless store_ip is explicitly set to false.
	shouldStoreIP := true
	if b, ok := settings["store_ip"].(bool); ok {
		shouldStoreIP = b
	}
	if shouldStoreIP {
		meta["ip"] = ip
	}

	submission := map[string]any{
		"form_id":    formID,
		"data":       submissionData,
		"metadata":   meta,
		"status":     "unread",
		"created_at": time.Now(),
	}

	created, err := p.host.DataCreate(ctx, submissionsTable, submission)
	if err != nil {
		return jsonError(500, "STORE_FAILED", err.Error()), nil
	}

	// Extract new submission ID
	newSubID := uint(0)
	if v, ok := created["id"].(float64); ok {
		newSubID = uint(v)
	}
	formSlug, _ := form["slug"].(string)

	// Trigger notifications
	go p.triggerNotifications(form, submissionData)

	// Emit event and fire webhook (after notifications to preserve ordering)
	p.emitSubmitted(ctx, formID, formSlug, newSubID, submissionData, meta)
	go p.fireWebhook(context.Background(), form, newSubID, submissionData, meta)

	return jsonResponse(200, map[string]any{"success": true, "message": "Submission received"}), nil
}

// parseSubmissionBody parses the request body as either JSON or multipart/form-data.
// Returns (formData, pendingFiles, error). pendingFiles contains any uploaded files.
func parseSubmissionBody(req *pb.PluginHTTPRequest) (map[string]any, []uploadedFile, error) {
	contentType := ""
	for k, v := range req.GetHeaders() {
		if strings.EqualFold(k, "content-type") {
			contentType = v
			break
		}
	}

	// If multipart/form-data, parse with multipart reader
	if contentType != "" {
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err == nil && strings.HasPrefix(mediaType, "multipart/") {
			boundary := params["boundary"]
			if boundary == "" {
				return nil, nil, fmt.Errorf("missing multipart boundary")
			}

			reader := multipart.NewReader(bytes.NewReader(req.GetBody()), boundary)
			data := make(map[string]any)
			var pendingFiles []uploadedFile

			const maxMultipartFileBytes = 50 * 1024 * 1024

			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					return nil, nil, fmt.Errorf("failed to parse multipart data: %w", err)
				}

				fieldName := part.FormName()
				if fieldName == "" {
					part.Close()
					continue
				}

				if part.FileName() != "" {
					body, err := io.ReadAll(io.LimitReader(part, maxMultipartFileBytes+1))
					part.Close()
					if err != nil {
						return nil, nil, fmt.Errorf("read file %s: %w", fieldName, err)
					}
					if len(body) > maxMultipartFileBytes {
						return nil, nil, fmt.Errorf("file %s exceeds 50MB hard cap", fieldName)
					}
					pendingFiles = append(pendingFiles, uploadedFile{
						FieldName: fieldName,
						FileName:  part.FileName(),
						MimeType:  part.Header.Get("Content-Type"),
						Size:      int64(len(body)),
						Body:      body,
					})
					continue
				}

				val, err := io.ReadAll(part)
				part.Close()
				if err != nil {
					return nil, nil, fmt.Errorf("failed to read field %s: %w", fieldName, err)
				}
				data[fieldName] = string(val)
			}
			return data, pendingFiles, nil
		}
	}

	// Default: parse as JSON
	var data map[string]any
	if err := json.Unmarshal(req.GetBody(), &data); err != nil {
		return nil, nil, fmt.Errorf("expected JSON or multipart/form-data body")
	}
	return data, nil, nil
}
