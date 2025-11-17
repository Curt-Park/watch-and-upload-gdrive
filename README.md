# Watch and Upload to Google Drive (WUG)

A Go-based CLI tool that watches a directory and automatically uploads new files to Google Drive.

**한국어**: 한국어 문서는 [README.ko.md](README.ko.md)를 참조하세요.

## Features

- Real-time directory monitoring
- Automatic file upload
- File extension filtering support
- Google Drive OAuth2 authentication

## Installation

### Option 1: Download Pre-built Binary (Recommended)

Download the latest release for your platform using one of the following commands:

#### Linux
```bash
wget https://github.com/Curt-Park/watch-and-upload-gdrive/releases/latest/download/wug-linux-amd64 -O wug
chmod +x wug
```

#### macOS (Intel)
```bash
wget https://github.com/Curt-Park/watch-and-upload-gdrive/releases/latest/download/wug-darwin-amd64 -O wug
chmod +x wug
```

#### macOS (Apple Silicon)
```bash
wget https://github.com/Curt-Park/watch-and-upload-gdrive/releases/latest/download/wug-darwin-arm64 -O wug
chmod +x wug
```

#### Windows
```bash
# Using PowerShell
Invoke-WebRequest -Uri https://github.com/Curt-Park/watch-and-upload-gdrive/releases/latest/download/wug-windows-amd64.exe -OutFile wug.exe

# Or using curl (if available)
curl -L -o wug.exe https://github.com/Curt-Park/watch-and-upload-gdrive/releases/latest/download/wug-windows-amd64.exe
```

**Note**: If `wget` is not available, you can use `curl` instead:
```bash
# Linux/macOS alternative using curl
curl -L -o wug https://github.com/Curt-Park/watch-and-upload-gdrive/releases/latest/download/wug-linux-amd64
chmod +x wug
```

#### Install to PATH (optional)
```bash
# Linux/macOS
sudo mv wug /usr/local/bin/wug

# Or use it directly from current directory
./wug /path/to/directory
```

### Option 2: Build from Source

1. Clone the repository:
```bash
git clone https://github.com/Curt-Park/watch-and-upload-gdrive.git
cd watch-and-upload-gdrive
```

2. Install dependencies:
```bash
go mod download
```

3. Build:
```bash
go build -o wug
```

Or use Makefile:
```bash
make build
```

## Configuration

### Step 1: Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click on the project dropdown at the top of the page
3. Click **"New Project"**
4. Enter a project name (e.g., "wug-drive-uploader")
5. Click **"Create"**
6. Wait for the project to be created and select it

### Step 2: Enable Google Drive API

1. In the Google Cloud Console, navigate to **"APIs & Services"** > **"Library"**
2. Search for **"Google Drive API"**
3. Click on **"Google Drive API"** from the results
4. Click the **"Enable"** button
5. Wait for the API to be enabled (this may take a few moments)

### Step 3: Create OAuth 2.0 Credentials

1. Navigate to **"APIs & Services"** > **"Credentials"**
2. Click **"+ CREATE CREDENTIALS"** at the top of the page
3. Select **"OAuth client ID"** from the dropdown menu

   **Note**: If this is your first time creating credentials, you'll need to configure the OAuth consent screen first:
   - Click **"Configure Consent Screen"**
   - Select **"External"** (unless you have a Google Workspace account)
   - Click **"Create"**
   - Fill in the required information:
     - **App name**: Enter a name (e.g., "WUG Drive Uploader")
     - **User support email**: Select your email
     - **Developer contact information**: Enter your email
   - Click **"Save and Continue"**
   - On the **Scopes** page, click **"Save and Continue"** (no need to add scopes manually)
   - On the **Test users** page, click **"Save and Continue"** (you can add test users later if needed)
   - Review and click **"Back to Dashboard"**

4. Back in the **"Create OAuth client ID"** dialog:
   - **Application type**: Select **"Desktop app"**
   - **Name**: Enter a name (e.g., "WUG Desktop Client")
   - Click **"Create"**

5. The credentials will be created. You'll download them as a JSON file in the next step.

### Step 4: Download Credentials JSON File

1. In **"APIs & Services"** > **"Credentials"**, find your OAuth 2.0 Client ID
2. Click the **download icon** (⬇️) next to your client ID
3. A JSON file will be downloaded (e.g., `client_secret_123456789-abc.apps.googleusercontent.com.json`)
4. **Rename the file to `credentials.json`** and place it in your home directory (e.g., `mv ~/Downloads/client_secret_123456789-abc.apps.googleusercontent.com.json ~/credentials.json`)

**Important Notes:**
- The file must be named exactly `credentials.json` and placed in your home directory
- Keep your `credentials.json` file secure and never commit it to version control (already in `.gitignore`)

## Usage

### Basic Usage (Upload all new files)

```bash
# If you downloaded the binary and renamed it to 'wug'
./wug /path/to/directory

# Or if installed to PATH
wug /path/to/directory

# Windows
wug.exe C:\path\to\directory
```

This will upload files to the root of your Google Drive.

### Using Filter Option (Upload only specific file extensions)

```bash
./wug /path/to/directory --filter "*.safetensors"
```

Or using the short option:

```bash
./wug /path/to/directory -f "*.txt"
```

