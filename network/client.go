package network

import (
	"crypto/tls"
	"net/http"
	"sync"
	"time"
)

// BaseURL is the server address — must use IP 144.34.181.64 with HTTPS and port 8443.
// All API paths use the /api/v1/ prefix per FRONTEND_API_GUIDE.md.
const BaseURL = "https://144.34.181.64:8443/api/v1"

type inFlightCall struct {
	wg  sync.WaitGroup
	val []byte
	err error
}

// APIClient holds the HTTP client and auth state.
type APIClient struct {
	client     *http.Client
	token      string
	cache      *lruCache
	inFlight   map[string]*inFlightCall
	inFlightMu sync.Mutex
}

// NewAPIClient creates a new API client with explicit timeout, connection pool optimization, and safe transport.
// TLS verification is skipped because the server uses a CloudFlare origin certificate
// that does not include the IP address 144.34.181.64 in its SANs.
func NewAPIClient() *APIClient {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		MaxConnsPerHost:     50,
		IdleConnTimeout:     90 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, // IP cert workaround
	}
	return &APIClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second, // 15s request timeout per optimization requirement
		},
		token:    "",
		cache:    newLRUCache(200, 60*time.Second), // LRU cache capacity 200, TTL 60s
		inFlight: make(map[string]*inFlightCall),
	}
}

// CloseIdleConnections recycles idle connections in the HTTP transport (6.3.2).
func (c *APIClient) CloseIdleConnections() {
	if tr, ok := c.client.Transport.(*http.Transport); ok {
		tr.CloseIdleConnections()
	}
}

// SetToken sets the authentication token.
func (c *APIClient) SetToken(token string) {
	c.token = token
}

// GetToken returns the current token.
func (c *APIClient) GetToken() string {
	return c.token
}
