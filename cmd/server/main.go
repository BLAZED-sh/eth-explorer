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

	hub := api.NewHub(cfg.AllowOrigins)
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
		Handler: api.NewServer(st, feed, hub, dist, cfg.AllowOrigins),
	}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	// A Let's Encrypt cert is bound to the hostname, not a port, so the TLS
	// listener (cfg.ListenAddr) can be any port. When ACME is enabled we obtain
	// the cert in-process via a MANUAL dns-01 flow: on first run (or when the
	// cached cert is near expiry) the server logs the TXT record to create,
	// waits ACME_DNS_WAIT for you to add it, then validates and issues. A valid
	// cached cert in CERT_CACHE is reused on restart with no wait. Explicit
	// TLS_CERT_FILE / TLS_KEY_FILE take precedence and skip ACME.
	certFile, keyFile := cfg.TLSCertFile, cfg.TLSKeyFile
	if certFile == "" && cfg.ACMEEnabled {
		if cfg.Domain == "" {
			log.Fatal().Msg("ACME_ENABLED requires DOMAIN to be set")
		}
		c, k, err := ensureCert(ctx, cfg.CertCache, cfg.Domain, cfg.ACMEEmail, cfg.ACMEDirectory, cfg.ACMEDNSWait)
		if err != nil {
			log.Fatal().Err(err).Msg("acme dns-01 failed")
		}
		certFile, keyFile = c, k
	}

	if certFile != "" && keyFile != "" {
		log.Info().Str("addr", cfg.ListenAddr).Str("cert", certFile).Msg("listening (https)")
		if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("server failed")
		}
		return
	}

	log.Info().Str("addr", cfg.ListenAddr).Msg("listening (http)")
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("server failed")
	}
}
