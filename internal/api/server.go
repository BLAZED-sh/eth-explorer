package api

import (
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/fipso/blazed-explorer/internal/ingest"
	"github.com/fipso/blazed-explorer/internal/store"
)

type Server struct {
	st        *store.Store
	feed      *ingest.Feed
	hub       *Hub
	startedAt time.Time
}

func NewServer(st *store.Store, feed *ingest.Feed, hub *Hub, webDist fs.FS) http.Handler {
	s := &Server{st: st, feed: feed, hub: hub, startedAt: time.Now()}

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
	mux.Handle("GET /", spaHandler(webDist))

	return mux
}

// spaHandler serves the embedded frontend build; paths without a matching
// file fall back to index.html for client-side routing.
func spaHandler(dist fs.FS) http.Handler {
	fileServer := http.FileServerFS(dist)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
