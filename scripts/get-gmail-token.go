package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run get-gmail-token.go <client-id> <client-secret>")
		fmt.Println("\nThis tool will help you get a refresh token for Gmail API access.")
		fmt.Println("\nSteps:")
		fmt.Println("1. Run this tool with your client ID and secret")
		fmt.Println("2. Visit the URL printed by this tool")
		fmt.Println("3. Authorize the application")
		fmt.Println("4. Copy the authorization code from the redirect URL")
		fmt.Println("5. Paste it when prompted")
		os.Exit(1)
	}

	clientID := os.Args[1]
	clientSecret := os.Args[2]

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:8090/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/gmail.readonly",
		},
	}

	// Generate the authorization URL
	authURL := config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	
	fmt.Println("\n=== Gmail OAuth2 Token Generator ===")
	fmt.Println("\n1. Visit this URL in your browser:")
	fmt.Printf("\n%s\n", authURL)
	
	// Start local server to receive the callback
	authCode := make(chan string)
	server := &http.Server{Addr: ":8090"}
	
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			fmt.Fprintf(w, "Error: No authorization code received")
			return
		}
		
		fmt.Fprintf(w, `
<html>
<body>
<h1>Authorization Successful!</h1>
<p>You can close this window and return to the terminal.</p>
<p>Authorization code: <code>%s</code></p>
</body>
</html>`, code)
		
		authCode <- code
	})
	
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	
	fmt.Println("\n2. After authorizing, you'll be redirected to http://localhost:8090/callback")
	fmt.Println("   (The authorization code will be captured automatically)")
	fmt.Println("\nWaiting for authorization...")
	
	// Wait for the authorization code
	code := <-authCode
	
	// Shutdown the server
	server.Shutdown(context.Background())
	
	fmt.Printf("\n3. Received authorization code: %s\n", code)
	
	// Exchange the authorization code for tokens
	ctx := context.Background()
	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Fatalf("Failed to exchange token: %v", err)
	}
	
	fmt.Println("\n=== SUCCESS! ===")
	fmt.Println("\nYour tokens:")
	fmt.Printf("Access Token: %s\n", token.AccessToken)
	fmt.Printf("Refresh Token: %s\n", token.RefreshToken)
	fmt.Printf("Token Type: %s\n", token.TokenType)
	fmt.Printf("Expiry: %v\n", token.Expiry)
	
	// Save tokens to file
	tokenFile := "gmail-tokens.json"
	file, err := os.Create(tokenFile)
	if err != nil {
		log.Printf("Warning: Could not save tokens to file: %v", err)
	} else {
		defer file.Close()
		json.NewEncoder(file).Encode(token)
		fmt.Printf("\nTokens saved to: %s\n", tokenFile)
	}
	
	fmt.Println("\n=== Configuration for email-tracker ===")
	fmt.Println("\nAdd these to your .env file or export as environment variables:")
	fmt.Printf("GMAIL_CLIENT_ID=%s\n", clientID)
	fmt.Printf("GMAIL_CLIENT_SECRET=%s\n", clientSecret)
	fmt.Printf("GMAIL_REFRESH_TOKEN=%s\n", token.RefreshToken)
}