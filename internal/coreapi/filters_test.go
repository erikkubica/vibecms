package coreapi

import (
	"context"
	"testing"

	"vibecms/internal/cms"
	"vibecms/internal/email"
	"vibecms/internal/events"

	"github.com/gofiber/fiber/v2"
)

// newImplForFilterTests returns a coreImpl with just the filter map
// initialised — none of the DB-backed services are needed for filter tests.
func newImplForFilterTests() *coreImpl {
	// We bypass NewCoreImpl because it requires DB/services; the filter
	// implementation only touches `filters` and `nextFilterID`.
	return &coreImpl{
		filters: make(map[string][]filterEntry),
	}
}

// TestFilters_PriorityOrder verifies registered handlers run in
// priority-ascending order, mutating the value through the chain.
func TestFilters_PriorityOrder(t *testing.T) {
	c := newImplForFilterTests()
	ctx := context.Background()

	_, err := c.RegisterFilter(ctx, "title", 20, func(v any) any {
		return v.(string) + "-second"
	})
	if err != nil {
		t.Fatalf("register 20: %v", err)
	}
	_, err = c.RegisterFilter(ctx, "title", 10, func(v any) any {
		return v.(string) + "-first"
	})
	if err != nil {
		t.Fatalf("register 10: %v", err)
	}

	out, err := c.ApplyFilters(ctx, "title", "base")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	got, _ := out.(string)
	if got != "base-first-second" {
		t.Fatalf("expected base-first-second, got %q", got)
	}
}

// TestFilters_UnsubscribeRemoves verifies the returned UnsubscribeFunc
// actually drops the handler — historically a no-op due to a pointer-
// compare bug (impl_filters.go:31-42). Now uses an opaque registration ID.
func TestFilters_UnsubscribeRemoves(t *testing.T) {
	c := newImplForFilterTests()
	ctx := context.Background()

	unsub, err := c.RegisterFilter(ctx, "name", 0, func(v any) any { return "X" })
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	out, _ := c.ApplyFilters(ctx, "name", "input")
	if out.(string) != "X" {
		t.Fatalf("expected handler to fire, got %v", out)
	}

	unsub()
	out, _ = c.ApplyFilters(ctx, "name", "input")
	if out.(string) != "input" {
		t.Fatalf("expected handler removed, got %v", out)
	}

	// Calling unsub twice must be safe.
	unsub()
}

// TestFilters_ApplyValidation rejects empty names / nil handlers.
func TestFilters_ApplyValidation(t *testing.T) {
	c := newImplForFilterTests()
	ctx := context.Background()
	if _, err := c.RegisterFilter(ctx, "", 0, func(v any) any { return v }); err == nil {
		t.Fatal("expected error for empty filter name")
	}
	if _, err := c.RegisterFilter(ctx, "x", 0, nil); err == nil {
		t.Fatal("expected error for nil handler")
	}
}

// keep imports honest — the package would normally pull these via the
// filter implementation, but the standalone test file needs the references
// for the compile-time check below.
var (
	_ = cms.NewContentService
	_ = email.NewLogService
	_ = events.New
	_ = (&fiber.Ctx{})
)
