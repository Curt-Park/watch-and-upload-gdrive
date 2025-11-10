# Watch and Upload to Google Drive (WUG)

A Go-based CLI tool that watches a directory and automatically uploads new files to Google Drive.

## Features

- Real-time directory monitoring
- Automatic file upload
- File extension filtering support
- Google Drive OAuth2 authentication

## Installation

1. Clone the repository:
```bash
git clone https://github.com/user/watch-and-upload-gdrive.git
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
./wug /path/to/directory
```

### Using Filter Option (Upload only specific file extensions)

```bash
./wug /path/to/directory --filter "*.safetensors"
```

Or using the short option:

```bash
./wug /path/to/directory -f "*.txt"
```

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
3. The program waits until the file is completely written (checks for file size stability)
4. If a filter is specified, the file extension is checked
5. The file is uploaded to Google Drive

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
- The program waits up to 30 seconds for a file to be completely written
- The `credentials.json` and `token.json` files are stored in your home directory and contain sensitive information and should never be committed to Git (already included in `.gitignore`)

## License

See the LICENSE file for details.
