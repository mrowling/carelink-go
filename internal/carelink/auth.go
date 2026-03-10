package carelink

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mrowling/carelink-go/internal/paths"
	"github.com/mrowling/carelink-go/internal/types"
)

// LoadLoginData reads logindata.json from configured locations
func LoadLoginData() (*types.LoginData, error) {
	// Try to find logindata.json (checks current dir, then ~/.carelink/)
	path, err := paths.FindFile("logindata.json")
	if err != nil {
		return nil, fmt.Errorf("logindata.json not found - see README for authentication setup: %w", err)
	}

	log.Printf("[Auth] Loading logindata.json from %s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read logindata.json: %w", err)
	}

	var loginData types.LoginData
	if err := json.Unmarshal(data, &loginData); err != nil {
		return nil, fmt.Errorf("failed to parse logindata.json: %w", err)
	}

	// Validate required fields
	if loginData.AccessToken == "" {
		return nil, fmt.Errorf("logindata.json missing access_token")
	}
	if loginData.RefreshToken == "" {
		return nil, fmt.Errorf("logindata.json missing refresh_token")
	}
	if loginData.ClientID == "" {
		return nil, fmt.Errorf("logindata.json missing client_id")
	}
	if loginData.TokenURL == "" {
		return nil, fmt.Errorf("logindata.json missing token_url")
	}

	return &loginData, nil
}

// SaveLoginData writes logindata.json to the same location it was loaded from
func SaveLoginData(loginData *types.LoginData) error {
	// Try to find existing logindata.json
	path, err := paths.FindFile("logindata.json")
	if err != nil {
		// If not found, save to ~/.carelink/
		configDir, err := paths.GetConfigDir()
		if err != nil {
			return fmt.Errorf("failed to get config directory: %w", err)
		}
		path = configDir + "/logindata.json"
		log.Printf("[Auth] Saving new logindata.json to %s", path)
	} else {
		log.Printf("[Auth] Updating logindata.json at %s", path)
	}

	data, err := json.MarshalIndent(loginData, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal logindata: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write logindata.json: %w", err)
	}

	return nil
}

// IsTokenExpired checks if the JWT access token is expired
func IsTokenExpired(accessToken string) bool {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return true
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Try standard base64
		payload, err = base64.RawStdEncoding.DecodeString(parts[1])
		if err != nil {
			log.Printf("[Token] Failed to decode JWT: %v", err)
			return true
		}
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		log.Printf("[Token] Failed to parse JWT claims: %v", err)
		return true
	}

	if claims.Exp == 0 {
		return true
	}

	// Expired if less than 1 minute remaining
	return claims.Exp*1000 < time.Now().UnixMilli()+60*1000
}

// RefreshToken refreshes the access token using the refresh token
func RefreshToken(loginData *types.LoginData) error {
	log.Println("[Token] Refreshing access token...")

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", loginData.ClientID)
	form.Set("refresh_token", loginData.RefreshToken)

	req, err := http.NewRequest("POST", loginData.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp types.TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	loginData.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		loginData.RefreshToken = tokenResp.RefreshToken
	}

	log.Println("[Token] Token refreshed successfully")
	return nil
}

// Authenticate loads and refreshes tokens if needed
func Authenticate() (*types.LoginData, error) {
	loginData, err := LoadLoginData()
	if err != nil {
		return nil, fmt.Errorf("no logindata.json found - run 'npm run login' first: %w", err)
	}

	if IsTokenExpired(loginData.AccessToken) {
		if err := RefreshToken(loginData); err != nil {
			// Try to delete stale logindata
			if path, err := paths.FindFile("logindata.json"); err == nil {
				_ = os.Remove(path)
				log.Printf("[Token] Deleted stale logindata.json from %s", path)
			}
			log.Println("[Token] Refresh token expired - see README for re-authentication steps")
			return nil, fmt.Errorf("refresh token expired - authentication required")
		}

		// Save updated tokens
		if err := SaveLoginData(loginData); err != nil {
			log.Printf("[Token] Warning: failed to save updated tokens: %v", err)
		}
	}

	log.Println("[Token] Using token-based auth from logindata.json")
	return loginData, nil
}

// MakeAuthRequest makes an authenticated HTTP request with Bearer token
func MakeAuthRequest(method, url string, body []byte, accessToken string, headers map[string]string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	// Note: Don't set Accept-Encoding manually - Go's http.Client handles gzip automatically
	req.Header.Set("Connection", "keep-alive")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects automatically
		},
	}

	return client.Do(req)
}
