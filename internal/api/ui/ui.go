package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:build placeholder.html
var assets embed.FS

func Handler() http.Handler {
	build, err := fs.Sub(assets, "build")
	if err != nil {
		panic(err)
	}
	files := http.FileServerFS(build)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if requested != "." {
			if info, err := fs.Stat(build, requested); err == nil && !info.IsDir() {
				files.ServeHTTP(w, r)
				return
			}
		}

		index, err := fs.ReadFile(build, "index.html")
		if err != nil {
			index, err = assets.ReadFile("placeholder.html")
			if err != nil {
				http.Error(w, "UI is unavailable", http.StatusInternalServerError)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	})
}
