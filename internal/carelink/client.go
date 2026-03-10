package carelink

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/mrowling/carelink-go/internal/config"
	"github.com/mrowling/carelink-go/internal/paths"
	"github.com/mrowling/carelink-go/internal/types"
)

const (
	userAgent           = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	maxRequestsPerFetch = 30
)

// Client is the main CareLink API client
type Client struct {
	httpClient   *http.Client
	proxyRotator *ProxyRotator
	urls         *URLs
	config       *config.Config
	loginData    *types.LoginData
	requestCount int
}

// NewClient creates a new CareLink client
func NewClient(cfg *config.Config) (*Client, error) {
	serverName := ResolveServerName(cfg.Server, "")
	urls := NewURLs(serverName, cfg.CountryCode, cfg.Language)

	// Load proxy list if enabled
	var proxies []Proxy
	if cfg.UseProxy {
		// Try to find https.txt (checks current dir, then ~/.carelink/)
		proxyFile, err := paths.FindFile("https.txt")
		if err != nil {
			log.Printf("[Proxy] Proxy file not found: %v", err)
			proxyFile = "" // Will result in empty proxy list
		}
		proxies = LoadProxyList(proxyFile)
	}
	proxyRotator := NewProxyRotator(proxies, 10)

	// Create HTTP client
	transport := &http.Transport{}
	if proxyRotator.HasProxies() {
		proxy := proxyRotator.GetNext()
		if proxy != nil {
			transport = CreateProxyTransport(proxy)
			log.Printf("[Proxy] Using proxy: %s:%s%s", proxy.IP, proxy.Port,
				func() string {
					if proxy.Username != "" {
						return " (authenticated)"
					}
					return ""
				}())
		}
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &Client{
		httpClient:   httpClient,
		proxyRotator: proxyRotator,
		urls:         urls,
		config:       cfg,
	}, nil
}

func (c *Client) applyProxy(proxy *Proxy) {
	if proxy != nil {
		transport := CreateProxyTransport(proxy)
		c.httpClient.Transport = transport
		log.Printf("[Proxy] Using proxy: %s:%s%s", proxy.IP, proxy.Port,
			func() string {
				if proxy.Username != "" {
					return " (authenticated)"
				}
				return ""
			}())
	} else {
		c.httpClient.Transport = &http.Transport{}
	}
}

func (c *Client) authenticate() error {
	loginData, err := Authenticate()
	if err != nil {
		return err
	}
	c.loginData = loginData
	return nil
}

func (c *Client) makeRequest(method, url string, body []byte) (*http.Response, error) {
	c.requestCount++
	if c.requestCount > maxRequestsPerFetch {
		return nil, fmt.Errorf("request count exceeds maximum (%d) in one fetch", maxRequestsPerFetch)
	}

	headers := map[string]string{}
	if body != nil {
		headers["Content-Type"] = "application/json"
	}

	return MakeAuthRequest(method, url, body, c.loginData.AccessToken, headers)
}

func (c *Client) getCurrentRole() (string, error) {
	resp, err := c.makeRequest("GET", c.urls.Me, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get user info: HTTP %d", resp.StatusCode)
	}

	var userInfo types.UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", fmt.Errorf("failed to decode user info: %w", err)
	}

	return strings.ToUpper(userInfo.Role), nil
}

func (c *Client) isBLEDevice(deviceFamily string) bool {
	if deviceFamily == "" {
		return false
	}
	upper := strings.ToUpper(deviceFamily)
	return strings.Contains(upper, "BLE") || strings.Contains(upper, "SIMPLERA")
}

