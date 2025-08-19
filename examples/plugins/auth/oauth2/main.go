package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/forest6511/gdl/pkg/plugin"
)

// OAuth2Plugin implements authentication using OAuth2 flow
type OAuth2Plugin struct {
	clientID     string
	clientSecret string
	tokenURL     string
	scopes       []string
	tokenCache   *TokenCache
}

// TokenResponse represents the OAuth2 token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// TokenCache holds cached access tokens
type TokenCache struct {
	token     string
	expiresAt time.Time
}

// NewOAuth2Plugin creates a new OAuth2 authentication plugin
func NewOAuth2Plugin(clientID, clientSecret, tokenURL string, scopes []string) *OAuth2Plugin {
	return &OAuth2Plugin{
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     tokenURL,
		scopes:       scopes,
		tokenCache:   &TokenCache{},
	}
}

// Name returns the plugin name
func (p *OAuth2Plugin) Name() string {
	return "oauth2-auth"
}

// Version returns the plugin version
func (p *OAuth2Plugin) Version() string {
	return "1.0.0"
}

// Init initializes the OAuth2 plugin with configuration
func (p *OAuth2Plugin) Init(config map[string]interface{}) error {
	if clientID, ok := config["client_id"].(string); ok && clientID != "" {
		p.clientID = clientID
	}

	if clientSecret, ok := config["client_secret"].(string); ok && clientSecret != "" {
		p.clientSecret = clientSecret
	}

	if tokenURL, ok := config["token_url"].(string); ok && tokenURL != "" {
		p.tokenURL = tokenURL
	}

	if scopes, ok := config["scopes"].([]string); ok {
		p.scopes = scopes
	} else if scopesInterface, ok := config["scopes"].([]interface{}); ok {
		// Handle case where scopes come as []interface{}
		p.scopes = make([]string, len(scopesInterface))
		for i, scope := range scopesInterface {
			if scopeStr, ok := scope.(string); ok {
				p.scopes[i] = scopeStr
			}
		}
	}

	// Validate required fields
	if p.clientID == "" || p.clientSecret == "" || p.tokenURL == "" {
		return fmt.Errorf("client_id, client_secret, and token_url are required")
	}

	return nil
}

// Close cleans up the plugin resources
func (p *OAuth2Plugin) Close() error {
	// Clear cached token
	p.tokenCache.token = ""
	p.tokenCache.expiresAt = time.Time{}
	return nil
}

// ValidateAccess validates security access for operations
func (p *OAuth2Plugin) ValidateAccess(operation, resource string) error {
	// Allow basic auth operations
	allowedOps := []string{"authenticate", "authorize", "token", "refresh"}
	for _, op := range allowedOps {
		if operation == op {
			return nil
		}
	}
	return fmt.Errorf("operation %s not allowed for oauth2 plugin", operation)
}

// Authenticate adds OAuth2 authentication to the HTTP request
func (p *OAuth2Plugin) Authenticate(ctx context.Context, req *http.Request) error {
	token, err := p.getToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get OAuth2 token: %w", err)
	}

	// Add the Bearer token to the Authorization header
	req.Header.Set("Authorization", "Bearer "+token)

	return nil
}

// getToken retrieves a valid access token, using cache if available
func (p *OAuth2Plugin) getToken(ctx context.Context) (string, error) {
	// Check if we have a cached token that's still valid
	if p.tokenCache.token != "" && time.Now().Before(p.tokenCache.expiresAt) {
		return p.tokenCache.token, nil
	}

	// Get a new token
	token, err := p.requestToken(ctx)
	if err != nil {
		return "", err
	}

	return token.AccessToken, nil
}

// requestToken performs the OAuth2 client credentials flow
func (p *OAuth2Plugin) requestToken(ctx context.Context) (*TokenResponse, error) {
	// Prepare the token request
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)

	if len(p.scopes) > 0 {
		data.Set("scope", strings.Join(p.scopes, " "))
	}

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request token: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed with status: %d", resp.StatusCode)
	}

	// Parse the response
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Cache the token
	p.tokenCache.token = tokenResp.AccessToken

	// Calculate expiry time with some buffer (90% of actual expiry)
	expiryBuffer := time.Duration(float64(tokenResp.ExpiresIn) * 0.9)
	p.tokenCache.expiresAt = time.Now().Add(time.Second * expiryBuffer)

	return &tokenResp, nil
}

// IsTokenExpired checks if the current token is expired
func (p *OAuth2Plugin) IsTokenExpired() bool {
	return time.Now().After(p.tokenCache.expiresAt)
}

// RefreshToken manually refreshes the OAuth2 token
func (p *OAuth2Plugin) RefreshToken(ctx context.Context) error {
	_, err := p.requestToken(ctx)
	return err
}

// GetCachedToken returns the currently cached token (if any)
func (p *OAuth2Plugin) GetCachedToken() string {
	if p.IsTokenExpired() {
		return ""
	}
	return p.tokenCache.token
}

// Plugin variable to be loaded by the plugin system
var Plugin plugin.AuthPlugin = &OAuth2Plugin{}

func main() {
	// This is a plugin, so main() is not used when loaded as a shared library
	// But it can be useful for testing the plugin standalone
	fmt.Println("OAuth2 Authentication Plugin")
	fmt.Printf("Name: %s\n", Plugin.Name())
	fmt.Printf("Version: %s\n", Plugin.Version())
}
