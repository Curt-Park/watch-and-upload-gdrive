package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	// OAuth2 endpoints
	authURI              = "https://accounts.google.com/o/oauth2/auth"
	tokenURI             = "https://oauth2.googleapis.com/token"
	authProviderCertURL  = "https://www.googleapis.com/oauth2/v1/certs"
	defaultTokenFileName = "token.json"
	maxWaitTime          = 30 * time.Second
	checkInterval        = 500 * time.Millisecond
)

// Exit codes
const (
	exitSuccess = iota
	exitConfigError
	exitAuthError
	exitRuntimeError
)

// Custom error types
var (
	ErrConfigNotFound     = errors.New("configuration not found")
	ErrInvalidDirectory   = errors.New("invalid directory")
	ErrInvalidFilter      = errors.New("invalid filter pattern")
	ErrAuthFailed         = errors.New("authentication failed")
	ErrDriveServiceFailed = errors.New("drive service initialization failed")
	ErrRuntimeError       = errors.New("runtime error")
)

var (
	directoryPath string
	filterPattern string
	folderID      string
)

var rootCmd = &cobra.Command{
	Use:   "wug [directory]",
	Short: "Watch and Upload to Google Drive",
	Long:  "A tool that watches a directory and uploads new files to Google Drive",
	Args:  cobra.ExactArgs(1),
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringVarP(&filterPattern, "filter", "f", "", "File extension filter for uploads (e.g., *.txt)")
	rootCmd.Flags().StringVarP(&folderID, "path", "p", "", "Google Drive folder name or ID to upload files to (will be created if it doesn't exist)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(exitRuntimeError)
	}
}

func run(cmd *cobra.Command, args []string) error {
	directoryPath = args[0]

	// Validate directory path
	if err := validateDirectory(directoryPath); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidDirectory, err)
	}

	// Validate filter pattern if provided
	if filterPattern != "" {
		if err := validateFilter(filterPattern); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidFilter, err)
		}
	}

	// Note: folderID validation is now done when finding/creating the folder

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down gracefully...")
		cancel()
	}()

	// Initialize Google Drive client
	client, err := getDriveClient(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAuthFailed, err)
	}

	fmt.Printf("Watching directory: %s\n", directoryPath)
	if filterPattern != "" {
		fmt.Printf("Filter pattern: %s\n", filterPattern)
	}

	// Resolve folder ID if folder name is provided
	var resolvedFolderID string
	if folderID != "" {
		var err error
		resolvedFolderID, err = findOrCreateFolder(ctx, client, folderID)
		if err != nil {
			return fmt.Errorf("failed to find or create folder: %v", err)
		}
		fmt.Printf("Using folder: %s (ID: %s)\n", folderID, resolvedFolderID)
	}

	// Start file watching
	if err := watchDirectory(ctx, client, directoryPath, filterPattern, resolvedFolderID); err != nil {
		return fmt.Errorf("%w: %v", ErrRuntimeError, err)
	}

	return nil
}

func validateDirectory(dirPath string) error {
	// Check if path is empty
	if dirPath == "" {
		return errors.New("directory path cannot be empty")
	}

	// Check if directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", dirPath)
		}
		return fmt.Errorf("cannot access directory: %v", err)
	}

	// Check if it's actually a directory
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", dirPath)
	}

	// Check if directory is readable
	if info.Mode().Perm()&0444 == 0 {
		return fmt.Errorf("directory is not readable: %s", dirPath)
	}

	return nil
}

func validateFilter(filter string) error {
	if filter == "" {
		return nil
	}

	// Basic validation: should start with *.
	if strings.HasPrefix(filter, "*.") {
		ext := strings.TrimPrefix(filter, "*.")
		if ext == "" {
			return errors.New("filter extension cannot be empty")
		}
		// Check for invalid characters
		if strings.Contains(ext, "/") || strings.Contains(ext, "\\") {
			return errors.New("filter extension cannot contain path separators")
		}
	}

	return nil
}

// isLikelyFolderID checks if a string looks like a Google Drive folder ID
func isLikelyFolderID(folderNameOrID string) bool {
	return len(folderNameOrID) >= 20 && !strings.Contains(folderNameOrID, "/")
}

