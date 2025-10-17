package proxy

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/1342tools/kanti/backend/pkg/models"
	"github.com/andybalholm/brotli"
	"github.com/elazarl/goproxy"
)

const (
	MaxBodySize       = 10 * 1024 * 1024 // 10MB
	MaxCachedRequests = 1000
	BatchSize         = 50
	BatchInterval     = 100 * time.Millisecond
)

// ProxyServer implements the HTTP/HTTPS proxy with SSL interception
type ProxyServer struct {
	proxy    *goproxy.ProxyHttpServer
	certMgr  *CertificateManager
	config   *models.ProxyConfig
	listener net.Listener

	// Request tracking
	requestID  int64
	reqCache   []models.RequestDetails
	cacheMu    sync.RWMutex
	cacheHead  int
	cacheTail  int
	cacheCount int

	// Batching
	reqBatch   []models.RequestDetails
	respBatch  []models.RequestDetails
	batchMu    sync.Mutex
	batchTimer *time.Timer

	// Event callbacks
	onRequest    func(models.RequestDetails)
	onResponse   func(models.RequestDetails)
	onBatchFlush func([]models.RequestDetails, []models.RequestDetails)

	// Server state
	isRunning bool
	mu        sync.RWMutex
}

// NewProxyServer creates a new proxy server instance
func NewProxyServer(dataDir string, config *models.ProxyConfig) (*ProxyServer, error) {
	certMgr, err := NewCertificateManager(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate manager: %w", err)
	}

	if config == nil {
		config = &models.ProxyConfig{
			Port:            8080,
			SSLInterception: true,
			CustomHeaders:   make(map[string]string),
			InScope:         []string{},
			OutOfScope:      []string{},
		}
	}

	// Store certificate path in config
	config.CertPath = certMgr.GetCACertificatePath()

	ps := &ProxyServer{
		proxy:      goproxy.NewProxyHttpServer(),
		certMgr:    certMgr,
		config:     config,
		reqCache:   make([]models.RequestDetails, MaxCachedRequests),
		cacheHead:  0,
		cacheTail:  0,
		cacheCount: 0,
	}

	// Configure proxy
	ps.proxy.Verbose = false

	// Set up SSL interception if enabled
	if config.SSLInterception {
		ps.setupSSLInterception()
	}

	// Set up request/response handlers
	ps.setupHandlers()

	return ps, nil
}

// setupSSLInterception configures SSL/TLS MITM
func (ps *ProxyServer) setupSSLInterception() {
	// Set up CA for MITM
	caCert := ps.certMgr.GetCACertificate()
	if caCert == nil {
		log.Println("Warning: CA certificate not available for SSL interception")
		return
	}

	// Configure MITM handler with certificate generation
	ps.proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	// Set custom certificate generation function
	goproxy.GoproxyCa = *caCert
	goproxy.MitmConnect = &goproxy.ConnectAction{
		Action:    goproxy.ConnectMitm,
		TLSConfig: ps.generateTLSConfig,
	}
}

// generateTLSConfig generates TLS config with dynamic certificate generation
func (ps *ProxyServer) generateTLSConfig(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
	// Extract hostname without port
	hostname := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostname = h
	}

	// Generate certificate for this hostname
	cert, err := ps.certMgr.GenerateServerCertificate(hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate for %s: %w", hostname, err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{*cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// setupHandlers configures request and response interceptors
func (ps *ProxyServer) setupHandlers() {
	// Request handler - intercept all requests
	ps.proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		startTime := time.Now()
		reqID := atomic.AddInt64(&ps.requestID, 1)

		// Store request start time in context for response handler
		ctx.UserData = map[string]interface{}{
			"startTime": startTime,
			"reqID":     reqID,
		}

		// Sanitize and add custom headers
		ps.sanitizeHeaders(req)
		ps.addCustomHeaders(req)

		// Capture request details
		details := ps.captureRequest(req, reqID, startTime)

		// Check scope and emit request
		if ps.shouldSave(details.Host) {
			ps.emitRequest(details)
		}

		return req, nil
	})

	// Response handler - intercept all responses
	ps.proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if resp == nil || ctx.UserData == nil {
			return resp
		}

		userData, ok := ctx.UserData.(map[string]interface{})
		if !ok {
			return resp
		}

		startTime, _ := userData["startTime"].(time.Time)
		reqID, _ := userData["reqID"].(int64)

		// Capture response details
		details := ps.captureResponse(ctx.Req, resp, reqID, startTime)

		// Check scope and emit response
		if ps.shouldSave(details.Host) {
			ps.emitResponse(details)
		}

		return resp
	})
}

