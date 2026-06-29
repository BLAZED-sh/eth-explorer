package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	EthHTTPURL   string
	EthWSURL     string
	ListenAddr   string
	DBPath       string
	Retention    time.Duration
	MaxPool      int
	AllowOrigins []string

	// TLS / ACME. Explicit cert files take precedence; otherwise, when
	// ACMEEnabled, a cert is obtained via a manual dns-01 flow and cached.
	TLSCertFile   string
	TLSKeyFile    string
	ACMEEnabled   bool
	Domain        string
	ACMEEmail     string
	ACMEDirectory string
	CertCache     string
	ACMEDNSWait   time.Duration
}

func Load() Config {
	loadDotEnv(".env")
	return Config{
		EthHTTPURL:   getenv("ETH_HTTP_URL", "http://127.0.0.1:8545"),
		EthWSURL:     getenv("ETH_WS_URL", "ws://127.0.0.1:8546"),
		ListenAddr:   getenv("LISTEN_ADDR", ":8080"),
		DBPath:       getenv("DB_PATH", "./explorer.db"),
		Retention:    time.Duration(getenvInt("RETENTION_HOURS", 48)) * time.Hour,
		MaxPool:      getenvInt("MAX_POOL", 10000),
		AllowOrigins: getenvList("ALLOW_ORIGINS"),

		TLSCertFile:   os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:    os.Getenv("TLS_KEY_FILE"),
		ACMEEnabled:   getenvBool("ACME_ENABLED", false),
		Domain:        os.Getenv("DOMAIN"),
		ACMEEmail:     os.Getenv("ACME_EMAIL"),
		ACMEDirectory: os.Getenv("ACME_DIRECTORY"),
		CertCache:     getenv("CERT_CACHE", "./certs"),
		ACMEDNSWait:   time.Duration(getenvInt("ACME_DNS_WAIT", 600)) * time.Second,
	}
}

// loadDotEnv reads KEY=VALUE lines; real env vars take precedence.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if os.Getenv(key) == "" {
			os.Setenv(key, strings.TrimSpace(val))
		}
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getenvList parses a comma-separated env var into a trimmed, non-empty slice.
func getenvList(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
