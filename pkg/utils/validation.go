package utils

import (
	"errors"
	"net/url"
	"strings"
)

// ValidateURL validates that a string is a valid URL
func ValidateURL(urlStr string) error {
	if urlStr == "" {
		return errors.New("URL cannot be empty")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	if parsedURL.Scheme == "" {
		return errors.New("URL must have a scheme (http/https)")
	}

	if parsedURL.Host == "" {
		return errors.New("URL must have a host")
	}

	return nil
}

// ValidatePath validates that a string is a valid path
func ValidatePath(path string) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}

	if !strings.HasPrefix(path, "/") {
		return errors.New("path must start with /")
	}

	return nil
}

// ValidateServiceName validates that a service name is valid
func ValidateServiceName(name string) error {
	if name == "" {
		return errors.New("service name cannot be empty")
	}

	if strings.Contains(name, " ") {
		return errors.New("service name cannot contain spaces")
	}

	return nil
}

// ValidateAPIKey validates that an API key is not empty
func ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return errors.New("API key cannot be empty")
	}

	if len(apiKey) < 16 {
		return errors.New("API key must be at least 16 characters")
	}

	return nil
}
