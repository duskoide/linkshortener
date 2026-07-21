package handlers

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var shortCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

const (
	minShortCodeLength = 3
	maxShortCodeLength = 64
)

func validateLinkInput(shortCode, originalURL string) (string, string, error) {
	shortCode = strings.TrimSpace(shortCode)
	originalURL = strings.TrimSpace(originalURL)

	if len(shortCode) < minShortCodeLength || len(shortCode) > maxShortCodeLength || !shortCodePattern.MatchString(shortCode) {
		return "", "", errors.New("short code must be 3-64 characters: letters, numbers, hyphens, or underscores")
	}

	parsedURL, err := url.Parse(originalURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return "", "", errors.New("URL must be a valid http(s) URL")
	}

	return shortCode, originalURL, nil
}
