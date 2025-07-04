#!/bin/bash

# Gmail OAuth2 Token Helper Script
# This script helps you get a refresh token for the email-tracker

echo "=== Gmail OAuth2 Token Generator ==="
echo
echo "This script will help you get a refresh token for Gmail API access."
echo

# Check if client ID and secret are provided
if [ $# -lt 2 ]; then
    echo "Usage: $0 <client-id> <client-secret>"
    echo
    echo "You can get these from Google Cloud Console:"
    echo "1. Go to https://console.cloud.google.com"
    echo "2. Select your project"
    echo "3. Go to 'APIs & Services' > 'Credentials'"
    echo "4. Create or use existing 'OAuth 2.0 Client ID'"
    echo "5. Application type should be 'Desktop app' or 'Web application'"
    echo "   - For Desktop: No redirect URI needed"
    echo "   - For Web: Add http://localhost:8090/callback as redirect URI"
    exit 1
fi

CLIENT_ID="$1"
CLIENT_SECRET="$2"

# Build the script if needed
echo "Building OAuth2 helper..."
cd "$(dirname "$0")"
go run get-gmail-token.go "$CLIENT_ID" "$CLIENT_SECRET"