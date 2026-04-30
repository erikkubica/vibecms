package cms

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"squilla/internal/events"
	"squilla/internal/settings"
)

// ExtensionSettingsBridge wires extension lifecycle events to the
// in-process settings registry. Extensions declare settings schemas in
// their extension.json `settings` array (each entry is a complete
// settings.Schema). On activation the bridge reads the manifest and
// registers every schema under "ext.<slug>.<id>"; on deactivation it
// unregisters everything with that prefix. Boot replay registers
// schemas for already-active extensions so the registry survives
// process restarts without requiring a deactivate/activate cycle.
type ExtensionSettingsBridge struct {
	loader   *ExtensionLoader
	registry *settings.Registry
	eventBus *events.EventBus
}

// NewExtensionSettingsBridge constructs the bridge. Subscribe still has
// to be called explicitly so wiring order in main.go stays controllable.
func NewExtensionSettingsBridge(loader *ExtensionLoader, registry *settings.Registry, eventBus *events.EventBus) *ExtensionSettingsBridge {
	return &ExtensionSettingsBridge{loader: loader, registry: registry, eventBus: eventBus}
}

// Subscribe attaches handlers to extension.activated / extension.deactivated.
func (b *ExtensionSettingsBridge) Subscribe() {
	b.eventBus.Subscribe("extension.activated", func(_ string, p events.Payload) {
		slug, _ := p["slug"].(string)
		if slug == "" {
			return
		}
		if err := b.RegisterFromDisk(slug); err != nil {
			log.Printf("[settings-bridge] register %q: %v", slug, err)
		}
	})
	b.eventBus.Subscribe("extension.deactivated", func(_ string, p events.Payload) {
		slug, _ := p["slug"].(string)
		if slug == "" {
			return
		}
		b.UnregisterAll(slug)
	})
}

// ReplayActive scans every currently-active extension and registers its
// schemas. Call once at boot after the loader has finished its initial
// scan — without this, schemas only land in the registry when the
// admin toggles an extension off and on again.
func (b *ExtensionSettingsBridge) ReplayActive() {
	exts, err := b.loader.GetActive()
	if err != nil {
		log.Printf("[settings-bridge] list active: %v", err)
		return
	}
	for _, ext := range exts {
		if err := b.RegisterFromDisk(ext.Slug); err != nil {
			log.Printf("[settings-bridge] replay %q: %v", ext.Slug, err)
		}
	}
}

// RegisterFromDisk reads the extension's manifest from disk and registers
// every declared schema. We re-read rather than relying on cached state
// because the loader doesn't keep the raw `settings` array around after
// the initial scan.
func (b *ExtensionSettingsBridge) RegisterFromDisk(slug string) error {
	manifestPath, err := b.findManifest(slug)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	var m ExtensionManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}
	for i, raw := range m.Settings {
		var schema settings.Schema
		if err := json.Unmarshal(raw, &schema); err != nil {
			log.Printf("[settings-bridge] %s settings[%d]: %v", slug, i, err)
			continue
		}
		if schema.ID == "" {
			log.Printf("[settings-bridge] %s settings[%d]: missing id", slug, i)
			continue
		}
		schema.ID = fmt.Sprintf("ext.%s.%s", slug, schema.ID)
		if err := b.registry.Register(schema); err != nil {
			log.Printf("[settings-bridge] %s register %q: %v", slug, schema.ID, err)
		}
	}
	return nil
}

// UnregisterAll removes every schema registered under the extension's
// namespace. Called on deactivation. We snapshot the current registry
// list to find matches because the registry exposes Get/List rather
// than range iteration.
func (b *ExtensionSettingsBridge) UnregisterAll(slug string) {
	prefix := fmt.Sprintf("ext.%s.", slug)
	for _, s := range b.registry.List() {
		if strings.HasPrefix(s.ID, prefix) {
			b.registry.Unregister(s.ID)
		}
	}
}

// findManifest resolves slug → extension.json path. Mirrors the loader's
// dataDir-wins-over-bundledDir rule so admin-installed extensions
// shadow image-bundled ones.
func (b *ExtensionSettingsBridge) findManifest(slug string) (string, error) {
	candidates := []string{
		filepath.Join(b.loader.dataDir, slug, "extension.json"),
		filepath.Join(b.loader.extensionsDir, slug, "extension.json"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("manifest not found for %q", slug)
}