// captureRequest captures request details
func (ps *ProxyServer) captureRequest(req *http.Request, reqID int64, startTime time.Time) models.RequestDetails {
	protocol := "http"
	if req.TLS != nil {
		protocol = "https"
	}

	// Read and buffer request body
	var body string
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(io.LimitReader(req.Body, MaxBodySize))
		if err == nil && len(bodyBytes) > 0 {
			body = string(bodyBytes)
			// Restore body for forwarding
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}

	return models.RequestDetails{
		ID:        int(reqID),
		Host:      req.Host,
		Method:    req.Method,
		Path:      req.URL.Path,
		Query:     req.URL.RawQuery,
		Headers:   req.Header.Clone(),
		Timestamp: startTime,
		Protocol:  protocol,
		Body:      body,
	}
}

// captureResponse captures response details
func (ps *ProxyServer) captureResponse(req *http.Request, resp *http.Response, reqID int64, startTime time.Time) models.RequestDetails {
	responseTime := time.Since(startTime).Milliseconds()

	protocol := "http"
	if req.TLS != nil {
		protocol = "https"
	}

	// Read and decompress response body
	var responseBody string
	var contentLength int

	if resp.Body != nil && shouldCaptureBody(resp.Header.Get("Content-Type")) {
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, MaxBodySize))
		if err == nil {
			contentLength = len(bodyBytes)

			// Decompress if needed
			decompressed, err := decompressResponse(bodyBytes, resp.Header.Get("Content-Encoding"))
			if err == nil {
				responseBody = string(decompressed)
			}

			// Restore body for client
			resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}

	return models.RequestDetails{
		ID:              int(reqID),
		Host:            req.Host,
		Method:          req.Method,
		Path:            req.URL.Path,
		Query:           req.URL.RawQuery,
		Headers:         req.Header.Clone(),
		Timestamp:       startTime,
		ResponseLength:  contentLength,
		Status:          resp.StatusCode,
		ResponseTime:    responseTime,
		Protocol:        protocol,
		ResponseHeaders: resp.Header.Clone(),
		ResponseBody:    responseBody,
	}
}

// decompressResponse decompresses response data based on encoding
func decompressResponse(data []byte, encoding string) ([]byte, error) {
	if encoding == "" {
		return data, nil
	}

	encoding = strings.ToLower(encoding)

	if strings.Contains(encoding, "gzip") {
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return data, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	} else if strings.Contains(encoding, "br") {
		reader := brotli.NewReader(bytes.NewReader(data))
		return io.ReadAll(reader)
	} else if strings.Contains(encoding, "deflate") {
		// Deflate is more complex - try both raw and zlib
		// For simplicity, return as-is (can enhance later)
		return data, nil
	}

	return data, nil
}

// shouldCaptureBody checks if body should be captured based on content type
func shouldCaptureBody(contentType string) bool {
	if contentType == "" {
		return true
	}

	contentType = strings.ToLower(contentType)
	textTypes := []string{
		"text/",
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-www-form-urlencoded",
		"application/graphql",
	}

	for _, t := range textTypes {
		if strings.Contains(contentType, t) {
			return true
		}
	}

	return false
}

