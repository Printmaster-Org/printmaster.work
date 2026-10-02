package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebsiteRoutes(t *testing.T) {
	handler, err := newHandler()
	if err != nil {
		t.Fatalf("newHandler() error = %v", err)
	}

	tests := []struct {
		name           string
		path           string
		wantStatus     int
		wantContent    string
		wantHeader     string
		wantHeaderText string
	}{
		{
			name:           "homepage",
			path:           "/",
			wantStatus:     http.StatusOK,
			wantContent:    "PrintMaster — Know your print fleet",
			wantHeader:     "Content-Type",
			wantHeaderText: "text/html; charset=utf-8",
		},
		{
			name:           "health check",
			path:           "/healthz",
			wantStatus:     http.StatusOK,
			wantContent:    "ok\n",
			wantHeader:     "Content-Type",
			wantHeaderText: "text/plain; charset=utf-8",
		},
		{
			name:           "embedded stylesheet",
			path:           "/assets/site.css",
			wantStatus:     http.StatusOK,
			wantContent:    "--bg:#002b36",
			wantHeader:     "Content-Type",
			wantHeaderText: "text/css",
		},
		{
			name:       "unknown page",
			path:       "/not-a-page",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			if res.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", res.Code, tt.wantStatus)
			}
			if tt.wantContent != "" && !strings.Contains(res.Body.String(), tt.wantContent) {
				t.Errorf("body does not contain %q", tt.wantContent)
			}
			if tt.wantHeader != "" && !strings.Contains(res.Header().Get(tt.wantHeader), tt.wantHeaderText) {
				t.Errorf("header %s = %q, want it to contain %q", tt.wantHeader, res.Header().Get(tt.wantHeader), tt.wantHeaderText)
			}

			for header, want := range map[string]string{
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":        "DENY",
				"Referrer-Policy":        "strict-origin-when-cross-origin",
			} {
				if got := res.Header().Get(header); got != want {
					t.Errorf("security header %s = %q, want %q", header, got, want)
				}
			}

			if tt.path == "/" && res.Header().Get("Cache-Control") != "no-cache" {
				t.Errorf("homepage Cache-Control = %q, want no-cache", res.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestListenAddress(t *testing.T) {
	tests := []struct {
		name string
		port string
		want string
	}{
		{name: "default", port: "", want: ":8080"},
		{name: "port number", port: "9090", want: ":9090"},
		{name: "colon-prefixed port", port: ":7070", want: ":7070"},
		{name: "surrounding whitespace", port: " 6060 ", want: ":6060"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PORT", tt.port)
			if got := listenAddress(); got != tt.want {
				t.Errorf("listenAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnknownPageDoesNotExposeHomepage(t *testing.T) {
	handler, err := newHandler()
	if err != nil {
		t.Fatalf("newHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if strings.Contains(res.Body.String(), "PrintMaster — Know your print fleet") {
		t.Fatal("unknown route returned homepage content")
	}
}