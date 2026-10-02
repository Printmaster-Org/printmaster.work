package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed web
var siteFiles embed.FS

func main() {
	releases := newReleaseCatalogService(
		os.Getenv("GITHUB_API_URL"),
		os.Getenv("GITHUB_TOKEN"),
		os.Getenv("RELEASE_WEBHOOK_SECRET"),
		&http.Client{Timeout: 8 * time.Second},
	)
	handler, err := newHandlerWithReleaseService(releases)
	if err != nil {
		log.Fatalf("initialize website handler: %v", err)
	}
	startupCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	if err := releases.Refresh(startupCtx, ""); err != nil {
		log.Printf("initial GitHub release metadata refresh failed: %v", err)
	}
	cancel()

	server := &http.Server{
		Addr:              listenAddress(),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("PrintMaster website listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("website server: %v", err)
	}
}

func newHandler() (http.Handler, error) {
	releases := newReleaseCatalogService(
		os.Getenv("GITHUB_API_URL"),
		os.Getenv("GITHUB_TOKEN"),
		os.Getenv("RELEASE_WEBHOOK_SECRET"),
		&http.Client{Timeout: 8 * time.Second},
	)
	return newHandlerWithReleaseService(releases)
}

func newHandlerWithReleaseService(releases *releaseCatalogService) (http.Handler, error) {
	assets, err := fs.Sub(siteFiles, "web/assets")
	if err != nil {
		return nil, err
	}

	homepage, err := siteFiles.ReadFile("web/index.html")
	if err != nil {
		return nil, err
	}
	downloadsPage, err := siteFiles.ReadFile("web/downloads.html")
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets))))
	mux.HandleFunc("GET /api/releases", releases.handleHTTP)
	mux.HandleFunc("POST /api/releases/refresh", releases.handleRefreshHTTP)
	mux.HandleFunc("GET /downloads", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		if _, err := w.Write(downloadsPage); err != nil {
			log.Printf("write downloads page: %v", err)
		}
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		if _, err := w.Write(homepage); err != nil {
			log.Printf("write homepage: %v", err)
		}
	})

	return secureHeaders(mux), nil
}

func listenAddress() string {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	return port
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}