func (c *Client) fetchBLEDeviceData(patientID, role string) (*types.CareLinkData, error) {
	if c.config.Verbose {
		log.Println("[Client] Fetching BLE device data")
	}

	// Get country settings for BLE endpoint
	resp, err := c.makeRequest("GET", c.urls.CountrySettings, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var settings types.CountrySettings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("failed to decode country settings: %w", err)
	}

	if settings.BLEPeriodicDataEndpoint == "" {
		return nil, fmt.Errorf("no BLE endpoint found in country settings")
	}

	// Get patient ID if not provided
	if patientID == "" && role == "patient" {
		meResp, err := c.makeRequest("GET", c.urls.Me, nil)
		if err == nil {
			defer meResp.Body.Close()
			var userInfo types.UserInfo
			if json.NewDecoder(meResp.Body).Decode(&userInfo) == nil {
				patientID = userInfo.ID
			}
		}
	}

	// Build request body
	bodyMap := map[string]string{
		"username": c.config.Username,
		"role":     role,
	}
	if patientID != "" {
		bodyMap["patientId"] = patientID
	}

	bodyJSON, _ := json.Marshal(bodyMap)
	resp, err = c.makeRequest("POST", settings.BLEPeriodicDataEndpoint, bodyJSON)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BLE endpoint returned HTTP %d", resp.StatusCode)
	}

	var data types.CareLinkData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode BLE data: %w", err)
	}

	if c.config.Verbose {
		log.Printf("[Client] GET data (BLE) %s", settings.BLEPeriodicDataEndpoint)
	}

	return &data, nil
}

func (c *Client) fetchAsPatient() (*types.CareLinkData, error) {
	// Try monitor endpoint first (works for 7xxG pumps)
	resp, err := c.makeRequest("GET", c.urls.MonitorData, nil)
	if err == nil {
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var data types.CareLinkData
			bodyBytes, _ := io.ReadAll(resp.Body)
			if err := json.Unmarshal(bodyBytes, &data); err == nil {
				// Check if it's a BLE device
				if c.isBLEDevice(data.MedicalDeviceFamily) {
					if c.config.Verbose {
						log.Println("[Client] BLE device detected, using BLE endpoint")
					}
					return c.fetchBLEDeviceData("", "patient")
				}

				// Check if response has meaningful data
				if len(bodyBytes) > 50 {
					if c.config.Verbose {
						log.Printf("[Client] GET data %s", c.urls.MonitorData)
					}
					return &data, nil
				}
			}
		}
	}

	// Fall back to legacy connect endpoint
	url := c.urls.ConnectData(time.Now().UnixMilli())
	resp, err = c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("connect data endpoint returned HTTP %d", resp.StatusCode)
	}

	var data types.CareLinkData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode connect data: %w", err)
	}

	if c.config.Verbose {
		log.Printf("[Client] GET data %s", url)
	}

	return &data, nil
}

func (c *Client) fetchAsCarepartner(role string) (*types.CareLinkData, error) {
	patientID := c.config.PatientID

	// Get linked patients if no patient ID configured
	if patientID == "" {
		resp, err := c.makeRequest("GET", c.urls.LinkedPatients, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var patients []types.PatientLink
		if err := json.NewDecoder(resp.Body).Decode(&patients); err != nil {
			return nil, fmt.Errorf("failed to decode linked patients: %w", err)
		}

		if len(patients) == 0 {
			return nil, fmt.Errorf("no linked patients found for care partner account")
		}

		patientID = patients[0].Username
		if c.config.Verbose {
			log.Printf("[Client] Using linked patient: %s", patientID)
		}
	}

	// Check if patient has BLE device by fetching monitor data first
	resp, err := c.makeRequest("GET", c.urls.MonitorData, nil)
	if err == nil {
		defer resp.Body.Close()
		var data types.CareLinkData
		if json.NewDecoder(resp.Body).Decode(&data) == nil && c.isBLEDevice(data.MedicalDeviceFamily) {
			if c.config.Verbose {
				log.Println("[Client] BLE device detected for carepartner, using BLE endpoint")
			}
			return c.fetchBLEDeviceData(patientID, "carepartner")
		}
	}

	// Standard carepartner flow: Get country settings
	resp, err = c.makeRequest("GET", c.urls.CountrySettings, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var settings types.CountrySettings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("failed to decode country settings: %w", err)
	}

	dataRetrievalURL := settings.BLEPeriodicDataEndpoint
	if dataRetrievalURL == "" {
		return nil, fmt.Errorf("unable to retrieve data retrieval URL for care partner account")
	}

	if c.config.Verbose {
		log.Printf("[Client] Data retrieval URL: %s", dataRetrievalURL)
	}

	// Try multiple API versions
	endpoints := []string{
		dataRetrievalURL,
		strings.Replace(dataRetrievalURL, "/v6/", "/v5/", 1),
		strings.Replace(dataRetrievalURL, "/v6/", "/v11/", 1),
		strings.Replace(dataRetrievalURL, "/v5/", "/v6/", 1),
		strings.Replace(dataRetrievalURL, "/v5/", "/v11/", 1),
	}

	bodyMap := map[string]string{
		"username":  c.config.Username,
		"role":      "carepartner",
		"patientId": patientID,
	}
	bodyJSON, _ := json.Marshal(bodyMap)

	for _, endpoint := range endpoints {
		if c.config.Verbose {
			log.Printf("[Client] Trying carepartner endpoint: %s", endpoint)
		}

		resp, err := c.makeRequest("POST", endpoint, bodyJSON)
		if err != nil {
			if c.config.Verbose {
				log.Printf("[Client] Endpoint failed: %s - %v", endpoint, err)
			}
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var data types.CareLinkData
			if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
				if c.config.Verbose {
					log.Printf("[Client] GET data (as carepartner) %s", endpoint)
				}
				return &data, nil
			}
		}

		if c.config.Verbose {
			log.Printf("[Client] Endpoint failed: %s", endpoint)
		}
	}

	return nil, fmt.Errorf("all carepartner data endpoints failed")
}

