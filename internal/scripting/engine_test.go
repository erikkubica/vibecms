package scripting

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunScript_BasicExecution(t *testing.T) {
	engine := &ScriptEngine{
		eventHandlers: make(map[string][]string),
	}

	// Create temp scripts dir
	tmpDir := t.TempDir()
	engine.scriptsDir = tmpDir

	// Write a simple test script
	script := `
response = {
	status: 200,
	message: "hello from tengo"
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "test_handler.tengo"), []byte(script), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := engine.runScript("test_handler", nil)
	if err != nil {
		t.Fatalf("runScript failed: %v", err)
	}

	resp, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if resp["message"] != "hello from tengo" {
		t.Errorf("expected 'hello from tengo', got %v", resp["message"])
	}
}

func TestRunScript_WithVariables(t *testing.T) {
	engine := &ScriptEngine{
		eventHandlers: make(map[string][]string),
	}

	tmpDir := t.TempDir()
	engine.scriptsDir = tmpDir

	script := `
response = {
	action: event.action,
	node_id: event.payload.node_id
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "event_handler.tengo"), []byte(script), 0644); err != nil {
		t.Fatal(err)
	}

	vars := map[string]interface{}{
		"event": map[string]interface{}{
			"action": "node.created",
			"payload": map[string]interface{}{
				"node_id": 42,
			},
		},
	}

	result, err := engine.runScript("event_handler", vars)
	if err != nil {
		t.Fatalf("runScript failed: %v", err)
	}

	resp, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if resp["action"] != "node.created" {
		t.Errorf("expected 'node.created', got %v", resp["action"])
	}
}

func TestRunScript_ImportThemeModule(t *testing.T) {
	engine := &ScriptEngine{
		eventHandlers: make(map[string][]string),
	}

	tmpDir := t.TempDir()
	engine.scriptsDir = tmpDir

	// Create lib directory and helper module
	libDir := filepath.Join(tmpDir, "lib")
	os.MkdirAll(libDir, 0755)

	helperSrc := `
double := func(x) { return x * 2 }
export { double: double }
`
	os.WriteFile(filepath.Join(libDir, "math_helpers.tengo"), []byte(helperSrc), 0644)

	// Script that imports the helper
	script := `
helpers := import("./lib/math_helpers")
response = { result: helpers.double(21) }
`
	os.WriteFile(filepath.Join(tmpDir, "use_helpers.tengo"), []byte(script), 0644)

	result, err := engine.runScript("use_helpers", nil)
	if err != nil {
		t.Fatalf("runScript failed: %v", err)
	}

	resp, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if resp["result"] != int64(42) {
		t.Errorf("expected 42, got %v (%T)", resp["result"], resp["result"])
	}
}

func TestRunScript_StdlibAccess(t *testing.T) {
	engine := &ScriptEngine{
		eventHandlers: make(map[string][]string),
	}

	tmpDir := t.TempDir()
	engine.scriptsDir = tmpDir

	script := `
fmt := import("fmt")
math := import("math")
response = {
	formatted: fmt.sprintf("pi=%.2f", math.pi),
	abs_val: math.abs(-5)
}
`
	os.WriteFile(filepath.Join(tmpDir, "stdlib_test.tengo"), []byte(script), 0644)

	result, err := engine.runScript("stdlib_test", nil)
	if err != nil {
		t.Fatalf("runScript failed: %v", err)
	}

	resp, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if resp["formatted"] != "pi=3.14" {
		t.Errorf("expected 'pi=3.14', got %v", resp["formatted"])
	}
}

func TestRunScript_LogModule(t *testing.T) {
	engine := &ScriptEngine{
		eventHandlers: make(map[string][]string),
	}

	tmpDir := t.TempDir()
	engine.scriptsDir = tmpDir

	script := `
log := import("cms/log")
log.info("test message")
log.warn("warning message")
response = { logged: true }
`
	os.WriteFile(filepath.Join(tmpDir, "log_test.tengo"), []byte(script), 0644)

	result, err := engine.runScript("log_test", nil)
	if err != nil {
		t.Fatalf("runScript failed: %v", err)
	}

	resp, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if resp["logged"] != true {
		t.Errorf("expected true, got %v", resp["logged"])
	}
}

func TestLoadThemeScripts_NoScriptsDir(t *testing.T) {
	engine := &ScriptEngine{
		eventHandlers: make(map[string][]string),
	}

	// Non-existent theme dir — should not error
	err := engine.LoadThemeScripts("/tmp/nonexistent-theme-dir")
	if err != nil {
		t.Fatalf("expected nil error for missing scripts, got: %v", err)
	}
}

func TestLoadThemeScripts_EntryScript(t *testing.T) {
	engine := &ScriptEngine{
		eventHandlers: make(map[string][]string),
	}

	tmpDir := t.TempDir()
	scriptsDir := filepath.Join(tmpDir, "scripts")
	os.MkdirAll(scriptsDir, 0755)

	// Write a minimal theme.tengo that registers an event handler
	entryScript := `
log := import("cms/log")
events := import("cms/events")
http := import("cms/http")

log.info("test theme loading")
events.on("node.created", "handlers/on_created")
http.get("/test", "api/test_endpoint")
`
	os.WriteFile(filepath.Join(scriptsDir, "theme.tengo"), []byte(entryScript), 0644)

	err := engine.LoadThemeScripts(tmpDir)
	if err != nil {
		t.Fatalf("LoadThemeScripts failed: %v", err)
	}

	// Check registrations
	engine.mu.RLock()
	defer engine.mu.RUnlock()

	if handlers, ok := engine.eventHandlers["node.created"]; !ok || len(handlers) != 1 {
		t.Error("expected 1 event handler for node.created")
	} else if handlers[0] != "handlers/on_created" {
		t.Errorf("expected handler path 'handlers/on_created', got %q", handlers[0])
	}

	if len(engine.httpRoutes) != 1 {
		t.Errorf("expected 1 HTTP route, got %d", len(engine.httpRoutes))
	} else {
		if engine.httpRoutes[0].method != "GET" || engine.httpRoutes[0].path != "/test" {
			t.Errorf("unexpected route: %+v", engine.httpRoutes[0])
		}
	}
}
