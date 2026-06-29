package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/acme"
)

const leDirectoryProd = "https://acme-v02.api.letsencrypt.org/directory"

// renewWindow is how long before expiry a cached cert is considered stale and
// the dns-01 flow is re-run on the next startup.
const renewWindow = 30 * 24 * time.Hour

// ensureCert returns paths to a valid cert + key for domain. If a cached cert in
// cacheDir is still valid (not within renewWindow of expiry) it is reused with no
// network calls. Otherwise it runs a MANUAL dns-01 flow: it logs the TXT record
// you must create, waits dnsWait for you to create it, then asks Let's Encrypt to
// validate and issues the cert. directory may be empty for LE production, or set
// to the staging URL for testing. email is optional (used for expiry notices).
func ensureCert(ctx context.Context, cacheDir, domain, email, directory string, dnsWait time.Duration) (certPath, keyPath string, err error) {
	certPath = filepath.Join(cacheDir, domain+".crt")
	keyPath = filepath.Join(cacheDir, domain+".key")

	if expiry, ok := certUsable(certPath, keyPath, domain); ok {
		log.Info().Str("domain", domain).Time("expires", expiry).Msg("using cached tls cert")
		return certPath, keyPath, nil
	}

	if err = os.MkdirAll(cacheDir, 0o700); err != nil {
		return "", "", fmt.Errorf("create cert cache dir: %w", err)
	}

	// Persist the ACME account key so we reuse the same account across runs.
	accountKey, err := loadOrCreateKey(filepath.Join(cacheDir, "account.key"))
	if err != nil {
		return "", "", fmt.Errorf("account key: %w", err)
	}

	if directory == "" {
		directory = leDirectoryProd
	}
	client := &acme.Client{Key: accountKey, DirectoryURL: directory}

	acct := &acme.Account{}
	if email != "" {
		acct.Contact = []string{"mailto:" + email}
	}
	if _, err = client.Register(ctx, acct, acme.AcceptTOS); err != nil && !errors.Is(err, acme.ErrAccountAlreadyExists) {
		return "", "", fmt.Errorf("acme register: %w", err)
	}

	order, err := client.AuthorizeOrder(ctx, acme.DomainIDs(domain))
	if err != nil {
		return "", "", fmt.Errorf("authorize order: %w", err)
	}

	for _, authzURL := range order.AuthzURLs {
		z, err := client.GetAuthorization(ctx, authzURL)
		if err != nil {
			return "", "", fmt.Errorf("get authorization: %w", err)
		}
		if z.Status == acme.StatusValid {
			continue
		}

		var chal *acme.Challenge
		for _, c := range z.Challenges {
			if c.Type == "dns-01" {
				chal = c
				break
			}
		}
		if chal == nil {
			return "", "", fmt.Errorf("no dns-01 challenge offered for %s", domain)
		}

		value, err := client.DNS01ChallengeRecord(chal.Token)
		if err != nil {
			return "", "", fmt.Errorf("compute dns-01 record: %w", err)
		}

		log.Warn().
			Str("name", "_acme-challenge."+domain).
			Str("type", "TXT").
			Str("value", value).
			Dur("waiting", dnsWait).
			Msg("ACTION REQUIRED: create this DNS TXT record now, then wait for validation")

		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-time.After(dnsWait):
		}

		if _, err = client.Accept(ctx, chal); err != nil {
			return "", "", fmt.Errorf("accept dns-01 challenge: %w", err)
		}
		if _, err = client.WaitAuthorization(ctx, z.URI); err != nil {
			return "", "", fmt.Errorf("dns-01 validation failed (is the TXT record set and propagated?): %w", err)
		}
		log.Info().Str("domain", domain).Msg("dns-01 challenge validated")
	}

	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate cert key: %w", err)
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: domain},
		DNSNames: []string{domain},
	}, certKey)
	if err != nil {
		return "", "", fmt.Errorf("create csr: %w", err)
	}

	der, _, err := client.CreateOrderCert(ctx, order.FinalizeURL, csr, true)
	if err != nil {
		return "", "", fmt.Errorf("finalize order: %w", err)
	}

	if err = writeCertPEM(certPath, der); err != nil {
		return "", "", err
	}
	if err = writeKeyPEM(keyPath, certKey); err != nil {
		return "", "", err
	}
	log.Info().Str("domain", domain).Str("cert", certPath).Msg("obtained tls cert via dns-01")
	return certPath, keyPath, nil
}

// certUsable reports whether the cached cert covers domain and is not within
// renewWindow of expiry, returning its expiry time.
func certUsable(certPath, keyPath, domain string) (time.Time, bool) {
	if _, err := os.Stat(keyPath); err != nil {
		return time.Time{}, false
	}
	pemBytes, err := os.ReadFile(certPath)
	if err != nil {
		return time.Time{}, false
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return time.Time{}, false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, false
	}
	if cert.VerifyHostname(domain) != nil {
		return time.Time{}, false
	}
	if time.Now().Add(renewWindow).After(cert.NotAfter) {
		return cert.NotAfter, false
	}
	return cert.NotAfter, true
}

func loadOrCreateKey(path string) (*ecdsa.PrivateKey, error) {
	if pemBytes, err := os.ReadFile(path); err == nil {
		block, _ := pem.Decode(pemBytes)
		if block == nil {
			return nil, fmt.Errorf("bad PEM in %s", path)
		}
		return x509.ParseECPrivateKey(block.Bytes)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	if err := writeKeyPEM(path, key); err != nil {
		return nil, err
	}
	return key, nil
}

func writeCertPEM(path string, der [][]byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, b := range der {
		if err := pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: b}); err != nil {
			return err
		}
	}
	return nil
}

func writeKeyPEM(path string, key *ecdsa.PrivateKey) error {
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
}
