package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromPasswdFile(t *testing.T) {
	// Create a temporary passwd file
	tmpDir := t.TempDir()
	passwdFile := filepath.Join(tmpDir, ".passwd-s3fs")
	
	// Write test credentials
	err := os.WriteFile(passwdFile, []byte("TEST_ACCESS_KEY:TEST_SECRET_KEY"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test passwd file: %v", err)
	}

	cred := NewCredentials()
	err = cred.LoadFromPasswdFile(passwdFile)
	if err != nil {
		t.Fatalf("Failed to load credentials: %v", err)
	}

	if cred.AccessKeyID != "TEST_ACCESS_KEY" {
		t.Errorf("Expected AccessKeyID 'TEST_ACCESS_KEY', got '%s'", cred.AccessKeyID)
	}

	if cred.SecretAccessKey != "TEST_SECRET_KEY" {
		t.Errorf("Expected SecretAccessKey 'TEST_SECRET_KEY', got '%s'", cred.SecretAccessKey)
	}
}

func TestLoadFromPasswdFileInvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	passwdFile := filepath.Join(tmpDir, ".passwd-s3fs")
	
	// Write invalid format (no colon)
	err := os.WriteFile(passwdFile, []byte("INVALID_FORMAT"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test passwd file: %v", err)
	}

	cred := NewCredentials()
	err = cred.LoadFromPasswdFile(passwdFile)
	if err == nil {
		t.Error("Expected error for invalid format, got nil")
	}
}

func TestLoadFromPasswdFileNotFound(t *testing.T) {
	cred := NewCredentials()
	err := cred.LoadFromPasswdFile("/nonexistent/file")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	os.Setenv("AWS_ACCESS_KEY_ID", "ENV_ACCESS_KEY")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "ENV_SECRET_KEY")
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	}()

	cred := NewCredentials()
	err := cred.LoadFromEnvironment()
	if err != nil {
		t.Fatalf("Failed to load credentials from environment: %v", err)
	}

	if cred.AccessKeyID != "ENV_ACCESS_KEY" {
		t.Errorf("Expected AccessKeyID 'ENV_ACCESS_KEY', got '%s'", cred.AccessKeyID)
	}

	if cred.SecretAccessKey != "ENV_SECRET_KEY" {
		t.Errorf("Expected SecretAccessKey 'ENV_SECRET_KEY', got '%s'", cred.SecretAccessKey)
	}
}

func TestIsValid(t *testing.T) {
	cred := NewCredentials()
	if cred.IsValid() {
		t.Error("Expected invalid credentials for empty cred, got valid")
	}

	cred.AccessKeyID = "TEST_KEY"
	cred.SecretAccessKey = "TEST_SECRET"
	if !cred.IsValid() {
		t.Error("Expected valid credentials, got invalid")
	}
}
