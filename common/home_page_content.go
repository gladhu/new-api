package common

import (
	"net/url"
	"strings"
)

// NormalizeHomePageContentSource rewrites absolute URLs that point at the
// configured server address into site-relative paths so CDN mirror domains
// (for example www.hk.example.com) can load the same custom home page.
func NormalizeHomePageContentSource(source, serverAddress string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return source
	}
	if strings.HasPrefix(source, "/") {
		return source
	}
	if !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") {
		return source
	}

	parsed, err := url.Parse(source)
	if err != nil || parsed.Path == "" || parsed.Path == "/" {
		return source
	}

	serverBase := strings.TrimRight(strings.TrimSpace(serverAddress), "/")
	if serverBase == "" {
		return source
	}

	serverParsed, err := url.Parse(serverBase)
	if err != nil || serverParsed.Host == "" {
		return source
	}

	if !strings.EqualFold(parsed.Host, serverParsed.Host) {
		return source
	}

	relative := parsed.EscapedPath()
	if parsed.RawQuery != "" {
		relative += "?" + parsed.RawQuery
	}
	if parsed.Fragment != "" {
		relative += "#" + parsed.Fragment
	}
	return relative
}
