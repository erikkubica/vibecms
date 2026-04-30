// Package testutil provides shared scaffolding for unit tests that need a
// real GORM connection without a Postgres container. We use the pure-Go
// modernc.org/sqlite driver (via glebarez/sqlite) so tests stay CGO-free —
// the CLAUDE.md hard-rule about Alpine-compatible builds applies to prod
// binaries; test binaries are unaffected because go test compiles _test.go
// files separately.
//
// SQLite is a poor stand-in for some Postgres features (JSONB, GIN, custom
// extensions), so don't reach for this helper to test JSONB query paths or
// migrations. It's tuned for small relational tables — site_settings,
// languages, etc. — where every dialect agrees on the basics.
package testutil

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewSQLiteDB returns an in-memory SQLite database wrapped in a GORM
// connection. The connection lives for the duration of the test — :memory:
// databases vanish when the last connection closes, so callers should
// keep the *gorm.DB around until the test completes. Logging is silenced
// to keep `go test` output clean.
func NewSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	// shared cache + the file: prefix means every gorm.Open call for the
	// same DSN reuses the same in-memory DB. We include the test name so
	// parallel tests get isolated databases.
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}