// buildFolderSearchQuery builds a Google Drive API query to search for a folder by name
func buildFolderSearchQuery(folderName string) string {
	escapedName := strings.ReplaceAll(folderName, "'", "\\'")
	return fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder' and trashed=false", escapedName)
}

// findOrCreateFolder finds a folder by name or ID, or creates it if it doesn't exist
func findOrCreateFolder(ctx context.Context, driveService *drive.Service, folderNameOrID string) (string, error) {
	// First, check if it's a folder ID (typically longer than 20 characters)
	// If it looks like an ID, try to use it directly
	if isLikelyFolderID(folderNameOrID) {
		// Try to get the folder by ID
		folder, err := driveService.Files.Get(folderNameOrID).Fields("id", "name", "mimeType").Context(ctx).Do()
		if err == nil {
			// Check if it's actually a folder
			if folder.MimeType == "application/vnd.google-apps.folder" {
				return folder.Id, nil
			}
			return "", fmt.Errorf("the specified ID is not a folder")
		}
		// If 404, it's not a valid ID, so treat it as a folder name
	}

	// Search for folder by name in root directory
	query := buildFolderSearchQuery(folderNameOrID)
	list, err := driveService.Files.List().
		Q(query).
		Fields("files(id, name)").
		PageSize(10).
		Context(ctx).
		Do()
	if err != nil {
		return "", fmt.Errorf("failed to search for folder: %v", err)
	}

	// If folder exists, return its ID
	if len(list.Files) > 0 {
		return list.Files[0].Id, nil
	}

	// Folder doesn't exist, create it
	fmt.Printf("Folder '%s' not found, creating it...\n", folderNameOrID)
	folder := &drive.File{
		Name:     folderNameOrID,
		MimeType: "application/vnd.google-apps.folder",
	}

	createdFolder, err := driveService.Files.Create(folder).Fields("id", "name").Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create folder: %v", err)
	}

	fmt.Printf("Created folder '%s' (ID: %s)\n", createdFolder.Name, createdFolder.Id)
	return createdFolder.Id, nil
}

func getDriveClient(ctx context.Context) (*drive.Service, error) {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	// Credentials file path in home directory
	credentialsFile := filepath.Join(homeDir, "credentials.json")

	// Try to read credentials.json file
	credentialsJSON, err := os.ReadFile(credentialsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(`%w: credentials.json file not found

Please follow these steps to set up Google Drive credentials:

1. Go to Google Cloud Console: https://console.cloud.google.com/
2. Create a new project or select an existing one
3. Enable Google Drive API:
   - Navigate to "APIs & Services" > "Library"
   - Search for "Google Drive API" and enable it
4. Create OAuth 2.0 Credentials:
   - Navigate to "APIs & Services" > "Credentials"
   - Click "+ CREATE CREDENTIALS" > "OAuth client ID"
   - If prompted, configure the OAuth consent screen first
   - Application type: Select "Desktop app"
   - Click "Create"
5. Download the credentials:
   - Click the download icon (⬇️) next to your OAuth 2.0 Client ID
   - Save the JSON file as "credentials.json" in your home directory (%s)

For detailed instructions, see the README.md file.`, ErrConfigNotFound, homeDir)
		}
		return nil, fmt.Errorf("failed to read credentials file: %v", err)
	}

	config, err := google.ConfigFromJSON(credentialsJSON, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("failed to configure OAuth2: %v", err)
	}

	// Token file path in home directory
	tokenFile := filepath.Join(homeDir, defaultTokenFileName)

	// Load existing token
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		// Generate new token if not exists
		tok, err = getTokenFromWeb(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to obtain token: %v", err)
		}
		if err := saveToken(tokenFile, tok); err != nil {
			return nil, fmt.Errorf("failed to save token: %v", err)
		}
	}

	// Create Drive service
	client := config.Client(ctx, tok)
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDriveServiceFailed, err)
	}

	return srv, nil
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, fmt.Errorf("failed to decode token: %v", err)
	}
	return tok, nil
}

