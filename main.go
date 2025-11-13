package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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

// initLogger initializes the logger based on verbose flag
func initLogger() {
	var level slog.Level
	if verbose {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	logger = slog.New(slog.NewTextHandler(os.Stdout, opts))
}

var (
	directoryPath string
	filterPattern string
	folderID      string
	verbose       bool
	logger        *slog.Logger
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
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose (debug) logging")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(exitRuntimeError)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Initialize logger based on verbose flag
	initLogger()

	directoryPath = args[0]

	if err := validateDirectory(directoryPath); err != nil {
		logger.Error("Invalid directory", "path", directoryPath, "error", err)
		return fmt.Errorf("%w: %v", ErrInvalidDirectory, err)
	}

	if filterPattern != "" {
		if err := validateFilter(filterPattern); err != nil {
			logger.Error("Invalid filter pattern", "pattern", filterPattern, "error", err)
			return fmt.Errorf("%w: %v", ErrInvalidFilter, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Shutting down gracefully...")
		cancel()
	}()

	client, err := getDriveClient(ctx)
	if err != nil {
		logger.Error("Authentication failed", "error", err)
		return fmt.Errorf("%w: %v", ErrAuthFailed, err)
	}

	logger.Info("Watching directory", "path", directoryPath)
	if filterPattern != "" {
		logger.Info("Filter pattern", "pattern", filterPattern)
	}

	var resolvedFolderID string
	if folderID != "" {
		// Remove surrounding quotes if present (e.g., "test" -> test, 'test' -> test)
		cleanedFolderID := trimQuotes(folderID)
		logger.Debug("Processing folderID", "original", folderID, "cleaned", cleanedFolderID)
		var err error
		resolvedFolderID, err = findOrCreateFolder(ctx, client, cleanedFolderID)
		if err != nil {
			logger.Error("Failed to find or create folder", "name", cleanedFolderID, "error", err)
			return fmt.Errorf("failed to find or create folder: %v", err)
		}
		logger.Info("Using folder", "name", cleanedFolderID, "id", resolvedFolderID)
		logger.Debug("Resolved folderID", "id", resolvedFolderID)
	} else {
		logger.Info("Uploading to Google Drive root directory")
	}

	if err := watchDirectory(ctx, client, directoryPath, filterPattern, resolvedFolderID); err != nil {
		logger.Error("Runtime error in watchDirectory", "error", err)
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
			logger.Error("Directory does not exist", "path", dirPath)
			return fmt.Errorf("directory does not exist: %s", dirPath)
		}
		logger.Error("Cannot access directory", "path", dirPath, "error", err)
		return fmt.Errorf("cannot access directory: %v", err)
	}

	// Check if it's actually a directory
	if !info.IsDir() {
		logger.Error("Path is not a directory", "path", dirPath)
		return fmt.Errorf("path is not a directory: %s", dirPath)
	}

	// Check if directory is readable
	if info.Mode().Perm()&0444 == 0 {
		logger.Error("Directory is not readable", "path", dirPath)
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

// trimQuotes removes surrounding quotes (single or double) from a string if present.
// It only removes quotes if both the first and last characters are the same quote type.
// This handles cases where users input quoted strings like "test" or 'test'.
func trimQuotes(s string) string {
	if len(s) < 2 {
		return s
	}

	first := s[0]
	last := s[len(s)-1]

	// Check if string is surrounded by matching quotes
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		return s[1 : len(s)-1]
	}

	return s
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
			logger.Error("Specified ID is not a folder", "id", folderNameOrID)
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
		logger.Error("Failed to search for folder", "name", folderNameOrID, "error", err)
		return "", fmt.Errorf("failed to search for folder: %v", err)
	}

	// If folder exists, return its ID
	if len(list.Files) > 0 {
		return list.Files[0].Id, nil
	}

	// Folder doesn't exist, create it
	logger.Info("Folder not found, creating it", "name", folderNameOrID)
	folder := &drive.File{
		Name:     folderNameOrID,
		MimeType: "application/vnd.google-apps.folder",
	}

	createdFolder, err := driveService.Files.Create(folder).Fields("id", "name").Context(ctx).Do()
	if err != nil {
		logger.Error("Failed to create folder", "name", folderNameOrID, "error", err)
		return "", fmt.Errorf("failed to create folder: %v", err)
	}

	logger.Info("Created folder", "name", createdFolder.Name, "id", createdFolder.Id)
	return createdFolder.Id, nil
}

func getDriveClient(ctx context.Context) (*drive.Service, error) {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("Failed to get home directory", "error", err)
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	// Credentials file path in home directory
	credentialsFile := filepath.Join(homeDir, "credentials.json")

	// Try to read credentials.json file
	credentialsJSON, err := os.ReadFile(credentialsFile)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Error("Credentials file not found", "path", credentialsFile, "homeDir", homeDir)
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
		logger.Error("Failed to read credentials file", "path", credentialsFile, "error", err)
		return nil, fmt.Errorf("failed to read credentials file: %v", err)
	}

	config, err := google.ConfigFromJSON(credentialsJSON, drive.DriveFileScope)
	if err != nil {
		logger.Error("Failed to configure OAuth2", "error", err)
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
			logger.Error("Failed to obtain token", "error", err)
			return nil, fmt.Errorf("failed to obtain token: %v", err)
		}
		if err := saveToken(tokenFile, tok); err != nil {
			logger.Error("Failed to save token", "path", tokenFile, "error", err)
			return nil, fmt.Errorf("failed to save token: %v", err)
		}
	}

	// Create Drive service
	client := config.Client(ctx, tok)
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		logger.Error("Drive service initialization failed", "error", err)
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
		logger.Error("Failed to decode token", "file", file, "error", err)
		return nil, fmt.Errorf("failed to decode token: %v", err)
	}
	return tok, nil
}

