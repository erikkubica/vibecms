package api

import (
	"math"

	"github.com/gofiber/fiber/v2"
)

// ErrorResponse represents a standardized error response.
type ErrorResponse struct {
	Error struct {
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields,omitempty"`
	} `json:"error"`
}

// Success returns a 200 response with the given data wrapped in {"data": ...}.
func Success(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": data,
	})
}

// Created returns a 201 response with the given data wrapped in {"data": ...}.
func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": data,
	})
}

// Error returns an error response with the given status code, error code, and message.
func Error(c *fiber.Ctx, status int, code, message string) error {
	resp := ErrorResponse{}
	resp.Error.Code = code
	resp.Error.Message = message
	return c.Status(status).JSON(resp)
}

// ValidationError returns a 422 response with field-level validation errors.
func ValidationError(c *fiber.Ctx, fields map[string]string) error {
	resp := ErrorResponse{}
	resp.Error.Code = "VALIDATION_ERROR"
	resp.Error.Message = "One or more fields failed validation"
	resp.Error.Fields = fields
	return c.Status(fiber.StatusUnprocessableEntity).JSON(resp)
}

// PaginationMeta holds pagination metadata.
type PaginationMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int   `json:"total_pages"`
}

// Paginated returns a 200 response with paginated data and metadata.
func Paginated(c *fiber.Ctx, data interface{}, total int64, page, perPage int) error {
	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": data,
		"meta": PaginationMeta{
			Total:      total,
			Page:       page,
			PerPage:    perPage,
			TotalPages: totalPages,
		},
	})
}
