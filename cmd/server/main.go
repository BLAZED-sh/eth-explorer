package main

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/fipso/blazed-explorer/internal/api"
	"github.com/fipso/blazed-explorer/internal/config"
	"github.com/fipso/blazed-explorer/internal/ingest"
	"github.com/fipso/blazed-explorer/internal/store"
	"github.com/fipso/blazed-explorer/web"
)

// Disclaimer: AI Generated Code
func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.TimeOnly})

	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open store")
	}
	go st.RunWriter(ctx)
	go st.RunPruner(ctx, cfg.Retention)

	hub := api.NewHub()
	go hub.Run()

	// The node may not be up yet; keep retrying until it is.
	var feed *ingest.Feed
	for {
		feed, err = ingest.New(ctx, cfg.EthHTTPURL, cfg.EthWSURL, cfg.MaxPool, st, hub)
		if err == nil {
			break
		}
		log.Error().Err(err).Msg("eth node connect failed; retrying in 5s")
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
	hub.SetHelloer(feed)
	go feed.Run(ctx)

	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		log.Fatal().Err(err).Msg("embedded frontend missing")
	}

	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: api.NewServer(st, feed, hub, dist),
	}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	log.Info().Str("addr", cfg.ListenAddr).Msg("listening")
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("server failed")
	}
}
