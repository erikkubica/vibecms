package coreapi

import (
	"context"
	"errors"
	"testing"
)

func TestIsKernelPrivateTable(t *testing.T) {
	cases := []struct {
		table string
		want  bool
	}{
		{"users", true},
		{"USERS", true}, // case-insensitive
		{`"users"`, true}, // tolerates quoted form
		{" users ", true}, // trims whitespace
		{"sessions", true},
		{"password_reset_tokens", true},
		{"site_settings", true},
		{"audit_log", true},
		{"schema_migrations", true},
		{"media_files", false},
		{"content_nodes", false},
		{"foo_bar", false},
	}
	for _, c := range cases {
		if got := IsKernelPrivateTable(c.table); got != c.want {
			t.Errorf("IsKernelPrivateTable(%q) = %v, want %v", c.table, got, c.want)
		}
	}
}

// TestCheckTableAccess_InternalAlways verifies internal callers ignore
// both the kernel-private list and the OwnedTables allowlist. Kernel
// code must be able to manage every table — that's how migrations,
// seeding, and admin handlers work.
func TestCheckTableAccess_InternalAlways(t *testing.T) {
	ctx := WithCaller(context.Background(), InternalCaller())
	for _, tab := range []string{"users", "sessions", "anything", ""} {
		if err := checkTableAccess(ctx, tab); err != nil {
			t.Errorf("internal caller blocked on %q: %v", tab, err)
		}
	}
}

// TestCheckTableAccess_KernelPrivateAlwaysDenied ensures the deny
// list is enforced even for an extension that "owns" the table —
// declaring data_owned_tables: [users] in a hostile manifest must
// not grant a tenant-extension access to authentication material.
func TestCheckTableAccess_KernelPrivateAlwaysDenied(t *testing.T) {
	caller := CallerInfo{
		Slug:         "evil-ext",
		Type:         "grpc",
		Capabilities: map[string]bool{"data:read": true},
		OwnedTables:  map[string]bool{"users": true}, // hostile manifest entry
	}
	ctx := WithCaller(context.Background(), caller)

	err := checkTableAccess(ctx, "users")
	if err == nil {
		t.Fatal("kernel-private table should never be reachable, even with manifest claim")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !errors.Is(apiErr.Code, ErrCapabilityDenied) {
		t.Fatalf("expected ErrCapabilityDenied, got %v", err)
	}
}

func TestCheckTableAccess_OwnedAllowsExtensionTable(t *testing.T) {
	caller := CallerInfo{
		Slug:         "media-manager",
		Type:         "grpc",
		Capabilities: map[string]bool{"data:read": true},
		OwnedTables:  map[string]bool{"media_files": true},
	}
	ctx := WithCaller(context.Background(), caller)
	if err := checkTableAccess(ctx, "media_files"); err != nil {
		t.Fatalf("owned table should be allowed: %v", err)
	}
}

func TestCheckTableAccess_UnownedDenied(t *testing.T) {
	// An extension that owns media_files must NOT be able to reach
	// content_nodes or any other extension's tables. Default deny
	// is the whole point.
	caller := CallerInfo{
		Slug:         "media-manager",
		Type:         "grpc",
		Capabilities: map[string]bool{"data:read": true},
		OwnedTables:  map[string]bool{"media_files": true},
	}
	ctx := WithCaller(context.Background(), caller)
	for _, tab := range []string{"content_nodes", "menus", "themes", "extensions"} {
		if err := checkTableAccess(ctx, tab); err == nil {
			t.Errorf("non-owned table %q was allowed", tab)
		}
	}
}

func TestCheckTableAccess_NormalizesInputs(t *testing.T) {
	// A caller that owns "Media_Files" should match a request for
	// `"media_files"` — operators shouldn't be able to bypass either
	// the deny list or the allowlist via case or quotation games.
	caller := CallerInfo{
		Slug:         "media-manager",
		Type:         "grpc",
		Capabilities: map[string]bool{"data:read": true},
		OwnedTables:  map[string]bool{"media_files": true},
	}
	ctx := WithCaller(context.Background(), caller)
	cases := []string{"media_files", "MEDIA_FILES", `"media_files"`, " media_files "}
	for _, c := range cases {
		if err := checkTableAccess(ctx, c); err != nil {
			t.Errorf("normalized variant %q rejected: %v", c, err)
		}
	}
}

func TestCheckTableAccess_NilOwnedTablesDeniesAll(t *testing.T) {
	// Most extensions don't declare any data_owned_tables — they
	// shouldn't have raw DB access at all. nil OwnedTables must
	// behave like an empty allowlist.
	caller := CallerInfo{
		Slug:         "no-data",
		Type:         "grpc",
		Capabilities: map[string]bool{"data:read": true},
		OwnedTables:  nil,
	}
	ctx := WithCaller(context.Background(), caller)
	if err := checkTableAccess(ctx, "media_files"); err == nil {
		t.Fatal("nil OwnedTables should deny everything")
	}
}
