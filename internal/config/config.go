package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	EthHTTPURL string
	EthWSURL   string
	ListenAddr string
	DBPath     string
	Retention  time.Duration
	MaxPool    int
}

func Load() Config {
	loadDotEnv(".env")
	return Config{
		EthHTTPURL: getenv("ETH_HTTP_URL", "http://127.0.0.1:8545"),
		EthWSURL:   getenv("ETH_WS_URL", "ws://127.0.0.1:8546"),
		ListenAddr: getenv("LISTEN_ADDR", ":8080"),
		DBPath:     getenv("DB_PATH", "./explorer.db"),
		Retention:  time.Duration(getenvInt("RETENTION_HOURS", 48)) * time.Hour,
		MaxPool:    getenvInt("MAX_POOL", 10000),
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

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
