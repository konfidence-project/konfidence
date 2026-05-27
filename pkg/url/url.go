package url

import (
	"net/url"
	"strings"
)

const (
	WwwPrefix = "www."
)

// ExtractHostnameWithOptionalPort extracts the hostname with port number from a url
func ExtractHostnameWithOptionalPort(rawURL string) (string, error) {
	u, err := parseUrl(rawURL)
	if err != nil {
		return "", err
	}

	host := u.Host
	host = strings.TrimPrefix(host, WwwPrefix)
	return host, nil
}

// ExtractHostname extracts the hostname without port number from a url
func ExtractHostname(rawURL string) (string, error) {
	u, err := parseUrl(rawURL)
	if err != nil {
		return "", err
	}

	host := u.Hostname()
	host = strings.TrimPrefix(host, WwwPrefix)
	return host, nil
}

func parseUrl(rawURL string) (*url.URL, error) {
	if !strings.Contains(rawURL, "://") {
		rawURL = "//" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	return u, nil
}
