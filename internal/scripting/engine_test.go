package scripting

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunScript_BasicExecution(t *testing.T) {
	engine := &ScriptEngine{
		eventHandlers: make(map[string][]scriptHandler),
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

	result, err := engine.runScript("test_handler", "test", nil, "", nil, nil)
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
		eventHandlers: make(map[string][]scriptHandler),
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

	result, err := engine.runScript("event_handler", "test", nil, "", vars, nil)
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
		eventHandlers: make(map[string][]scriptHandler),
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

	result, err := engine.runScript("use_helpers", "test", nil, "", nil, nil)
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
		eventHandlers: make(map[string][]scriptHandler),
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

	result, err := engine.runScript("stdlib_test", "test", nil, "", nil, nil)
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
	t.Skip("requires CoreAPI stub — rebuild as part of test pyramid (task #66)")
}

func TestLoadThemeScripts_NoScriptsDir(t *testing.T) {
	engine := &ScriptEngine{
		eventHandlers: make(map[string][]scriptHandler),
	}

	// Non-existent theme dir — should not error
	err := engine.LoadThemeScripts("/tmp/nonexistent-theme-dir")
	if err != nil {
		t.Fatalf("expected nil error for missing scripts, got: %v", err)
	}
}

func TestLoadThemeScripts_EntryScript(t *testing.T) {
	t.Skip("requires CoreAPI stub — rebuild as part of test pyramid (task #66)")
}
