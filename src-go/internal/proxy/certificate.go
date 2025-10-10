package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CertificateManager handles CA and server certificate generation
type CertificateManager struct {
	mu           sync.RWMutex
	caKey        *rsa.PrivateKey
	caCert       *x509.Certificate
	certPath     string
	keyPath      string
	certCache    map[string]*tls.Certificate
	cacheMaxSize int
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager(dataDir string) (*CertificateManager, error) {
	cm := &CertificateManager{
		certCache:    make(map[string]*tls.Certificate),
		cacheMaxSize: 100,
		certPath:     filepath.Join(dataDir, "certificates", "ca.crt"),
		keyPath:      filepath.Join(dataDir, "certificates", "ca.key"),
	}

	// Ensure certificates directory exists
	certDir := filepath.Dir(cm.certPath)
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create certificates directory: %w", err)
	}

	// Load or generate CA certificate
	if err := cm.initializeCA(); err != nil {
		return nil, fmt.Errorf("failed to initialize CA: %w", err)
	}

	return cm, nil
}

// initializeCA loads or generates the CA certificate
func (cm *CertificateManager) initializeCA() error {
	// Try to load existing CA certificate
	if _, err := os.Stat(cm.certPath); err == nil {
		if _, err := os.Stat(cm.keyPath); err == nil {
			return cm.loadCA()
		}
	}

	// Generate new CA certificate
	return cm.generateCA()
}

// generateCA generates a new CA certificate
func (cm *CertificateManager) generateCA() error {
	// Generate RSA key pair
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(10 * 365 * 24 * time.Hour) // 10 years

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:         "KProxy CA",
			Country:            []string{"US"},
			Organization:       []string{"KProxy"},
			OrganizationalUnit: []string{"KProxy Certificate Authority"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Save certificate
	certOut, err := os.Create(cm.certPath)
	if err != nil {
		return fmt.Errorf("failed to create certificate file: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Save private key
	keyOut, err := os.OpenFile(cm.keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyOut.Close()

	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	cm.caKey = key
	cm.caCert = cert

	return nil
}

// loadCA loads the existing CA certificate
func (cm *CertificateManager) loadCA() error {
	// Load certificate
	certPEM, err := os.ReadFile(cm.certPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Load private key
	keyPEM, err := os.ReadFile(cm.keyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("failed to decode private key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	cm.caKey = key
	cm.caCert = cert

	return nil
}

// GenerateServerCertificate generates a certificate for a specific domain
func (cm *CertificateManager) GenerateServerCertificate(domain string) (*tls.Certificate, error) {
	cm.mu.RLock()
	if cert, ok := cm.certCache[domain]; ok {
		cm.mu.RUnlock()
		return cert, nil
	}
	cm.mu.RUnlock()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Double-check after acquiring write lock
	if cert, ok := cm.certCache[domain]; ok {
		return cert, nil
	}

	// Generate RSA key pair for server
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate server key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // 1 year

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:         domain,
			Country:            []string{"US"},
			Organization:       []string{"KProxy"},
			OrganizationalUnit: []string{"KProxy Proxy Server"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{domain},
	}

	// Create certificate signed by CA
	certDER, err := x509.CreateCertificate(rand.Reader, &template, cm.caCert, &key.PublicKey, cm.caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create server certificate: %w", err)
	}

	// Create tls.Certificate
	cert := &tls.Certificate{
		Certificate: [][]byte{certDER, cm.caCert.Raw},
		PrivateKey:  key,
	}

	// Cache certificate (implement simple LRU by clearing cache when full)
	if len(cm.certCache) >= cm.cacheMaxSize {
		// Simple cache eviction - clear half the cache
		for k := range cm.certCache {
			delete(cm.certCache, k)
			if len(cm.certCache) < cm.cacheMaxSize/2 {
				break
			}
		}
	}

	cm.certCache[domain] = cert

	return cert, nil
}

// GetCACertificatePath returns the path to the CA certificate
func (cm *CertificateManager) GetCACertificatePath() string {
	return cm.certPath
}

// GetCACertificate returns the CA certificate for use in TLS config
func (cm *CertificateManager) GetCACertificate() *tls.Certificate {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.caKey == nil || cm.caCert == nil {
		return nil
	}

	return &tls.Certificate{
		Certificate: [][]byte{cm.caCert.Raw},
		PrivateKey:  cm.caKey,
	}
}
