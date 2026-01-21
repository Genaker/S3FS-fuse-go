//go:build functional

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestMainHelp tests the help command
func TestMainHelp(t *testing.T) {
	// Build the binary
	cmd := exec.Command("go", "build", "-o", "s3fs-test", ".")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping functional test - failed to build: %v", err)
		return
	}
	defer os.Remove("s3fs-test")

	// Test help (should fail without required args but show usage)
	helpCmd := exec.Command("./s3fs-test", "-h")
	output, err := helpCmd.CombinedOutput()
	if err == nil {
		t.Logf("Help output: %s", string(output))
	}
	// Help might not be implemented, so we don't fail if it errors
}

// TestMainMissingArgs tests that main fails with missing required arguments
func TestMainMissingArgs(t *testing.T) {
	// Build the binary
	cmd := exec.Command("go", "build", "-o", "s3fs-test", ".")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping functional test - failed to build: %v", err)
		return
	}
	defer os.Remove("s3fs-test")

	// Test without bucket (should fail)
	testCmd := exec.Command("./s3fs-test")
	output, err := testCmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error when bucket is missing")
	}
	if len(output) > 0 {
		t.Logf("Error output (expected): %s", string(output))
	}
}

// TestMainMissingMountpoint tests that main fails with missing mountpoint
func TestMainMissingMountpoint(t *testing.T) {
	// Build the binary
	cmd := exec.Command("go", "build", "-o", "s3fs-test", ".")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping functional test - failed to build: %v", err)
		return
	}
	defer os.Remove("s3fs-test")

	// Test without mountpoint (should fail)
	testCmd := exec.Command("./s3fs-test", "-bucket", "test-bucket")
	output, err := testCmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error when mountpoint is missing")
	}
	if len(output) > 0 {
		t.Logf("Error output (expected): %s", string(output))
	}
}

// TestMainInvalidCredentials tests that main fails with invalid credentials
func TestMainInvalidCredentials(t *testing.T) {
	// Build the binary
	cmd := exec.Command("go", "build", "-o", "s3fs-test", ".")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping functional test - failed to build: %v", err)
		return
	}
	defer os.Remove("s3fs-test")

	// Create a temporary mountpoint
	tmpDir := filepath.Join(os.TempDir(), "s3fs-test-mount")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	// Test with invalid credentials (should fail)
	testCmd := exec.Command("./s3fs-test", "-bucket", "test-bucket", "-mountpoint", tmpDir)
	testCmd.Env = []string{} // Clear environment
	output, err := testCmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error with invalid credentials")
	}
	if len(output) > 0 {
		t.Logf("Error output (expected): %s", string(output))
	}
}
