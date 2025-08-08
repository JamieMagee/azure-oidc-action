#!/bin/sh

set -e

echo "🔐 Starting Azure OIDC Authentication..."

# Run the azure-oidc-action binary with all provided arguments
./azure-oidc-action "$@"

# Capture the exit code
exit_code=$?

if [ $exit_code -eq 0 ]; then
    echo "✅ Successfully authenticated with Azure!"
    echo "🎯 Azure credentials are now available for subsequent steps"
    
    # Read token expiry from temp file if it exists
    if [ -f "/tmp/token_expiry" ]; then
        token_expiry=$(cat /tmp/token_expiry)
        echo "token-expiry=$token_expiry" >> $GITHUB_OUTPUT
    else
        # Fallback to current time if file doesn't exist
        echo "token-expiry=$(date -u +%Y-%m-%dT%H:%M:%SZ)" >> $GITHUB_OUTPUT
    fi
    
    # Read azure token from temp file if it exists (only if --output-token was used)
    if [ -f "/tmp/azure_token" ]; then
        azure_token=$(cat /tmp/azure_token)
        echo "azure-token=$azure_token" >> $GITHUB_OUTPUT
        echo "🔐 Azure token has been output (handle securely!)"
        
        # Clean up the token file for security
        rm -f /tmp/azure_token
    fi
    
    # Clean up the expiry file
    rm -f /tmp/token_expiry
else
    echo "❌ Failed to authenticate with Azure (exit code: $exit_code)"
    exit $exit_code
fi
