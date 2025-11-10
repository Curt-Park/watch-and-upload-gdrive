package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestValidateDirectory(t *testing.T) {
	tests := []struct {
		name    string
		dirPath string
		wantErr bool
		setup   func() (string, func()) // returns temp dir and cleanup function
	}{
		{
			name:    "valid directory",
			dirPath: "",
			wantErr: false,
			setup: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "wug-test-*")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
		},
		{
			name:    "empty path",
			dirPath: "",
			wantErr: true,
			setup: func() (string, func()) {
				return "", func() {}
			},
		},
		{
			name:    "non-existent directory",
			dirPath: "/nonexistent/directory/path",
			wantErr: true,
			setup: func() (string, func()) {
				return "/nonexistent/directory/path", func() {}
			},
		},
		{
			name:    "file instead of directory",
			dirPath: "",
			wantErr: true,
			setup: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "wug-test-file-*")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				tmpFile.Close()
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirPath, cleanup := tt.setup()
			defer cleanup()

			if tt.dirPath == "" && dirPath != "" {
				dirPath = dirPath // Use the temp dir from setup
			} else if tt.dirPath != "" {
				dirPath = tt.dirPath // Use the specified path
			}

			err := validateDirectory(dirPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDirectory() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		wantErr bool
	}{
		{
			name:    "empty filter",
			filter:  "",
			wantErr: false,
		},
		{
			name:    "valid filter with extension",
			filter:  "*.txt",
			wantErr: false,
		},
		{
			name:    "valid filter with safetensors",
			filter:  "*.safetensors",
			wantErr: false,
		},
		{
			name:    "invalid filter - empty extension",
			filter:  "*.",
			wantErr: true,
		},
		{
			name:    "invalid filter - contains slash",
			filter:  "*.txt/path",
			wantErr: true,
		},
		{
			name:    "invalid filter - contains backslash",
			filter:  "*.txt\\path",
			wantErr: true,
		},
		{
			name:    "filter without asterisk",
			filter:  ".txt",
			wantErr: false, // Currently doesn't validate this case
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilter(tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFilter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		filter   string
		want     bool
	}{
		{
			name:     "match txt extension",
			filename: "test.txt",
			filter:   "*.txt",
			want:     true,
		},
		{
			name:     "match safetensors extension",
			filename: "model.safetensors",
			filter:   "*.safetensors",
			want:     true,
		},
		{
			name:     "no match - different extension",
			filename: "test.txt",
			filter:   "*.pdf",
			want:     false,
		},
		{
			name:     "case insensitive match",
			filename: "test.TXT",
			filter:   "*.txt",
			want:     true,
		},
		{
			name:     "case insensitive filter",
			filename: "test.txt",
			filter:   "*.TXT",
			want:     true,
		},
		{
			name:     "no match - no extension",
			filename: "test",
			filter:   "*.txt",
			want:     false,
		},
		{
			name:     "filter without asterisk - contains check",
			filename: "test.txt",
			filter:   ".txt",
			want:     true,
		},
		{
			name:     "filter without asterisk - no match",
			filename: "test.pdf",
			filter:   ".txt",
			want:     false,
		},
		{
			name:     "match with path",
			filename: "/path/to/file.txt",
			filter:   "*.txt",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesFilter(tt.filename, tt.filter)
			if got != tt.want {
				t.Errorf("matchesFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenFromFile(t *testing.T) {
	// Create a temporary token file
	tmpDir, err := os.MkdirTemp("", "wug-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tokenFile := filepath.Join(tmpDir, "token.json")

	// Test case 1: Valid token file
	t.Run("valid token file", func(t *testing.T) {
		token := &oauth2.Token{
			AccessToken:  "test-access-token",
			TokenType:    "Bearer",
			RefreshToken: "test-refresh-token",
			Expiry:       time.Now().Add(1 * time.Hour),
		}

		// Write token to file
		f, err := os.Create(tokenFile)
		if err != nil {
			t.Fatalf("Failed to create token file: %v", err)
		}
		if err := json.NewEncoder(f).Encode(token); err != nil {
			f.Close()
			t.Fatalf("Failed to encode token: %v", err)
		}
		f.Close()

		// Read token from file
		got, err := tokenFromFile(tokenFile)
		if err != nil {
			t.Errorf("tokenFromFile() error = %v", err)
			return
		}

		if got.AccessToken != token.AccessToken {
			t.Errorf("tokenFromFile() AccessToken = %v, want %v", got.AccessToken, token.AccessToken)
		}
		if got.RefreshToken != token.RefreshToken {
			t.Errorf("tokenFromFile() RefreshToken = %v, want %v", got.RefreshToken, token.RefreshToken)
		}
	})

	// Test case 2: Non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		_, err := tokenFromFile(filepath.Join(tmpDir, "nonexistent.json"))
		if err == nil {
			t.Error("tokenFromFile() expected error for non-existent file")
		}
	})

	// Test case 3: Invalid JSON
	t.Run("invalid JSON", func(t *testing.T) {
		invalidFile := filepath.Join(tmpDir, "invalid.json")
		if err := os.WriteFile(invalidFile, []byte("invalid json"), 0644); err != nil {
			t.Fatalf("Failed to write invalid JSON: %v", err)
		}

		_, err := tokenFromFile(invalidFile)
		if err == nil {
			t.Error("tokenFromFile() expected error for invalid JSON")
		}
	})
}

func TestSaveToken(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wug-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tokenFile := filepath.Join(tmpDir, "token.json")

	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		TokenType:    "Bearer",
		RefreshToken: "test-refresh-token",
		Expiry:       time.Now().Add(1 * time.Hour),
	}

	// Save token
	if err := saveToken(tokenFile, token); err != nil {
		t.Fatalf("saveToken() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tokenFile); os.IsNotExist(err) {
		t.Error("saveToken() file was not created")
	}

	// Verify token can be read back
	readToken, err := tokenFromFile(tokenFile)
	if err != nil {
		t.Fatalf("Failed to read saved token: %v", err)
	}

	if readToken.AccessToken != token.AccessToken {
		t.Errorf("saveToken() AccessToken = %v, want %v", readToken.AccessToken, token.AccessToken)
	}
	if readToken.RefreshToken != token.RefreshToken {
		t.Errorf("saveToken() RefreshToken = %v, want %v", readToken.RefreshToken, token.RefreshToken)
	}
}

func TestWaitForFileReady(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wug-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")

	t.Run("file ready immediately", func(t *testing.T) {
		// Create a file with some content
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		ctx := context.Background()
		err := waitForFileReady(ctx, testFile)
		if err != nil {
			t.Errorf("waitForFileReady() error = %v", err)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		ctx := context.Background()
		err := waitForFileReady(ctx, filepath.Join(tmpDir, "nonexistent.txt"))
		if err == nil {
			t.Error("waitForFileReady() expected error for non-existent file")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		err := waitForFileReady(ctx, testFile)
		if err == nil {
			t.Error("waitForFileReady() expected error for cancelled context")
		}
	})
}

func TestWaitForFileReady_FileSizeStabilization(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wug-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "growing.txt")

	// Create a file that grows over time
	go func() {
		f, err := os.Create(testFile)
		if err != nil {
			return
		}
		defer f.Close()

		for i := 0; i < 5; i++ {
			f.WriteString("test data\n")
			f.Sync()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Wait a bit for file to start growing
	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	err = waitForFileReady(ctx, testFile)
	if err != nil {
		t.Errorf("waitForFileReady() error = %v", err)
	}
}

