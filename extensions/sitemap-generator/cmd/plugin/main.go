package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"squilla/internal/coreapi"
	vibeplugin "squilla/pkg/plugin"
	coreapipb "squilla/pkg/plugin/coreapipb"
	pb "squilla/pkg/plugin/proto"
)

// --- XML sitemap structures ---

type sitemapIndex struct {
	XMLName  xml.Name       `xml:"sitemapindex"`
	XMLNS    string         `xml:"xmlns,attr"`
	Sitemaps []sitemapEntry `xml:"sitemap"`
}

type sitemapEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type urlset struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

// --- Plugin ---

type SitemapPlugin struct {
	host *coreapi.GRPCHostClient
}

func (p *SitemapPlugin) GetSubscriptions() ([]*pb.Subscription, error) {
	return []*pb.Subscription{
		{EventName: "sitemap.rebuild", Priority: 10},
		{EventName: "sitemap.get", Priority: 10},
		{EventName: "setting.updated", Priority: 50},
		{EventName: "node.created", Priority: 50},
		{EventName: "node.updated", Priority: 50},
		{EventName: "node.published", Priority: 50},
		{EventName: "node.deleted", Priority: 50},
	}, nil
}

func (p *SitemapPlugin) HandleEvent(action string, payload []byte) (*pb.EventResponse, error) {
	switch action {
	case "sitemap.rebuild", "sitemap.get":
		if err := p.buildAllSitemaps(); err != nil {
			return &pb.EventResponse{Handled: true, Error: fmt.Sprintf("sitemap build failed: %v", err)}, nil
		}
		return &pb.EventResponse{Handled: true}, nil
	case "setting.updated":
		// Skip rebuilds triggered by our own writes (sitemap_*_*) — otherwise
		// every rebuild's SetSetting calls fan back in here, causing a storm
		// of nested rebuilds whenever ANY setting is saved.
		if isOwnSitemapEvent(payload) {
			return &pb.EventResponse{Handled: false}, nil
		}
		_ = p.buildAllSitemaps()
		return &pb.EventResponse{Handled: false}, nil
	case "node.created", "node.updated", "node.published", "node.deleted":
		_ = p.buildAllSitemaps()
		return &pb.EventResponse{Handled: false}, nil
	default:
		return &pb.EventResponse{Handled: false}, nil
	}
}

// isOwnSitemapEvent returns true when the setting.updated payload reports a
// key starting with "sitemap_" — i.e. one this plugin just wrote. The payload
// is a JSON object {"key": "..."} (and may also carry "bulk":true for batch
// updates, which we still want to handle).
func isOwnSitemapEvent(payload []byte) bool {
	// Cheap substring check — avoids importing encoding/json just for this
	// hot path. The bus payload is a small JSON object, so a contains test
	// is unambiguous in practice.
	return strings.Contains(string(payload), `"key":"sitemap_`)
}

func (p *SitemapPlugin) Initialize(hostConn *grpc.ClientConn) error {
	p.host = coreapi.NewGRPCHostClient(coreapipb.NewSquillaHostClient(hostConn))

	if err := p.buildAllSitemaps(); err != nil {
		ctx := context.Background()
		_ = p.host.Log(ctx, "warn", fmt.Sprintf("initial sitemap build failed: %v", err), nil)
	}
	return nil
}

func (p *SitemapPlugin) HandleHTTPRequest(req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	return &pb.PluginHTTPResponse{StatusCode: 404, Body: []byte(`{"error":"not found"}`)}, nil
}

func (p *SitemapPlugin) Shutdown() error {
	return nil
}

// langInfo holds language metadata needed for sitemap generation.
type langInfo struct {
	Code       string
	Slug       string
	IsDefault  bool
	HidePrefix bool
}

