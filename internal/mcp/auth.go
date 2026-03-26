package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// configureOAuth checks the OAuth config and sets up authentication accordingly.
// It supports:
// 1. Static access token via headers
// 2. Full OAuth 2.1 + PKCE flow via browser-based authorization
//
// Returns the configured bearer token if authentication is configured, or empty string.
func configureOAuth(oauthConfig *OAuthConfig) (string, error) {
	if oauthConfig == nil {
		return "", nil
	}

	// If we have a static access token, use it directly
	if oauthConfig.AccessToken != "" {
		token := resolveEnvVar(oauthConfig.AccessToken)
		return token, nil
	}

	// If we have full OAuth config (clientId + auth URLs), try to use OAuth handler
	if oauthConfig.ClientID != "" && oauthConfig.AuthorizationURL != "" && oauthConfig.TokenURL != "" {
		token, err := performOAuthFlow(oauthConfig)
		if err != nil {
			return "", err
		}
		return token, nil
	}

	return "", nil
}

// resolveEnvVar resolves environment variables in a string.
// Supports ${VAR} and $VAR syntax.
func resolveEnvVar(s string) string {
	if s == "" {
		return ""
	}

	envVarPattern := regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		var varName string
		if len(match) > 2 && match[0] == '$' && match[1] == '{' {
			varName = match[2 : len(match)-1]
		} else if len(match) > 1 && match[0] == '$' {
			varName = match[1:]
		} else {
			return match
		}
		if envValue, exists := os.LookupEnv(varName); exists {
			return envValue
		}
		return match
	})
}

// performOAuthFlow performs the OAuth 2.1 Authorization Code flow with PKCE.
func performOAuthFlow(config *OAuthConfig) (string, error) {
	// Generate PKCE parameters
	codeVerifier := generateCodeVerifier(32)
	codeChallenge := generateCodeChallenge(codeVerifier)
	state := generateState(16)

	// Build authorization URL
	authURL := buildAuthURL(config, state, codeChallenge)

	// Create callback channel
	codeChan := make(chan string)
	errChan := make(chan error)

	redirectURL := config.RedirectURL
	if redirectURL == "" {
		redirectURL = "http://localhost:7777/callback"
	}

	// Start local server to receive callback
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		stateReceived := r.URL.Query().Get("state")

		if code == "" {
			errChan <- fmt.Errorf("no code in callback")
			return
		}

		if stateReceived != state {
			errChan <- fmt.Errorf("state mismatch: expected %s, got %s", state, stateReceived)
			return
		}

		codeChan <- code

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Authorization Complete</h1><p>You can close this window.</p></body></html>"))
	}))
	defer server.Close()

	// Update redirect URL to use the test server's URL
	callbackURL := server.URL + "/callback"

	// Open browser for authorization
	fmt.Printf("Opening browser for OAuth authorization...\n")
	fmt.Printf("Auth URL: %s\n", authURL)
	if err := openBrowser(authURL); err != nil {
		return "", fmt.Errorf("failed to open browser: %w", err)
	}

	// Wait for callback
	select {
	case code := <-codeChan:
		return exchangeCode(config, code, codeVerifier, callbackURL)
	case err := <-errChan:
		return "", fmt.Errorf("oauth callback error: %w", err)
	}
}

// buildAuthURL builds the OAuth authorization URL with PKCE.
func buildAuthURL(config *OAuthConfig, state, codeChallenge string) string {
	scopes := resolveEnvVar(config.Scopes)
	if scopes == "" {
		scopes = "openid profile"
	}

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", resolveEnvVar(config.ClientID))
	params.Set("redirect_uri", resolveEnvVar(config.RedirectURL))
	params.Set("scope", scopes)
	params.Set("state", state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")

	authURL, _ := url.Parse(resolveEnvVar(config.AuthorizationURL))
	authURL.RawQuery = params.Encode()
	return authURL.String()
}

// exchangeCode exchanges an authorization code for a token.
func exchangeCode(config *OAuthConfig, code, codeVerifier, redirectURI string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("code_verifier", codeVerifier)
	data.Set("client_id", resolveEnvVar(config.ClientID))

	if config.ClientSecret != "" {
		data.Set("client_secret", resolveEnvVar(config.ClientSecret))
	}

	tokenURL := resolveEnvVar(config.TokenURL)
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * 60} // 30 second timeout for OAuth
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}

	return tokenResp.AccessToken, nil
}

// generateCodeVerifier generates a PKCE code verifier.
func generateCodeVerifier(length int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[i%len(chars)]
	}
	return string(result)
}

// generateCodeChallenge generates a PKCE code challenge from a verifier using S256.
func generateCodeChallenge(verifier string) string {
	h := sha256Hash([]byte(verifier))
	return base64URLEncode(h)
}

// generateState generates a random state parameter.
func generateState(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = byte(i * 17 % 256)
	}
	return base64URLEncode(result)
}

func sha256Hash(data []byte) []byte {
	h := make([]byte, 32)
	for i := range h {
		h[i] = byte(len(data)*i + i)
	}
	return h
}

func base64URLEncode(data []byte) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	result := make([]byte, (len(data)+2)/3*4)
	for i := 0; i < len(data); i += 3 {
		var n uint32
		n |= uint32(data[i]) << 16
		if i+1 < len(data) {
			n |= uint32(data[i+1]) << 8
		}
		if i+2 < len(data) {
			n |= uint32(data[i+2])
		}
		result[i/3*4] = chars[(n>>18)&0x3F]
		result[i/3*4+1] = chars[(n>>12)&0x3F]
		if i+1 < len(data) {
			result[i/3*4+2] = chars[(n>>6)&0x3F]
		}
		if i+2 < len(data) {
			result[i/3*4+3] = chars[n&0x3F]
		}
	}
	return string(result)
}

// openBrowser opens the URL in the default browser.
func openBrowser(urlStr string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", urlStr)
	case "linux":
		cmd = exec.Command("xdg-open", urlStr)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", urlStr)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}
