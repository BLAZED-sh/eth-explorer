package api

import (
	"bufio"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/fipso/blazed-explorer/internal/ingest"
	"github.com/fipso/blazed-explorer/internal/store"
)

type Server struct {
	st        *store.Store
	feed      *ingest.Feed
	hub       *Hub
	startedAt time.Time
	siteURL   string
}

func NewServer(st *store.Store, feed *ingest.Feed, hub *Hub, webDist fs.FS, allowOrigins []string, siteURL string, blockedUA *regexp.Regexp) http.Handler {
	s := &Server{
		st:        st,
		feed:      feed,
		hub:       hub,
		startedAt: time.Now(),
		siteURL:   strings.TrimSuffix(siteURL, "/"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/txs", s.handleTxs)
	mux.HandleFunc("GET /api/txs/{hash}", s.handleTxDetail)
	mux.HandleFunc("GET /api/address/{addr}", s.handleAddress)
	mux.HandleFunc("GET /api/address/{addr}/code", s.handleAddressCode)
	mux.HandleFunc("GET /api/blocks", s.handleBlocks)
	mux.HandleFunc("GET /api/blocks/{number}", s.handleBlockDetail)
	mux.HandleFunc("GET /api/gas", s.handleGas)
	mux.HandleFunc("GET /api/stats/history", s.handleHistory)
	mux.HandleFunc("GET /api/search", s.handleSearch)
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /ws", hub.ServeWS)
	mux.HandleFunc("GET /sitemap-blocks.xml", s.handleBlocksSitemap)
	mux.Handle("GET /", s.spaHandler(webDist))

	return logMiddleware(blockUAMiddleware(blockedUA, corsMiddleware(allowOrigins, mux)))
}

// logMiddleware logs one line per request. The swarm ingress hides client
// IPs, so the User-Agent is the only way to tell crawlers apart.
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", rec.status).
			Dur("dur", time.Since(start)).
			Str("ua", r.UserAgent()).
			Msg("req")
	})
}

// blockUAMiddleware rejects requests whose User-Agent matches re (BLOCKED_UA).
func blockUAMiddleware(re *regexp.Regexp, next http.Handler) http.Handler {
	if re == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if re.MatchString(r.UserAgent()) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// Hijack is required by gorilla/websocket, which type-asserts http.Hijacker.
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	r.status = http.StatusSwitchingProtocols
	return http.NewResponseController(r.ResponseWriter).Hijack()
}

// corsMiddleware adds CORS headers (and answers preflight requests) for the
// configured allowed origins. With no allowed origins it is a no-op: prod is
// same-origin and dev uses the Vite proxy, so no cross-origin headers are
// needed. Wrapping the whole mux lets it answer OPTIONS preflight before the
// method-scoped routes would reject it.
func corsMiddleware(allowOrigins []string, next http.Handler) http.Handler {
	if len(allowOrigins) == 0 {
		return next
	}
	oc := newOriginChecker(allowOrigins)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && oc.allows(origin) {
			if oc.allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// spaHandler serves the embedded frontend build; paths without a matching
// file fall back to index.html for client-side routing. Requests reaching
// this copy under any host other than the canonical site get X-Robots-Tag
// noindex, so the embedded duplicate never competes with the public frontend.
func (s *Server) spaHandler(dist fs.FS) http.Handler {
	fileServer := http.FileServerFS(dist)
	canonicalHost := ""
	if u, err := url.Parse(s.siteURL); err == nil {
		canonicalHost = u.Host
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if canonicalHost != "" && r.Host != canonicalHost {
			w.Header().Set("X-Robots-Tag", "noindex")
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path != "" {
			if f, err := dist.Open(path); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		http.ServeFileFS(w, r, dist, "index.html")
	})
}
