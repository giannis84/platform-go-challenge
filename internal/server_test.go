package internal

import (
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name             string
		cfg              ServiceConfig
		wantAddr         string
		wantReadTimeout  time.Duration
		wantWriteTimeout time.Duration
		wantIdleTimeout  time.Duration
	}{
		{
			name: "applies default timeouts when none provided",
			cfg: ServiceConfig{
				Addr:   ":8080",
				Logger: testLogger(),
			},
			wantAddr:         ":8080",
			wantReadTimeout:  15 * time.Second,
			wantWriteTimeout: 15 * time.Second,
			wantIdleTimeout:  60 * time.Second,
		},
		{
			name: "uses custom timeouts when provided",
			cfg: ServiceConfig{
				Addr:         ":9090",
				Logger:       testLogger(),
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  30 * time.Second,
			},
			wantAddr:         ":9090",
			wantReadTimeout:  5 * time.Second,
			wantWriteTimeout: 10 * time.Second,
			wantIdleTimeout:  30 * time.Second,
		},
		{
			name: "partial custom timeouts uses defaults for the rest",
			cfg: ServiceConfig{
				Addr:        ":8080",
				Logger:      testLogger(),
				ReadTimeout: 3 * time.Second,
			},
			wantAddr:         ":8080",
			wantReadTimeout:  3 * time.Second,
			wantWriteTimeout: 15 * time.Second,
			wantIdleTimeout:  60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.cfg)

			if svc.HTTPServer == nil {
				t.Fatal("expected HTTPServer to be set")
			}
			if svc.Router == nil {
				t.Fatal("expected Router to be set")
			}
			if svc.Logger == nil {
				t.Fatal("expected Logger to be set")
			}
			if svc.HTTPServer.Addr != tt.wantAddr {
				t.Errorf("expected Addr %q, got %q", tt.wantAddr, svc.HTTPServer.Addr)
			}
			if svc.HTTPServer.ReadTimeout != tt.wantReadTimeout {
				t.Errorf("expected ReadTimeout %v, got %v", tt.wantReadTimeout, svc.HTTPServer.ReadTimeout)
			}
			if svc.HTTPServer.WriteTimeout != tt.wantWriteTimeout {
				t.Errorf("expected WriteTimeout %v, got %v", tt.wantWriteTimeout, svc.HTTPServer.WriteTimeout)
			}
			if svc.HTTPServer.IdleTimeout != tt.wantIdleTimeout {
				t.Errorf("expected IdleTimeout %v, got %v", tt.wantIdleTimeout, svc.HTTPServer.IdleTimeout)
			}
		})
	}
}

func TestNewService_RoutesRegistered(t *testing.T) {
	tests := []struct {
		name        string
		routes      RoutesRegistry
		method      string
		path        string
		wantStatus  int
	}{
		{
			name: "custom route is reachable",
			routes: func(r chi.Router) {
				r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
			},
			method:     "GET",
			path:       "/ping",
			wantStatus: http.StatusOK,
		},
		{
			name:       "nil routes produces working router",
			routes:     nil,
			method:     "GET",
			path:       "/anything",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(ServiceConfig{
				Addr:   ":0",
				Logger: testLogger(),
				Routes: tt.routes,
			})

			// Use the router directly as an http.Handler — no need to start a server
			rr := &fakeResponseWriter{headers: http.Header{}}
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			svc.Router.ServeHTTP(rr, req)

			if rr.code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.code)
			}
		})
	}
}

func TestNewService_HandlerIsRouter(t *testing.T) {
	svc := NewService(ServiceConfig{
		Addr:   ":8080",
		Logger: testLogger(),
	})

	if svc.HTTPServer.Handler != svc.Router {
		t.Error("expected HTTPServer.Handler to be the chi router")
	}
}

// fakeResponseWriter is a minimal http.ResponseWriter for testing.
type fakeResponseWriter struct {
	code    int
	headers http.Header
	body    []byte
}

func (f *fakeResponseWriter) Header() http.Header         { return f.headers }
func (f *fakeResponseWriter) Write(b []byte) (int, error)  { f.body = append(f.body, b...); return len(b), nil }
func (f *fakeResponseWriter) WriteHeader(code int)         { f.code = code }
