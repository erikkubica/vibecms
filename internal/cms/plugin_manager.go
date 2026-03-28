package cms

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	goplugin "github.com/hashicorp/go-plugin"
	vibeplugin "vibecms/pkg/plugin"

	"vibecms/internal/events"

	"google.golang.org/grpc"
)

// HostServerRegistrar is a function that registers the VibeCMSHost gRPC service
// on a grpc.Server for a given extension slug with its capabilities.
// This avoids an import cycle between cms and coreapi.
type HostServerRegistrar func(slug string, capabilities map[string]bool) func(s *grpc.Server)

// PluginManifestEntry represents a single plugin binary in an extension manifest.
type PluginManifestEntry struct {
	Binary string   `json:"binary"`
	Events []string `json:"events"`
}

// runningPlugin tracks a running plugin process and its client.
type runningPlugin struct {
	slug       string
	binary     string
	client     *goplugin.Client
	impl       vibeplugin.ExtensionPlugin
	eventNames []string
}

// PluginManager manages the lifecycle of gRPC plugin processes.
type PluginManager struct {
	mu            sync.RWMutex
	plugins       map[string][]*runningPlugin // extension slug -> running plugins
	eventBus      *events.EventBus
	hostRegistrar HostServerRegistrar
}

// NewPluginManager creates a new PluginManager.
// hostRegistrar may be nil if no CoreAPI is available (plugins won't get host callbacks).
func NewPluginManager(eventBus *events.EventBus, hostRegistrar HostServerRegistrar) *PluginManager {
	return &PluginManager{
		plugins:       make(map[string][]*runningPlugin),
		eventBus:      eventBus,
		hostRegistrar: hostRegistrar,
	}
}

// StartPlugins starts all plugin binaries declared in an extension's manifest.
// capabilities is the set of permissions declared in the extension manifest.
func (pm *PluginManager) StartPlugins(extPath string, slug string, manifest json.RawMessage, capabilities map[string]bool) error {
	// Parse plugins from manifest
	var m struct {
		Plugins []PluginManifestEntry `json:"plugins"`
	}
	if err := json.Unmarshal(manifest, &m); err != nil {
		return fmt.Errorf("parsing manifest plugins: %w", err)
	}

	if len(m.Plugins) == 0 {
		return nil // No plugins to start
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Stop any existing plugins for this extension first
	pm.stopPluginsLocked(slug)

	var started []*runningPlugin

	for _, pe := range m.Plugins {
		binaryPath := filepath.Join(extPath, pe.Binary)

		// Verify binary exists
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			log.Printf("[plugins] binary not found for %s: %s", slug, binaryPath)
			continue
		}

		// Start the plugin process
		client := goplugin.NewClient(&goplugin.ClientConfig{
			HandshakeConfig: vibeplugin.Handshake,
			VersionedPlugins: map[int]goplugin.PluginSet{
				2: vibeplugin.PluginMap,
			},
			Cmd:              exec.Command(binaryPath),
			AllowedProtocols: []goplugin.Protocol{goplugin.ProtocolGRPC},
		})

		rpcClient, err := client.Client()
		if err != nil {
			log.Printf("[plugins] failed to start plugin %s/%s: %v", slug, pe.Binary, err)
			client.Kill()
			continue
		}

		raw, err := rpcClient.Dispense("extension")
		if err != nil {
			log.Printf("[plugins] failed to dispense plugin %s/%s: %v", slug, pe.Binary, err)
			client.Kill()
			continue
		}

		impl, ok := raw.(vibeplugin.ExtensionPlugin)
		if !ok {
			log.Printf("[plugins] plugin %s/%s does not implement ExtensionPlugin", slug, pe.Binary)
			client.Kill()
			continue
		}

		// Initialize: start gRPC host service so plugin can call back into VibeCMS.
		if pm.hostRegistrar != nil {
			if grpcClient, ok := impl.(*vibeplugin.GRPCClient); ok {
				registerFn := pm.hostRegistrar(slug, capabilities)
				if initErr := grpcClient.InitializeHost(registerFn); initErr != nil {
					log.Printf("[plugins] failed to initialize host service for %s/%s: %v", slug, pe.Binary, initErr)
					client.Kill()
					continue
				}
			}
		}

		// Get subscriptions from the plugin
		subs, err := impl.GetSubscriptions()
		if err != nil {
			log.Printf("[plugins] failed to get subscriptions from %s/%s: %v", slug, pe.Binary, err)
			client.Kill()
			continue
		}

		rp := &runningPlugin{
			slug:   slug,
			binary: pe.Binary,
			client: client,
			impl:   impl,
		}

		// Register event subscriptions
		for _, sub := range subs {
			eventName := sub.EventName
			rp.eventNames = append(rp.eventNames, eventName)

			// Create a closure that calls the plugin's HandleEvent
			pluginImpl := impl // capture for closure
			pm.eventBus.Subscribe(eventName, func(action string, payload events.Payload) {
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					log.Printf("[plugins] failed to marshal payload for %s: %v", action, err)
					return
				}

				resp, err := pluginImpl.HandleEvent(action, payloadBytes)
				if err != nil {
					log.Printf("[plugins] error from %s/%s handling %s: %v", slug, pe.Binary, action, err)
					return
				}
				if resp.Error != "" {
					log.Printf("[plugins] %s/%s reported error for %s: %s", slug, pe.Binary, action, resp.Error)
				}
			})
		}

		started = append(started, rp)
		log.Printf("[plugins] started %s/%s with %d subscriptions", slug, pe.Binary, len(subs))
	}

	if len(started) > 0 {
		pm.plugins[slug] = started
	}

	return nil
}

// StopPlugins stops all plugin processes for an extension.
func (pm *PluginManager) StopPlugins(slug string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.stopPluginsLocked(slug)
}

// stopPluginsLocked stops plugins while holding the lock.
func (pm *PluginManager) stopPluginsLocked(slug string) {
	plugins, exists := pm.plugins[slug]
	if !exists {
		return
	}

	for _, rp := range plugins {
		log.Printf("[plugins] stopping %s/%s", slug, rp.binary)
		_ = rp.impl.Shutdown()
		rp.client.Kill()
	}

	delete(pm.plugins, slug)
}

// StopAll stops all running plugins (for graceful shutdown).
func (pm *PluginManager) StopAll() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for slug := range pm.plugins {
		pm.stopPluginsLocked(slug)
	}
}

// GetClient returns the first active GRPCClient for the given extension slug,
// or nil if no plugin is running for that slug.
func (pm *PluginManager) GetClient(slug string) *vibeplugin.GRPCClient {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins, exists := pm.plugins[slug]
	if !exists || len(plugins) == 0 {
		return nil
	}

	// Return the first plugin's impl cast to GRPCClient.
	if client, ok := plugins[0].impl.(*vibeplugin.GRPCClient); ok {
		return client
	}
	return nil
}

// RunningCount returns the number of running plugin processes.
func (pm *PluginManager) RunningCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	count := 0
	for _, plugins := range pm.plugins {
		count += len(plugins)
	}
	return count
}