func saveToken(path string, token *oauth2.Token) error {
	logger.Info("Saving token", "path", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		logger.Error("Unable to open token file", "path", path, "error", err)
		return fmt.Errorf("unable to open token file: %v", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(token); err != nil {
		logger.Error("Failed to encode token", "path", path, "error", err)
		return fmt.Errorf("failed to encode token: %v", err)
	}
	return nil
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	logger.Info("Open the following URL in your browser and enter the authorization code", "url", authURL)
	fmt.Print("Enter authorization code: ")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		logger.Error("Failed to read authorization code", "error", err)
		return nil, fmt.Errorf("failed to read authorization code: %v", err)
	}

	tok, err := config.Exchange(ctx, authCode)
	if err != nil {
		logger.Error("Failed to exchange token", "error", err)
		return nil, fmt.Errorf("failed to exchange token: %v", err)
	}
	return tok, nil
}

// normalizePath converts a file path to an absolute, cleaned path for consistent comparison.
// On Linux/Ubuntu, it also resolves symbolic links to ensure consistent path comparison
// between filepath.Walk and fsnotify events.
func normalizePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		logger.Error("Failed to get absolute path", "path", path, "error", err)
		return "", fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Resolve symbolic links to ensure consistent path comparison
	// This is especially important on Linux/Ubuntu where symlinks are common
	// filepath.EvalSymlinks returns the path itself if it's not a symlink or doesn't exist
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If the path doesn't exist yet (e.g., during file creation), use the cleaned absolute path
		// This can happen when fsnotify reports a file creation event before the file is fully created
		return filepath.Clean(absPath), nil
	}

	return filepath.Clean(resolvedPath), nil
}

// scanExistingFiles scans the directory and returns a map of existing file paths
func scanExistingFiles(dirPath string) (map[string]bool, error) {
	existingFiles := make(map[string]bool)
	if err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			normalizedPath, err := normalizePath(path)
			if err != nil {
				return err
			}
			existingFiles[normalizedPath] = true
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return existingFiles, nil
}

// processFileEvent checks if a file should be processed for upload
func processFileEvent(normalizedPath string, filter string, existingFiles map[string]bool) bool {
	info, err := os.Stat(normalizedPath)
	if err != nil || info.IsDir() {
		return false
	}

	if filter != "" && !matchesFilter(normalizedPath, filter) {
		return false
	}

	if existingFiles[normalizedPath] {
		return false
	}

	existingFiles[normalizedPath] = true
	return true
}

