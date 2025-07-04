# Gmail OAuth2 Setup Guide

This guide explains how to properly set up Gmail OAuth2 credentials for the email-tracker program.

## Prerequisites

1. A Google Cloud Project
2. Gmail API enabled
3. OAuth 2.0 credentials created

## Step 1: Set Up Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a new project or select an existing one
3. Enable the Gmail API:
   - Go to "APIs & Services" > "Library"
   - Search for "Gmail API"
   - Click on it and press "Enable"

## Step 2: Create OAuth2 Credentials

1. Go to "APIs & Services" > "Credentials"
2. Click "+ CREATE CREDENTIALS" > "OAuth client ID"
3. If prompted, configure the OAuth consent screen first:
   - Choose "External" user type (unless using Google Workspace)
   - Fill in required fields (app name, user support email, developer email)
   - Add scope: `https://www.googleapis.com/auth/gmail.readonly`
   - Add your email as a test user if in testing mode
4. For Application type:
   - Choose "Desktop app" (recommended) OR
   - Choose "Web application" and add `http://localhost:8090/callback` as an authorized redirect URI
5. Give it a name (e.g., "Email Tracker")
6. Click "Create"
7. Save the Client ID and Client Secret

## Step 3: Get the Refresh Token

You need to complete the OAuth2 authorization flow to get a refresh token. Choose one of these methods:

### Method A: Using Our Helper Script (Recommended)

1. Navigate to the scripts directory:
   ```bash
   cd /home/jhjaggars/code/package-tracking/scripts
   ```

2. Run the OAuth2 helper script:
   ```bash
   ./get-gmail-token.sh YOUR_CLIENT_ID YOUR_CLIENT_SECRET
   ```

3. The script will:
   - Start a local web server on port 8090
   - Print an authorization URL
   - Open your browser to the URL (or copy/paste it)
   - Capture the authorization code automatically
   - Exchange it for tokens
   - Display your refresh token

4. Copy the refresh token from the output

### Method B: Using Google OAuth2 Playground

1. Go to [Google OAuth2 Playground](https://developers.google.com/oauthplayground/)
2. Click the gear icon (⚙️) in the top right
3. Check "Use your own OAuth credentials"
4. Enter your Client ID and Client Secret
5. Close the settings
6. In Step 1, find "Gmail API v1" and select:
   - `https://www.googleapis.com/auth/gmail.readonly`
7. Click "Authorize APIs"
8. Sign in and grant permissions
9. In Step 2, click "Exchange authorization code for tokens"
10. Copy the "Refresh token" value

### Method C: Manual Flow

If the above methods don't work, you can manually construct the flow:

1. Construct the authorization URL:
   ```
   https://accounts.google.com/o/oauth2/v2/auth?
   client_id=YOUR_CLIENT_ID&
   redirect_uri=http://localhost:8090/callback&
   response_type=code&
   scope=https://www.googleapis.com/auth/gmail.readonly&
   access_type=offline&
   prompt=consent
   ```

2. Visit the URL and authorize the application

3. Copy the authorization code from the redirect URL

4. Exchange the code for tokens using curl:
   ```bash
   curl -X POST https://oauth2.googleapis.com/token \
     -d "code=AUTHORIZATION_CODE" \
     -d "client_id=YOUR_CLIENT_ID" \
     -d "client_secret=YOUR_CLIENT_SECRET" \
     -d "redirect_uri=http://localhost:8090/callback" \
     -d "grant_type=authorization_code"
   ```

5. Extract the refresh_token from the JSON response

## Step 4: Configure email-tracker

Once you have your refresh token, configure the email-tracker:

### Using Environment Variables
```bash
export GMAIL_CLIENT_ID="your-client-id"
export GMAIL_CLIENT_SECRET="your-client-secret"
export GMAIL_REFRESH_TOKEN="your-refresh-token"
```

### Using .env File
Create a `.env` file in your project root:
```env
GMAIL_CLIENT_ID=your-client-id
GMAIL_CLIENT_SECRET=your-client-secret
GMAIL_REFRESH_TOKEN=your-refresh-token
EMAIL_API_URL=http://localhost:8080
EMAIL_CHECK_INTERVAL=5m
```

### Run the email-tracker
```bash
./bin/email-tracker
# Or with a custom config file
./bin/email-tracker --config=.env.production
```

## Troubleshooting

### "Refresh token not provided" Error
- Make sure you included `access_type=offline` in the authorization URL
- Ensure you selected the correct scopes
- Try adding `prompt=consent` to force re-authorization

### "Invalid client" Error
- Verify your Client ID and Client Secret are correct
- Check that the redirect URI matches exactly what's configured in Google Cloud Console
- Ensure the OAuth2 client type matches your setup (Desktop vs Web)

### "Access blocked" Error
- Make sure the Gmail API is enabled in your Google Cloud project
- If your app is in testing mode, add your email as a test user
- For production, you may need to verify your OAuth consent screen

### Token Expiration
- Access tokens expire after 1 hour
- Refresh tokens don't expire unless:
  - The user revokes access
  - The token hasn't been used for 6 months
  - You've exceeded the limit of 50 refresh tokens per user per client

## Security Notes

1. **Never commit tokens to git**: Add these to `.gitignore`:
   ```
   .env
   .env.*
   gmail-tokens.json
   *-token.json
   ```

2. **Secure storage**: In production, use:
   - Environment variables from secure sources
   - Secret management systems (Vault, AWS Secrets Manager, etc.)
   - Encrypted configuration files

3. **Minimal scopes**: Only request `gmail.readonly` scope unless you need write access

4. **Token rotation**: Implement token refresh logic in your application

## Additional Resources

- [Gmail API Documentation](https://developers.google.com/gmail/api)
- [OAuth 2.0 for Web Server Applications](https://developers.google.com/identity/protocols/oauth2/web-server)
- [OAuth 2.0 Scopes for Google APIs](https://developers.google.com/identity/protocols/oauth2/scopes#gmail)