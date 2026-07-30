// Package spa embeds the built SPA and serves it with client-side routing
// fallback (any non-asset path returns index.html).
package spa

import (
	"bytes"
	"embed"
	"fmt"
	"html"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// Handler serves the embedded SPA below basePath. Real asset paths are served
// directly; other paths inside the base fall back to index.html for client-side
// routing. Requests outside the configured base remain 404 so a shared host can
// safely serve other applications.
func Handler(rawBasePath string) (http.Handler, error) {
	basePath, err := normalizeBasePath(rawBasePath)
	if err != nil {
		return nil, err
	}
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	fileServer := http.FileServer(http.FS(sub))
	index, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		return nil, err
	}
	baseHref := basePath
	if baseHref != "/" {
		baseHref += "/"
	}
	baseTag := fmt.Sprintf(`<base href="%s">`, html.EscapeString(baseHref))
	metaTag := fmt.Sprintf(
		`<meta name="tenancit-base-path" content="%s">`,
		html.EscapeString(basePath),
	)
	basePlaceholder := firstExistingMarker(index, `<base href="/">`, `<base href="/" />`)
	metaPlaceholder := firstExistingMarker(
		index,
		`<meta name="tenancit-base-path" content="/">`,
		`<meta name="tenancit-base-path" content="/" />`,
	)
	hadBasePlaceholder := basePlaceholder != ""
	hadMetaPlaceholder := metaPlaceholder != ""
	if hadBasePlaceholder != hadMetaPlaceholder {
		return nil, fmt.Errorf("embedded SPA index has an incomplete base-path marker")
	}
	if hadBasePlaceholder {
		index = bytes.Replace(index, []byte(basePlaceholder), []byte(baseTag), 1)
		index = bytes.Replace(index, []byte(metaPlaceholder), []byte(metaTag), 1)
	} else {
		runtimeHead := fmt.Sprintf(
			"<head>\n    %s\n    %s",
			baseTag,
			metaTag,
		)
		index = bytes.Replace(index, []byte("<head>"), []byte(runtimeHead), 1)
	}
	if !bytes.Contains(index, []byte(`name="tenancit-base-path"`)) {
		return nil, fmt.Errorf("embedded SPA index is missing the head marker")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath := r.URL.Path
		if basePath != "/" {
			if requestPath == basePath {
				http.Redirect(w, r, basePath+"/", http.StatusPermanentRedirect)
				return
			}
			if !strings.HasPrefix(requestPath, basePath+"/") {
				http.NotFound(w, r)
				return
			}
			requestPath = strings.TrimPrefix(requestPath, basePath)
		}

		p := strings.TrimPrefix(requestPath, "/")
		if p == "v1" || strings.HasPrefix(p, "v1/") || (p != "" && !fs.ValidPath(p)) {
			http.NotFound(w, r)
			return
		}
		if p != "" {
			if f, openErr := sub.Open(p); openErr == nil {
				info, statErr := f.Stat()
				_ = f.Close()
				if statErr == nil && !info.IsDir() {
					requestCopy := r.Clone(r.Context())
					urlCopy := *r.URL
					urlCopy.Path = "/" + p
					urlCopy.RawPath = ""
					requestCopy.URL = &urlCopy
					fileServer.ServeHTTP(w, requestCopy)
					return
				}
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	}), nil
}

func normalizeBasePath(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" || value == "/" {
		return "/", nil
	}
	if !strings.HasPrefix(value, "/") ||
		strings.HasPrefix(value, "//") ||
		strings.HasPrefix(value, `/\`) ||
		strings.ContainsAny(value, `\?#<>"'`+"\r\n\t ") ||
		strings.Contains(value, "//") {
		return "", fmt.Errorf("TENANCIT_BASE_PATH must be an absolute URL path")
	}
	value = strings.TrimSuffix(value, "/")
	if path.Clean(value) != value {
		return "", fmt.Errorf("TENANCIT_BASE_PATH must not contain dot segments")
	}
	for _, segment := range strings.Split(strings.TrimPrefix(value, "/"), "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("TENANCIT_BASE_PATH contains an invalid segment")
		}
		for _, char := range segment {
			if !isBasePathChar(char) {
				return "", fmt.Errorf("TENANCIT_BASE_PATH contains unsupported characters")
			}
		}
	}
	return value, nil
}

func isBasePathChar(char rune) bool {
	return char >= 'a' && char <= 'z' ||
		char >= 'A' && char <= 'Z' ||
		char >= '0' && char <= '9' ||
		strings.ContainsRune("-._~", char)
}

func firstExistingMarker(content []byte, candidates ...string) string {
	for _, candidate := range candidates {
		if bytes.Contains(content, []byte(candidate)) {
			return candidate
		}
	}
	return ""
}