func (c *Client) getConnectData() (*types.CareLinkData, error) {
	role, err := c.getCurrentRole()
	if err != nil {
		return nil, err
	}

	if c.config.Verbose {
		log.Printf("[Client] getConnectData - currentRole: %s", role)
	}

	if role == "CARE_PARTNER_OUS" || role == "CARE_PARTNER" {
		return c.fetchAsCarepartner(role)
	}

	return c.fetchAsPatient()
}

// Fetch retrieves data from CareLink with retry logic and proxy rotation
func (c *Client) Fetch() (*types.CareLinkData, error) {
	c.requestCount = 0
	c.proxyRotator.ResetRetries()

	maxRetry := 1
	if c.proxyRotator.HasProxies() {
		maxRetry = 10
	}

	log.Printf("[Fetch] Starting fetch, max retries: %d", maxRetry)

	for i := 1; i <= maxRetry; i++ {
		c.requestCount = 0

		// Authenticate
		if err := c.authenticate(); err != nil {
			return nil, err
		}

		// Fetch data
		data, err := c.getConnectData()
		if err == nil {
			log.Println("[Fetch] Success!")
			return data, nil
		}

		// Check if error is retryable
		httpStatus := 0
		if strings.Contains(err.Error(), "HTTP ") {
			_, _ = fmt.Sscanf(err.Error(), "%*s %d", &httpStatus)
		}

		isProxyError := httpStatus == 400 || httpStatus == 403 || httpStatus == 407 ||
			httpStatus == 502 || httpStatus == 503
		isNetworkError := strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "timeout") ||
			strings.Contains(err.Error(), "connection reset")

		log.Printf("[Fetch] Attempt %d failed: %v", i, err)

		if (isProxyError || isNetworkError) && c.proxyRotator.HasProxies() {
			log.Println("[Fetch] Trying next proxy...")
			nextProxy := c.proxyRotator.TryNext()
			if nextProxy == nil {
				return nil, fmt.Errorf("all proxies failed: %w", err)
			}
			c.applyProxy(nextProxy)
			time.Sleep(1 * time.Second)
			continue
		}

		if i == maxRetry {
			return nil, fmt.Errorf("fetch failed after %d attempts: %w", maxRetry, err)
		}

		// Exponential backoff
		timeout := time.Duration(math.Pow(2, float64(i))) * time.Second
		time.Sleep(timeout)
	}

	return nil, fmt.Errorf("fetch failed after all retries")
}