// sanitizeHeaders removes proxy-revealing headers and adds browser-like headers
func (ps *ProxyServer) sanitizeHeaders(req *http.Request) {
	// Remove proxy-revealing headers
	proxyHeaders := []string{
		"X-Forwarded-For",
		"X-Forwarded-Host",
		"X-Forwarded-Proto",
		"X-Real-IP",
		"Via",
		"Forwarded",
		"Proxy-Connection",
	}

	for _, h := range proxyHeaders {
		req.Header.Del(h)
	}

	// Add realistic browser headers if missing
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	}

	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	}

	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	}

	if req.Header.Get("Accept-Encoding") == "" {
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	}
}

// addCustomHeaders adds custom headers to the request
func (ps *ProxyServer) addCustomHeaders(req *http.Request) {
	for key, value := range ps.config.CustomHeaders {
		req.Header.Set(key, value)
	}
}

// shouldSave checks if request should be saved based on scope
func (ps *ProxyServer) shouldSave(host string) bool {
	if !ps.config.SaveOnlyInScope {
		return true
	}

	// Check out-of-scope first (exclusions take precedence)
	for _, pattern := range ps.config.OutOfScope {
		if matchesPattern(host, pattern) {
			return false
		}
	}

	// Check in-scope patterns
	for _, pattern := range ps.config.InScope {
		if matchesPattern(host, pattern) {
			return true
		}
	}

	return false
}

// matchesPattern checks if host matches a scope pattern (supports wildcards)
func matchesPattern(host, pattern string) bool {
	if pattern == host {
		return true
	}

	// Wildcard match (e.g., *.example.com)
	if strings.HasPrefix(pattern, "*.") {
		domain := pattern[2:]
		return strings.HasSuffix(host, domain) || host == domain
	}

	return false
}

// emitRequest emits a request event and adds to batch
func (ps *ProxyServer) emitRequest(details models.RequestDetails) {
	// Skip CONNECT requests
	if details.Method == "CONNECT" {
		return
	}

	// Add to cache
	ps.addToCache(details)

	// Add to batch
	ps.batchMu.Lock()
	ps.reqBatch = append(ps.reqBatch, details)
	shouldFlush := len(ps.reqBatch) >= BatchSize
	ps.batchMu.Unlock()

	if shouldFlush {
		ps.flushBatches()
	} else {
		ps.scheduleBatchFlush()
	}

	// Call callback if set
	if ps.onRequest != nil {
		ps.onRequest(details)
	}
}

// emitResponse emits a response event and adds to batch
func (ps *ProxyServer) emitResponse(details models.RequestDetails) {
	// Skip CONNECT responses
	if details.Method == "CONNECT" {
		return
	}

	// Update in cache
	ps.updateInCache(details)

	// Add to batch
	ps.batchMu.Lock()
	ps.respBatch = append(ps.respBatch, details)
	shouldFlush := len(ps.respBatch) >= BatchSize
	ps.batchMu.Unlock()

	if shouldFlush {
		ps.flushBatches()
	} else {
		ps.scheduleBatchFlush()
	}

	// Call callback if set
	if ps.onResponse != nil {
		ps.onResponse(details)
	}
}

// scheduleBatchFlush schedules a batch flush if not already scheduled
func (ps *ProxyServer) scheduleBatchFlush() {
	ps.batchMu.Lock()
	defer ps.batchMu.Unlock()

	if ps.batchTimer == nil {
		ps.batchTimer = time.AfterFunc(BatchInterval, ps.flushBatches)
	}
}

// flushBatches sends batched requests and responses
func (ps *ProxyServer) flushBatches() {
	ps.batchMu.Lock()

	// Stop timer if active
	if ps.batchTimer != nil {
		ps.batchTimer.Stop()
		ps.batchTimer = nil
	}

	// Get batches
	reqBatch := ps.reqBatch
	respBatch := ps.respBatch

	// Reset batches
	ps.reqBatch = nil
	ps.respBatch = nil

	ps.batchMu.Unlock()

	// Send batches via callback
	if ps.onBatchFlush != nil && (len(reqBatch) > 0 || len(respBatch) > 0) {
		ps.onBatchFlush(reqBatch, respBatch)
	}
}

