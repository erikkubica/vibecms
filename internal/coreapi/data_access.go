package coreapi

import (
	"context"
	"strings"
)

// kernelPrivateTables lists tables that NO extension may read or
// write through the Data* API, regardless of declared ownership or
// capabilities. These hold authentication material, audit trails,
// and migration state — secrets that would constitute a
// privilege-escalation vector if a compromised extension could touch
// them.
//
// The list is intentionally restrictive: when in doubt, add the
// table. Extensions that legitimately need information from one of
// these (e.g. "list users with role X") should go through a typed
// CoreAPI method (GetUser, QueryUsers) where the kernel can apply
// row-level filtering before returning.
var kernelPrivateTables = map[string]bool{
	"users":                 true,
	"sessions":              true,
	"password_reset_tokens": true,
	"roles":                 true,
	"role_capabilities":     true,
	"audit_log":             true,
	"schema_migrations":     true,
	"site_settings":         true, // GetSetting/SetSetting are the supported path; raw row access bypasses redaction.
	"mcp_tokens":            true, // exposes other tokens' hashes; scope=full uses TokenSvc.List which preloads owner.
	"mcp_audit_log":         true,
	"mcp_token_audit":       true,
	"languages":             true, // language metadata can be enumerated through typed APIs; row access is rarely needed.
}

// IsKernelPrivateTable reports whether the named table is in the
// hard-coded deny list. Exposed for tests; callers in production
// should reach for checkTableAccess.
func IsKernelPrivateTable(table string) bool {
	return kernelPrivateTables[normalizeTableName(table)]
}

// normalizeTableName lowercases and trims double quotes so a caller
// passing `"Users"` or `users` reaches the same map entry as the
// guard's stored key.
func normalizeTableName(table string) string {
	t := strings.ToLower(strings.TrimSpace(table))
	t = strings.Trim(t, `"`)
	return t
}

// checkTableAccess implements the per-table allowlist for the Data*
// methods. The decision tree:
//
//  1. Internal callers bypass — kernel code is fully trusted.
//  2. Tables in kernelPrivateTables are denied unconditionally.
//  3. The caller must have declared the table in OwnedTables (from
//     extension.json's `data_owned_tables`); otherwise denied.
//
// Returning an error built with NewCapabilityDenied keeps the wire
// format consistent with the existing capability-denied responses
// extensions already know how to surface.
func checkTableAccess(ctx context.Context, table string) error {
	caller := CallerFromContext(ctx)
	if caller.Type == "internal" {
		return nil
	}
	tab := normalizeTableName(table)
	if kernelPrivateTables[tab] {
		return NewCapabilityDenied("data:" + tab + " (kernel-private table)")
	}
	if caller.OwnedTables[tab] {
		return nil
	}
	return NewCapabilityDenied("data:" + tab + " (not in extension's data_owned_tables)")
}
