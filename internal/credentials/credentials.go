package credentials

import (
	"fmt"
	"os"
	"strings"
)

// Credentials holds AWS credentials
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Region          string
}

// NewCredentials creates a new credentials instance
func NewCredentials() *Credentials {
	return &Credentials{}
}

// LoadFromPasswdFile loads credentials from a passwd file in format ACCESS_KEY:SECRET_KEY
func (c *Credentials) LoadFromPasswdFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read passwd file: %w", err)
	}

	content := strings.TrimSpace(string(data))
	parts := strings.Split(content, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid passwd file format, expected ACCESS_KEY:SECRET_KEY")
	}

	c.AccessKeyID = strings.TrimSpace(parts[0])
	c.SecretAccessKey = strings.TrimSpace(parts[1])

	return nil
}

// LoadFromEnvironment loads credentials from environment variables
func (c *Credentials) LoadFromEnvironment() error {
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	sessionToken := os.Getenv("AWS_SESSION_TOKEN")

	if accessKey == "" || secretKey == "" {
		return fmt.Errorf("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set")
	}

	c.AccessKeyID = accessKey
	c.SecretAccessKey = secretKey
	c.SessionToken = sessionToken

	return nil
}

// IsValid checks if credentials are valid (both access key and secret are set)
func (c *Credentials) IsValid() bool {
	return c.AccessKeyID != "" && c.SecretAccessKey != ""
}
