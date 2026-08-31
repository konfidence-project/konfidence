package ui

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/konfidence-project/konfidence/internal/api/apierror"
)

const (
	defaultCacheControl   = "public, max-age=3600"
	immutableCacheControl = "public, max-age=31536000, immutable"
	indexCacheControl     = "no-cache, must-revalidate"
)

type handler struct {
	fs        fs.FS
	files     http.Handler
	indexHTML []byte
	indexETag string
}

// New returns a static file handler with an index fallback for SPA routes.
func New(files fs.FS) (http.Handler, error) {
	indexHTML, err := fs.ReadFile(files, "index.html")
	if err != nil {
		return nil, fmt.Errorf("read UI index: %w", err)
	}
	indexHash := sha256.Sum256(indexHTML)

	return &handler{
		fs:        files,
		files:     http.FileServerFS(files),
		indexHTML: indexHTML,
		indexETag: `W/"` + hex.EncodeToString(indexHash[:8]) + `"`,
	}, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		notFound(w, r)
		return
	}

	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if info, err := fs.Stat(h.fs, name); err == nil && !info.IsDir() {
		if strings.HasPrefix(r.URL.Path, "/_app/immutable/") {
			w.Header().Set("Cache-Control", immutableCacheControl)
		} else {
			w.Header().Set("Cache-Control", defaultCacheControl)
		}
		h.files.ServeHTTP(w, r)
		return
	}
	if name != "." && path.Ext(name) != "" {
		notFound(w, r)
		return
	}

	w.Header().Set("Cache-Control", indexCacheControl)
	w.Header().Set("Content-Type", mime.TypeByExtension(".html"))
	w.Header().Set("ETag", h.indexETag)
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(h.indexHTML))
}

func notFound(w http.ResponseWriter, r *http.Request) {
	if !wantsJSON(r.Header.Get("Accept")) {
		http.NotFound(w, r)
		return
	}
	apierror.Write(w, apierror.NewNotFound("route", r.URL.Path))
}

func wantsJSON(accept string) bool {
	return strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/html")
}