// buildAllSitemaps generates per-language sitemap indexes and per-type sub-sitemaps.
func (p *SitemapPlugin) buildAllSitemaps() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get site URL.
	siteURL, err := p.host.GetSetting(ctx, "site_url")
	if err != nil || siteURL == "" {
		siteURL = "http://localhost:8099"
	}
	siteURL = strings.TrimRight(siteURL, "/")

	// Query all published nodes.
	nodeList, err := p.host.QueryNodes(ctx, coreapi.NodeQuery{
		Status: "published",
		Limit:  50000,
	})
	if err != nil {
		return fmt.Errorf("query nodes: %w", err)
	}

	// Get languages from settings. We store them as a comma-separated list
	// in "sitemap_languages". If not set, derive from the nodes themselves.
	languages := p.getLanguages(ctx, nodeList.Nodes)

	// Group nodes by language_code -> node_type -> []node
	grouped := make(map[string]map[string][]*coreapi.Node)
	for _, node := range nodeList.Nodes {
		if node.FullURL == "" {
			continue
		}
		lang := node.LanguageCode
		if lang == "" {
			lang = "en" // fallback
		}
		if grouped[lang] == nil {
			grouped[lang] = make(map[string][]*coreapi.Node)
		}
		grouped[lang][node.NodeType] = append(grouped[lang][node.NodeType], node)
	}

	totalURLs := 0
	now := time.Now().Format(time.RFC3339)

	// For each language, generate sub-sitemaps and an index.
	for _, lang := range languages {
		typeNodes, ok := grouped[lang.Code]
		if !ok {
			continue
		}

		langPrefix := lang.Slug + "/"
		if lang.HidePrefix {
			langPrefix = ""
		}

		var indexEntries []sitemapEntry

		// Generate a sub-sitemap per node type.
		for nodeType, nodes := range typeNodes {
			var urls []sitemapURL
			for _, node := range nodes {
				urls = append(urls, sitemapURL{
					Loc:     siteURL + node.FullURL,
					LastMod: node.UpdatedAt.Format(time.RFC3339),
				})
			}
			totalURLs += len(urls)

			set := urlset{
				XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
				URLs:  urls,
			}
			xmlBytes, err := xml.MarshalIndent(set, "", "  ")
			if err != nil {
				continue
			}
			xmlStr := xml.Header + string(xmlBytes)

			// Store: sitemap_<type>_<langcode>
			key := fmt.Sprintf("sitemap_%s_%s", nodeType, lang.Code)
			_ = p.host.SetSetting(ctx, key, xmlStr)

			// Add to index.
			indexEntries = append(indexEntries, sitemapEntry{
				Loc:     fmt.Sprintf("%s/%ssitemap-%s.xml", siteURL, langPrefix, nodeType),
				LastMod: now,
			})
		}

		// Generate the index sitemap for this language.
		idx := sitemapIndex{
			XMLNS:    "http://www.sitemaps.org/schemas/sitemap/0.9",
			Sitemaps: indexEntries,
		}
		idxBytes, err := xml.MarshalIndent(idx, "", "  ")
		if err != nil {
			continue
		}
		idxStr := xml.Header + string(idxBytes)

		// Store: sitemap_index_<langcode>
		key := fmt.Sprintf("sitemap_index_%s", lang.Code)
		_ = p.host.SetSetting(ctx, key, idxStr)
	}

	_ = p.host.Log(ctx, "info", fmt.Sprintf("sitemaps rebuilt: %d URLs across %d languages", totalURLs, len(languages)), nil)
	return nil
}

// getLanguages discovers available languages from node data and settings.
func (p *SitemapPlugin) getLanguages(ctx context.Context, nodes []*coreapi.Node) []langInfo {
	// Discover unique language codes from nodes.
	seen := make(map[string]bool)
	for _, n := range nodes {
		if n.LanguageCode != "" {
			seen[n.LanguageCode] = true
		}
	}

	// Get language configuration from settings.
	// Format stored by Squilla: default_language, and per-language settings.
	defaultLang, _ := p.host.GetSetting(ctx, "default_language")
	if defaultLang == "" {
		defaultLang = "en"
	}

	// Build langInfo for each discovered language.
	// For languages with hide_prefix (typically the default), the sitemap
	// is served at /sitemap.xml. Others at /<slug>/sitemap.xml.
	var langs []langInfo
	for code := range seen {
		li := langInfo{
			Code:       code,
			Slug:       code, // slug == code in most configurations
			IsDefault:  code == defaultLang,
			HidePrefix: code == defaultLang,
		}
		langs = append(langs, li)
	}

	// Ensure default language is first.
	for i, l := range langs {
		if l.IsDefault && i > 0 {
			langs[0], langs[i] = langs[i], langs[0]
			break
		}
	}

	if len(langs) == 0 {
		langs = []langInfo{{Code: "en", Slug: "en", IsDefault: true, HidePrefix: true}}
	}

	return langs
}

func main() {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: vibeplugin.Handshake,
		VersionedPlugins: map[int]goplugin.PluginSet{
			2: {"extension": &vibeplugin.ExtensionGRPCPlugin{Impl: &SitemapPlugin{}}},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})
}
