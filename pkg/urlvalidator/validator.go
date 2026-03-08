package urlvalidator

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateExternalURL checks that a URL is safe to make requests to.
// It blocks private IPs, localhost, and cloud metadata endpoints.
func ValidateExternalURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("URL tidak valid: %w", err)
	}

	// Only allow HTTPS (and HTTP for development flexibility)
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("URL harus menggunakan HTTP atau HTTPS")
	}

	host := parsed.Hostname()

	// Block localhost/loopback
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "0.0.0.0" {
		return fmt.Errorf("URL tidak boleh mengarah ke localhost")
	}

	// Block cloud metadata endpoints
	if host == "169.254.169.254" || host == "metadata.google.internal" {
		return fmt.Errorf("URL tidak boleh mengarah ke cloud metadata")
	}

	// Block private IP ranges
	ip := net.ParseIP(host)
	if ip != nil && (ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()) {
		return fmt.Errorf("URL tidak boleh mengarah ke jaringan internal")
	}

	// Block common internal hostnames
	lower := strings.ToLower(host)
	if strings.HasSuffix(lower, ".internal") || strings.HasSuffix(lower, ".local") {
		return fmt.Errorf("URL tidak boleh mengarah ke jaringan internal")
	}

	return nil
}