// addToCache adds a request to the circular buffer cache
func (ps *ProxyServer) addToCache(req models.RequestDetails) {
	ps.cacheMu.Lock()
	defer ps.cacheMu.Unlock()

	ps.reqCache[ps.cacheTail] = req
	ps.cacheTail = (ps.cacheTail + 1) % MaxCachedRequests

	if ps.cacheCount < MaxCachedRequests {
		ps.cacheCount++
	} else {
		ps.cacheHead = (ps.cacheHead + 1) % MaxCachedRequests
	}
}

// updateInCache updates a request in cache with response details
func (ps *ProxyServer) updateInCache(resp models.RequestDetails) {
	ps.cacheMu.Lock()
	defer ps.cacheMu.Unlock()

	for i := 0; i < ps.cacheCount; i++ {
		idx := (ps.cacheHead + i) % MaxCachedRequests
		if ps.reqCache[idx].ID == resp.ID {
			ps.reqCache[idx] = resp
			break
		}
	}
}

// GetRequests returns all cached requests (newest first)
func (ps *ProxyServer) GetRequests() []models.RequestDetails {
	ps.cacheMu.RLock()
	defer ps.cacheMu.RUnlock()

	result := make([]models.RequestDetails, 0, ps.cacheCount)

	// Read from tail backwards to get newest first
	for i := ps.cacheCount - 1; i >= 0; i-- {
		idx := (ps.cacheHead + i) % MaxCachedRequests
		result = append(result, ps.reqCache[idx])
	}

	return result
}

// ClearRequests clears the request cache
func (ps *ProxyServer) ClearRequests() {
	ps.cacheMu.Lock()
	defer ps.cacheMu.Unlock()

	ps.cacheHead = 0
	ps.cacheTail = 0
	ps.cacheCount = 0
}

// Start starts the proxy server
func (ps *ProxyServer) Start() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.isRunning {
		return fmt.Errorf("proxy server already running")
	}

	addr := fmt.Sprintf(":%d", ps.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start proxy server: %w", err)
	}

	ps.listener = listener
	ps.isRunning = true

	go func() {
		log.Printf("Proxy server listening on %s\n", addr)
		if err := http.Serve(listener, ps.proxy); err != nil {
			if ps.isRunning {
				log.Printf("Proxy server error: %v\n", err)
			}
		}
	}()

	return nil
}

// Stop stops the proxy server
func (ps *ProxyServer) Stop() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if !ps.isRunning {
		return fmt.Errorf("proxy server not running")
	}

	// Flush any pending batches
	ps.flushBatches()

	// Close listener
	if ps.listener != nil {
		if err := ps.listener.Close(); err != nil {
			return fmt.Errorf("failed to stop proxy server: %w", err)
		}
	}

	ps.isRunning = false
	log.Println("Proxy server stopped")

	return nil
}

// IsRunning returns whether the proxy server is running
func (ps *ProxyServer) IsRunning() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.isRunning
}

// GetStatus returns the current proxy status
func (ps *ProxyServer) GetStatus() models.ProxyStatus {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return models.ProxyStatus{
		IsRunning:       ps.isRunning,
		Port:            ps.config.Port,
		CertificatePath: ps.config.CertPath,
	}
}

// UpdateConfig updates the proxy configuration
func (ps *ProxyServer) UpdateConfig(config *models.ProxyConfig) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.config = config
}

// GetConfig returns the current configuration
func (ps *ProxyServer) GetConfig() *models.ProxyConfig {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.config
}

// SetOnRequest sets the callback for request events
func (ps *ProxyServer) SetOnRequest(callback func(models.RequestDetails)) {
	ps.onRequest = callback
}

// SetOnResponse sets the callback for response events
func (ps *ProxyServer) SetOnResponse(callback func(models.RequestDetails)) {
	ps.onResponse = callback
}

// SetOnBatchFlush sets the callback for batch flush events
func (ps *ProxyServer) SetOnBatchFlush(callback func([]models.RequestDetails, []models.RequestDetails)) {
	ps.onBatchFlush = callback
}

// GetCertificatePath returns the CA certificate path
func (ps *ProxyServer) GetCertificatePath() string {
	return ps.certMgr.GetCACertificatePath()
}
