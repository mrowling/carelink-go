package carelink

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Proxy represents a proxy server configuration
type Proxy struct {
	IP        string
	Port      string
	Username  string
	Password  string
	Protocols []string
}

// ProxyRotator manages rotation through a list of proxies
type ProxyRotator struct {
	proxies      []Proxy
	currentIndex int
	retryCount   int
	maxRetries   int
}

// LoadProxyList reads proxies from a file (one per line: ip:port or ip:port:user:pass)
func LoadProxyList(filePath string) []Proxy {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("[Proxy] Proxy file not found at: %s", filePath)
		return []Proxy{}
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("[Proxy] Failed to open proxy file: %v", err)
		return []Proxy{}
	}
	defer file.Close()

	var proxies []Proxy
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		proxy := Proxy{
			IP:        parts[0],
			Port:      parts[1],
			Protocols: []string{"http"},
		}

		if len(parts) >= 4 {
			proxy.Username = parts[2]
			proxy.Password = parts[3]
		}

		proxies = append(proxies, proxy)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[Proxy] Error reading proxy file: %v", err)
		return []Proxy{}
	}

	log.Printf("[Proxy] Loaded %d proxies from %s", len(proxies), filepath.Base(filePath))
	return proxies
}

// CreateProxyTransport creates an HTTP transport with proxy configuration
func CreateProxyTransport(proxy *Proxy) *http.Transport {
	if proxy == nil {
		return &http.Transport{}
	}

	auth := ""
	if proxy.Username != "" && proxy.Password != "" {
		auth = fmt.Sprintf("%s:%s@", proxy.Username, proxy.Password)
	}

	proxyURL, err := url.Parse(fmt.Sprintf("http://%s%s:%s", auth, proxy.IP, proxy.Port))
	if err != nil {
		log.Printf("[Proxy] Failed to create proxy URL: %v", err)
		return &http.Transport{}
	}

	return &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
}

// NewProxyRotator creates a new ProxyRotator
func NewProxyRotator(proxies []Proxy, maxRetries int) *ProxyRotator {
	if maxRetries == 0 {
		maxRetries = 10
	}
	return &ProxyRotator{
		proxies:    proxies,
		maxRetries: maxRetries,
	}
}

// HasProxies returns true if there are proxies configured
func (pr *ProxyRotator) HasProxies() bool {
	return len(pr.proxies) > 0
}

// GetNext returns the next proxy in rotation
func (pr *ProxyRotator) GetNext() *Proxy {
	if len(pr.proxies) == 0 {
		return nil
	}
	proxy := &pr.proxies[pr.currentIndex]
	pr.currentIndex = (pr.currentIndex + 1) % len(pr.proxies)
	return proxy
}

// TryNext increments retry counter and returns next proxy
func (pr *ProxyRotator) TryNext() *Proxy {
	if len(pr.proxies) == 0 {
		log.Println("[Proxy] No proxies available")
		return nil
	}
	pr.retryCount++
	if pr.retryCount > pr.maxRetries {
		log.Printf("[Proxy] Max proxy retries (%d) reached", pr.maxRetries)
		return nil
	}
	return pr.GetNext()
}

// ResetRetries resets the retry counter
func (pr *ProxyRotator) ResetRetries() {
	pr.retryCount = 0
}