// startUploadWorkers starts worker goroutines to process file uploads
func startUploadWorkers(ctx context.Context, wg *sync.WaitGroup, uploadQueue <-chan string, driveService *drive.Service, folderID string, numWorkers int) {
	logger.Debug("Starting upload workers", "folderID", folderID, "numWorkers", numWorkers)
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			logger.Debug("Worker started", "workerID", workerID, "folderID", folderID)
			for {
				select {
				case filePath := <-uploadQueue:
					if filePath == "" {
						return
					}
					logger.Info("New file detected", "path", filePath)
					logger.Debug("Worker uploading file", "workerID", workerID, "filePath", filePath, "folderID", folderID)
					if err := uploadFile(ctx, driveService, filePath, folderID); err != nil {
						logger.Error("Upload failed", "path", filePath, "error", err)
					} else {
						logger.Info("Upload completed", "path", filePath)
					}
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}
}

func watchDirectory(ctx context.Context, driveService *drive.Service, dirPath string, filter string, folderID string) error {
	logger.Debug("watchDirectory called", "dirPath", dirPath, "filter", filter, "folderID", folderID)

	normalizedDirPath, err := normalizePath(dirPath)
	if err != nil {
		logger.Error("Failed to normalize directory path", "path", dirPath, "error", err)
		return fmt.Errorf("failed to normalize directory path: %v", err)
	}
	logger.Debug("Normalized directory path", "path", normalizedDirPath)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("Failed to create file watcher", "error", err)
		return fmt.Errorf("failed to create file watcher: %v", err)
	}
	defer watcher.Close()

	if err := watcher.Add(normalizedDirPath); err != nil {
		logger.Error("Failed to add directory to watcher", "path", normalizedDirPath, "error", err)
		return fmt.Errorf("failed to add directory to watcher: %v", err)
	}

	existingFiles, err := scanExistingFiles(normalizedDirPath)
	if err != nil {
		logger.Error("Failed to scan directory", "path", normalizedDirPath, "error", err)
		return fmt.Errorf("failed to scan directory: %v", err)
	}

	logger.Info("Watching for files... (Press Ctrl+C to exit)")

	uploadQueue := make(chan string, 10)
	var wg sync.WaitGroup
	const numWorkers = 3

	logger.Debug("Starting upload workers", "folderID", folderID)
	startUploadWorkers(ctx, &wg, uploadQueue, driveService, folderID, numWorkers)

	for {
		select {
		case <-ctx.Done():
			close(uploadQueue)
			wg.Wait()
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if event.Op&fsnotify.Create == fsnotify.Create {
				logger.Debug("File creation event detected", "path", event.Name)
				normalizedPath, err := normalizePath(event.Name)
				if err != nil {
					logger.Error("Failed to normalize event path", "path", event.Name, "error", err)
					continue
				}
				logger.Debug("Normalized event path", "path", normalizedPath)

				if processFileEvent(normalizedPath, filter, existingFiles) {
					logger.Debug("File event processed, waiting for file ready", "path", normalizedPath)
					if err := waitForFileReady(ctx, normalizedPath); err != nil {
						logger.Error("Failed to wait for file ready", "path", normalizedPath, "error", err)
						continue
					}

					select {
					case uploadQueue <- normalizedPath:
						logger.Debug("File queued for upload", "path", normalizedPath)
					case <-ctx.Done():
						return nil
					}
				} else {
					logger.Debug("File event skipped (filtered or already exists)", "path", normalizedPath)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			logger.Error("Watch error", "error", err)
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
			logger.Error("File ready wait timeout", "path", filePath, "maxWaitTime", maxWaitTime)
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
	logger.Debug("uploadFile called", "filePath", filePath, "folderID", folderID)

	file, err := os.Open(filePath)
	if err != nil {
		logger.Error("Failed to open file", "path", filePath, "error", err)
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Extract filename
	fileName := filepath.Base(filePath)
	logger.Debug("Extracted filename", "filename", fileName)

	// Upload to Google Drive
	f := &drive.File{
		Name: fileName,
	}

	// Set parent folder if specified
	if folderID != "" {
		f.Parents = []string{folderID}
		logger.Debug("Setting parent folder ID", "folderID", folderID)
	} else {
		logger.Debug("No folderID specified, uploading to root directory")
	}

	// Upload file with context support
	_, err = driveService.Files.Create(f).Media(file).Context(ctx).Do()
	if err != nil {
		if folderID != "" {
			logger.Error("Failed to upload to Google Drive folder", "filePath", filePath, "folderID", folderID, "error", err)
			return fmt.Errorf("failed to upload to Google Drive folder (ID: %s): %v", folderID, err)
		}
		logger.Error("Failed to upload to Google Drive", "filePath", filePath, "error", err)
		return fmt.Errorf("failed to upload to Google Drive: %v", err)
	}

	logger.Debug("File uploaded successfully", "filename", fileName)
	return nil
}
