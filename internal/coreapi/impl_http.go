package coreapi

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout  = 30 * time.Second
	maxTimeout      = 120 * time.Second
	maxResponseBody = 10 * 1024 * 1024 // 10 MB
	maxRedirects    = 5
)

// blockedCIDRs are private/internal address ranges that the kernel must
// never speak to via Fetch — letting an extension's URL parameter route
// to localhost or a metadata service is the textbook SSRF foothold.
// Each iteration of CheckRedirect re-validates against this list because
// an attacker can host an http://attacker.example/redirect that 302s
// to http://169.254.169.254/.
var blockedCIDRs = []string{
	"127.0.0.0/8",     // loopback
	"10.0.0.0/8",      // RFC1918
	"172.16.0.0/12",   // RFC1918
	"192.168.0.0/16",  // RFC1918
	"169.254.0.0/16",  // link-local (incl. EC2/GCP/Azure metadata services)
	"::1/128",         // IPv6 loopback
	"fc00::/7",        // IPv6 unique local
	"fe80::/10",       // IPv6 link-local
	"100.64.0.0/10",   // CGNAT — often used for internal cluster networks
	"0.0.0.0/8",       // "this network"
}

// blockedNets is parsed once at init so the hot path stays cheap.
var blockedNets []*net.IPNet

func init() {
	for _, cidr := range blockedCIDRs {
		_, n, err := net.ParseCIDR(cidr)
		if err == nil {
			blockedNets = append(blockedNets, n)
		}
	}
}

// validateFetchURL parses the URL, enforces a scheme allowlist, and resolves
// the host against blockedNets to reject SSRF targets. Called both before
// the initial request and on every redirect.
func validateFetchURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return NewValidation("invalid URL: " + err.Error())
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return NewValidation(fmt.Sprintf("scheme %q not allowed (only http and https)", u.Scheme))
	}
	host := u.Hostname()
	if host == "" {
		return NewValidation("URL must include a host")
	}
	// Resolve all IPs the host advertises — DNS rebinding evasion would
	// fail-open if we only checked the first one.
	ips, err := net.LookupIP(host)
	if err != nil {
		return NewValidation("DNS lookup failed: " + err.Error())
	}
	for _, ip := range ips {
		for _, blocked := range blockedNets {
			if blocked.Contains(ip) {
				return NewValidation(fmt.Sprintf("destination %s resolves to blocked address %s", host, ip.String()))
			}
		}
	}
	return nil
}

func (c *coreImpl) Fetch(ctx context.Context, req FetchRequest) (*FetchResponse, error) {
	if req.URL == "" {
		return nil, NewValidation("URL is required")
	}

	if err := validateFetchURL(req.URL); err != nil {
		return nil, err
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}

	timeout := defaultTimeout
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
		if timeout > maxTimeout {
			timeout = maxTimeout
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, req.URL, bodyReader)
	if err != nil {
		return nil, NewInternal("failed to create request: " + err.Error())
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Bounded-redirect client that re-validates the target on each hop.
	// http.DefaultClient.Do would follow up to 10 redirects through any
	// IP — including back to localhost — without giving us a chance to
	// re-check.
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("too many redirects (>%d)", maxRedirects)
			}
			return validateFetchURL(r.URL.String())
		},
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, NewInternal("fetch failed: " + err.Error())
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxResponseBody)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, NewInternal("failed to read response body: " + err.Error())
	}

	headers := make(map[string]string, len(resp.Header))
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}

	return &FetchResponse{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       string(body),
	}, nil
}
