package server

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var rawDistFS embed.FS

// DistFS is rooted at the Vite build output itself (paths like "index.html",
// not "dist/index.html") -- vite.config.ts points build.outDir directly at
// this package's dist/ subdirectory so `go build` embeds the real frontend
// with no separate copy step.
var DistFS = mustSub(rawDistFS, "dist")

func mustSub(f embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

// spaHandler serves the embedded frontend. This app is a single screen with
// no client-side routing, so the fallback below is mostly a robustness net
// rather than something normal use ever exercises: any GET whose path isn't
// a real embedded file falls back to index.html instead of a bare 404.
func spaHandler() http.Handler {
	fileServer := http.FileServerFS(DistFS)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if len(p) > 0 && p[0] == '/' {
			p = p[1:]
		}
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(DistFS, p); err != nil {
			r2 := new(http.Request)
			*r2 = *r
			u := *r.URL
			u.Path = "/"
			r2.URL = &u
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