func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving token to %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to open token file: %v", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("failed to encode token: %v", err)
	}
	return nil
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Open the following URL in your browser and enter the authorization code:\n%v\n", authURL)
	fmt.Print("Enter authorization code: ")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("failed to read authorization code: %v", err)
	}

	tok, err := config.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %v", err)
	}
	return tok, nil
}

func watchDirectory(ctx context.Context, driveService *drive.Service, dirPath string, filter string, folderID string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %v", err)
	}
	defer watcher.Close()

	// Start watching directory
	if err := watcher.Add(dirPath); err != nil {
		return fmt.Errorf("failed to add directory to watcher: %v", err)
	}

	// Track existing files to prevent duplicate uploads
	existingFiles := make(map[string]bool)
	if err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			existingFiles[path] = true
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to scan directory: %v", err)
	}

	fmt.Println("Watching for files... (Press Ctrl+C to exit)")

	// Upload queue and worker pool
	uploadQueue := make(chan string, 10)
	var wg sync.WaitGroup
	const numWorkers = 3

	// Start upload workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case filePath := <-uploadQueue:
					if filePath == "" {
						return
					}
					fmt.Printf("New file detected: %s\n", filePath)
					if err := uploadFile(ctx, driveService, filePath, folderID); err != nil {
						log.Printf("Upload failed for %s: %v", filePath, err)
					} else {
						fmt.Printf("Upload completed: %s\n", filePath)
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Main watch loop
	for {
		select {
		case <-ctx.Done():
			// Graceful shutdown: close upload queue and wait for workers
			close(uploadQueue)
			wg.Wait()
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Process only file creation events
			if event.Op&fsnotify.Create == fsnotify.Create {
				// Check if it's a file (exclude directories)
				info, err := os.Stat(event.Name)
				if err != nil {
					continue
				}

				if !info.IsDir() {
					// Check filter
					if filter != "" && !matchesFilter(event.Name, filter) {
						continue
					}

					// Check if file already exists (exclude files found in initial scan)
					if existingFiles[event.Name] {
						continue
					}

					// Mark as existing to prevent duplicate processing
					existingFiles[event.Name] = true

					// Wait until file is completely written (file size stabilizes)
					if err := waitForFileReady(ctx, event.Name); err != nil {
						log.Printf("Failed to wait for file ready: %v", err)
						continue
					}

					// Queue file for upload
					select {
					case uploadQueue <- event.Name:
					case <-ctx.Done():
						return nil
					}
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("Watch error: %v", err)
		}
	}
}

func waitForFileReady(ctx context.Context, filePath string) error {
	startTime := time.Now()
	var lastSize int64 = -1

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		info, err := os.Stat(filePath)
		if err != nil {
			return err
		}

		currentSize := info.Size()
		if currentSize == lastSize && currentSize > 0 {
			// File size has stabilized
			return nil
		}

		if time.Since(startTime) > maxWaitTime {
			// Maximum wait time exceeded
			return fmt.Errorf("file ready wait timeout")
		}

		lastSize = currentSize
		time.Sleep(checkInterval)
	}
}

func matchesFilter(filename string, filter string) bool {
	// Handle *.extension format filter
	if strings.HasPrefix(filter, "*.") {
		ext := strings.TrimPrefix(filter, "*.")
		return strings.HasSuffix(strings.ToLower(filename), "."+strings.ToLower(ext))
	}
	// Simple glob pattern matching
	return strings.Contains(filename, filter)
}

func uploadFile(ctx context.Context, driveService *drive.Service, filePath string, folderID string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Extract filename
	fileName := filepath.Base(filePath)

	// Upload to Google Drive
	f := &drive.File{
		Name: fileName,
	}

	// Set parent folder if specified
	if folderID != "" {
		f.Parents = []string{folderID}
	}

	// Upload file with context support
	_, err = driveService.Files.Create(f).Media(file).Context(ctx).Do()
	if err != nil {
		if folderID != "" {
			return fmt.Errorf("failed to upload to Google Drive folder (ID: %s): %v", folderID, err)
		}
		return fmt.Errorf("failed to upload to Google Drive: %v", err)
	}

	return nil
}