### Uploading to a Specific Google Drive Folder

To upload files to a specific folder in Google Drive, use the `--path` or `-p` option with the folder name or ID:

```bash
# Using folder name (will be created if it doesn't exist)
./wug /path/to/directory --path "MyUploads"
```

Or using the short option:

```bash
./wug /path/to/directory -p "MyUploads"
```

**Using folder name:**
- If the folder doesn't exist, it will be automatically created in your Google Drive root directory
- Example: `./wug /path/to/directory -p "Backups"` - creates or uses the "Backups" folder

**Using folder ID (advanced):**
- You can also use a folder ID directly if you know it
- Example: `./wug /path/to/directory -p "1a2b3c4d5e6f7g8h9i0j"`
- **How to find a Google Drive folder ID:**
  1. Open Google Drive in your web browser
  2. Navigate to the folder where you want to upload files
  3. Click on the folder to open it
  4. Look at the URL in your browser's address bar
  5. The folder ID is the long string of characters after `/folders/` in the URL
     - Example URL: `https://drive.google.com/drive/folders/1a2b3c4d5e6f7g8h9i0j`
     - Folder ID: `1a2b3c4d5e6f7g8h9i0j`

**Note**: Always quote the filter pattern (e.g., `"*.txt"`) to prevent shell glob expansion.

## First Run Authentication

When you run the program for the first time, an authentication URL will be displayed:

1. Open the displayed URL in your browser
2. Sign in with your Google account and grant permissions
3. If "Google hasn't verified this app" appears, click the **"Continue"** button
4. After granting permissions, you will be redirected to `http://localhost` - **This is normal behavior**
5. The authorization code will be in the browser address bar URL. The URL will look like this:
   ```
   http://localhost/?state=state-token&code=4/0A0d-example_code_AbCdEf123456_GhIjKl789012_MnOpQr345678&scope=https://www.googleapis.com/auth/drive.file
   ```
6. **Copy only the authorization code** - this is the part after `code=` and before the next `&` symbol.
   - In the example above, you would copy: `4/0A0d-example_code_AbCdEf123456_GhIjKl789012_MnOpQr345678`
   - **Important**: Copy only the code value, not the entire URL or the `code=` prefix
7. Paste the authorization code into the terminal when prompted
8. The authentication token will be saved to `token.json` in your home directory (will be used automatically in subsequent runs)

**Note**: If you're using a test OAuth app, you may need to add your Google account as a test user in the OAuth consent screen settings.

## How It Works

1. The program watches the specified directory for new files
2. When a new file is created, it's automatically detected
3. The program waits until the file is completely written using an intelligent detection mechanism
4. If a filter is specified, the file extension is checked
5. The file is uploaded to Google Drive

### File Write Completion Detection

The program uses a sophisticated **debounce + rename pattern** to accurately detect when file writing is complete, eliminating the need for fixed timeouts:

#### 1. **Write Event Debouncing**
- Monitors `Write` events from the file system watcher
- Tracks the last write event timestamp for each file
- When write events stop for 2 seconds (debounce delay), the file is considered ready
- This handles files that are written incrementally or in chunks

#### 2. **Rename Event Detection**
- Many applications write to a temporary file first, then rename it to the final filename
- When a `Rename` event is detected, the file is immediately considered ready
- This provides instant detection for applications using the atomic write pattern

#### 3. **Combined Approach**
- **Create event**: Starts monitoring the file for write completion
- **Write events**: Updates the last write timestamp (debounce mechanism)
- **Rename event**: Immediately marks file as ready (atomic write pattern)
- The file is queued for upload when either:
  - Write events have stopped for 2 seconds (debounce), OR
  - A rename event is detected

#### Benefits
- **No fixed timeouts**: Adapts to files of any size automatically
- **Fast detection**: Rename events provide instant detection for atomic writes
- **Reliable**: Debouncing ensures files are fully written before upload
- **Efficient**: Works well with both small and large files (e.g., safetensors files)

This approach is particularly effective for:
- Large files that take time to write (model files, datasets, etc.)
- Applications that use atomic writes (temporary file → rename)
- Files written incrementally or in chunks
- Network file systems where write timing can vary

## Troubleshooting

### "credentials.json file not found"
- Make sure you've downloaded the credentials JSON file from Google Cloud Console
- Rename the downloaded file to exactly `credentials.json`
- Place the file in your home directory

### "Failed to configure OAuth2"
- Check that your JSON credentials are valid
- Ensure all required fields are present (client_id, client_secret, auth_uri, token_uri, etc.)

### "Access blocked: This app's request is invalid"
- Make sure you've enabled the Google Drive API in your Google Cloud project
- Verify that your OAuth consent screen is properly configured
- If using a test app, ensure your Google account is added as a test user

### "redirect_uri_mismatch" error
- The redirect URI in your credentials must match what the application expects
- For desktop apps, `http://localhost` is typically used
- Check your OAuth client configuration in Google Cloud Console

## Notes

- Files that already exist when the program starts are not uploaded (only newly created files are uploaded)
- The program uses intelligent file write detection (debounce + rename pattern) instead of fixed timeouts
- The `credentials.json` and `token.json` files are stored in your home directory and contain sensitive information and should never be committed to Git (already included in `.gitignore`)

## License

See the LICENSE file for details.
