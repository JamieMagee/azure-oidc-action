package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// OIDCToken represents the structure of the GitHub OIDC token response
type OIDCToken struct {
	Value string `json:"value"`
}

func main() {
	// Define CLI flags
	var tenantID = flag.String("tenant-id", "", "Azure AD tenant ID (required)")
	var clientID = flag.String("client-id", "", "Azure AD application client ID (required)")
	var audience = flag.String("audience", "api://AzureADTokenExchange", "OIDC token audience (optional)")
	var outputToken = flag.Bool("output-token", false, "Output the Azure token (security consideration)")
	var help = flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Validate required parameters
	if *tenantID == "" || *clientID == "" {
		fmt.Fprintf(os.Stderr, "Error: Both --tenant-id and --client-id are required\n\n")
		showHelp()
		os.Exit(1)
	}

	// Check for required GitHub Actions environment variables
	requestURL := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")
	requestToken := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")

	if requestURL == "" || requestToken == "" {
		log.Fatal("Error: GitHub Actions OIDC environment variables not found.\n" +
			"This tool must be run within a GitHub Actions workflow with 'id-token: write' permission.\n" +
			"Required environment variables:\n" +
			"  - ACTIONS_ID_TOKEN_REQUEST_URL\n" +
			"  - ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	}

	fmt.Printf("Requesting GitHub OIDC token for audience: %s\n", *audience)

	// Get the GitHub OIDC token
	oidcToken, err := getGitHubOIDCToken(requestURL, requestToken, *audience)
	if err != nil {
		log.Fatalf("Failed to get GitHub OIDC token: %v", err)
	}

	fmt.Println("Successfully obtained GitHub OIDC token")

	// Create the ClientAssertionCredential
	credential, err := azidentity.NewClientAssertionCredential(
		*tenantID,
		*clientID,
		func(ctx context.Context) (string, error) {
			// Return the GitHub OIDC token as the client assertion
			return oidcToken, nil
		},
		nil, // Use default options
	)
	if err != nil {
		log.Fatalf("Failed to create ClientAssertionCredential: %v", err)
	}

	fmt.Println("Successfully created ClientAssertionCredential")

	// Test the credential by requesting a token
	fmt.Println("Testing credential by requesting an Azure token...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"499b84ac-1321-427f-aa17-267ca6975798/.default"},
	})
	if err != nil {
		log.Fatalf("Failed to get Azure token: %v", err)
	}

	fmt.Printf("Successfully authenticated with Azure!\n")
	fmt.Printf("Token expires at: %s\n", token.ExpiresOn.Format(time.RFC3339))
	
	// Write token expiry to a temp file for entrypoint to read
	if err := os.WriteFile("/tmp/token_expiry", []byte(token.ExpiresOn.Format(time.RFC3339)), 0644); err != nil {
		log.Printf("Warning: Failed to write token expiry: %v", err)
	}
	
	if *outputToken {
		fmt.Printf("Token preview: %s...\n", token.Token[:min(len(token.Token), 50)])
		
		// Write the full token to a temp file for entrypoint to read
		if err := os.WriteFile("/tmp/azure_token", []byte(token.Token), 0600); err != nil {
			log.Printf("Warning: Failed to write azure token: %v", err)
		}
		
		fmt.Println("⚠️  WARNING: Azure token will be output as action output.")
		fmt.Println("   Make sure to handle this token securely in your workflow.")
	} else {
		fmt.Printf("Token preview: %s...\n", token.Token[:min(len(token.Token), 50)])
		fmt.Println("💡 Use --output-token=true to output the full token as an action output.")
	}

	fmt.Println("\nCredential is ready for use with Azure SDK clients.")
}

// getGitHubOIDCToken requests an OIDC token from GitHub Actions
func getGitHubOIDCToken(requestURL, requestToken, audience string) (string, error) {
	// Construct the request URL with audience parameter
	separator := "&"
	if !strings.Contains(requestURL, "?") {
		separator = "?"
	}
	url := fmt.Sprintf("%s%saudience=%s", requestURL, separator, audience)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	// Create the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", "bearer "+requestToken)
	req.Header.Set("Accept", "application/json")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub OIDC API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var tokenResponse OIDCToken
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if tokenResponse.Value == "" {
		return "", fmt.Errorf("empty token received from GitHub OIDC API")
	}

	return tokenResponse.Value, nil
}

// showHelp displays usage information
func showHelp() {
	fmt.Printf("Azure OIDC Action - GitHub Actions OIDC to Azure AD Authentication\n\n")
	fmt.Printf("This tool creates an Azure ClientAssertionCredential using GitHub Actions OIDC tokens.\n")
	fmt.Printf("It must be run within a GitHub Actions workflow with 'id-token: write' permission.\n\n")
	fmt.Printf("Usage:\n")
	fmt.Printf("  azure-oidc-action --tenant-id <tenant-id> --client-id <client-id> [options]\n\n")
	fmt.Printf("Required flags:\n")
	fmt.Printf("  --tenant-id    Azure AD tenant ID\n")
	fmt.Printf("  --client-id    Azure AD application client ID\n\n")
	fmt.Printf("Optional flags:\n")
	fmt.Printf("  --audience       OIDC token audience (default: api://AzureADTokenExchange)\n")
	fmt.Printf("  --output-token   Output the Azure token (default: false, security consideration)\n")
	fmt.Printf("  --help           Show this help message\n\n")
	fmt.Printf("Required environment variables (set by GitHub Actions):\n")
	fmt.Printf("  ACTIONS_ID_TOKEN_REQUEST_URL    GitHub OIDC provider URL\n")
	fmt.Printf("  ACTIONS_ID_TOKEN_REQUEST_TOKEN  Bearer token for OIDC provider\n\n")
	fmt.Printf("Example GitHub Actions workflow:\n")
	fmt.Printf("  permissions:\n")
	fmt.Printf("    id-token: write\n")
	fmt.Printf("    contents: read\n")
	fmt.Printf("  steps:\n")
	fmt.Printf("    - uses: actions/checkout@v4\n")
	fmt.Printf("    - name: Azure Login\n")
	fmt.Printf("      run: |\n")
	fmt.Printf("        azure-oidc-action \\\n")
	fmt.Printf("          --tenant-id ${{ secrets.AZURE_TENANT_ID }} \\\n")
	fmt.Printf("          --client-id ${{ secrets.AZURE_CLIENT_ID }}\n")
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
