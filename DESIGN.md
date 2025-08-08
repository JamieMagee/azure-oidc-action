# Azure OIDC Action - Technical Design

## Overview

This tool implements Azure AD authentication using GitHub Actions OIDC tokens through the Azure SDK's `ClientAssertionCredential`. It eliminates the need for long-lived secrets by leveraging GitHub's OIDC provider for workload identity federation.

## Architecture

### Authentication Flow

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│                 │    │                 │    │                 │
│  GitHub Actions │───▶│   GitHub OIDC   │───▶│    Azure AD     │
│   Workflow      │    │    Provider     │    │                 │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                        │                        │
         │                        │                        │
         ▼                        ▼                        ▼
   Request Token              Issue JWT                 Validate JWT
                             (with claims)              & Issue Token
```

### Key Components

1. **OIDC Token Acquisition**
   - Uses GitHub Actions environment variables (`ACTIONS_ID_TOKEN_REQUEST_URL`, `ACTIONS_ID_TOKEN_REQUEST_TOKEN`)
   - Requests JWT token from GitHub's OIDC provider
   - Configurable audience (default: `api://AzureADTokenExchange`)

2. **Client Assertion Creation**
   - GitHub OIDC token serves as the client assertion
   - Implements callback function for `ClientAssertionCredential`
   - Token is refreshed automatically when needed

3. **Azure Authentication**
   - Uses `azidentity.ClientAssertionCredential`
   - Exchanges GitHub OIDC token for Azure access token
   - Credential can be used with any Azure SDK client

## Security Model

### Trust Relationship

- **Azure Side**: Federated credential configuration defines trust conditions
  - Tenant ID + Client ID identify the Azure app registration
  - Subject claims restrict access to specific repositories/branches/environments
  - Audience claims ensure tokens are intended for Azure

- **GitHub Side**: OIDC token includes verifiable claims
  - Repository, branch, environment information
  - Actor (user) and workflow details
  - Cryptographically signed by GitHub

### Token Lifecycle

1. **GitHub OIDC Token**: Short-lived (typically 15 minutes)
2. **Azure Access Token**: Managed by Azure SDK (typically 1 hour)
3. **Automatic Refresh**: Azure SDK handles token refresh using the callback

## Configuration

### Azure App Registration

```json
{
  "appId": "client-id",
  "federatedCredentials": [
    {
      "name": "github-main",
      "issuer": "https://token.actions.githubusercontent.com",
      "subject": "repo:org/repo:ref:refs/heads/main",
      "audiences": ["api://AzureADTokenExchange"]
    }
  ]
}
```

### GitHub Workflow

```yaml
permissions:
  id-token: write    # Required for OIDC token
  contents: read     # Standard permission

jobs:
  job-name:
    steps:
      - name: Azure Auth
        run: |
          azure-oidc-action \
            --tenant-id ${{ secrets.AZURE_TENANT_ID }} \
            --client-id ${{ secrets.AZURE_CLIENT_ID }}
```

## Implementation Details

### Error Handling

- **Missing Environment Variables**: Clear error message for missing GitHub OIDC variables
- **Invalid Parameters**: Validation of required CLI arguments
- **Network Failures**: HTTP timeout and retry logic
- **Authentication Failures**: Detailed error messages for Azure AD issues

### HTTP Client Configuration

- TLS 1.2+ requirement
- 30-second timeout for GitHub OIDC requests
- Bearer authentication for GitHub API
- JSON content type handling

### Token Validation

- GitHub OIDC token format validation
- Non-empty token verification
- HTTP status code checking
- JSON response parsing

## Extension Points

### Custom Audiences

The tool supports custom audience values for different Azure scenarios:

```bash
# Default Azure audience
--audience api://AzureADTokenExchange

# Custom audience for specific apps
--audience api://custom-app-id
```

### Multiple Environments

Subject claims can be configured for different deployment environments:

```bash
# Production environment
"subject": "repo:org/repo:environment:production"

# Staging environment  
"subject": "repo:org/repo:environment:staging"
```

### Multi-tenancy

Different tenant IDs can be used for different scenarios:

```bash
# Production tenant
--tenant-id prod-tenant-id

# Development tenant
--tenant-id dev-tenant-id
```

## Best Practices

### Security

1. **Principle of Least Privilege**: Configure minimal Azure permissions
2. **Scope Restrictions**: Use specific subject claims (branch, environment)
3. **Secret Management**: Store tenant/client IDs in GitHub secrets
4. **Audit Logging**: Monitor Azure AD sign-in logs

### Reliability

1. **Error Handling**: Always check authentication success
2. **Timeout Configuration**: Use appropriate timeouts for network calls
3. **Retry Logic**: Implement retries for transient failures
4. **Health Checks**: Verify credentials before critical operations

### Maintenance

1. **Credential Rotation**: GitHub automatically rotates OIDC tokens
2. **Certificate Management**: No certificates to manage
3. **Monitoring**: Monitor Azure AD authentication events
4. **Updates**: Keep Azure SDK dependencies updated

## Integration Examples

### Azure Resource Manager

```go
import "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

client, err := armresources.NewResourceGroupsClient(subscriptionID, credential, nil)
```

### Azure Key Vault

```go
import "github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"

client, err := azsecrets.NewClient(vaultURL, credential, nil)
```

### Azure Storage

```go
import "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

client, err := azblob.NewClient(accountURL, credential, nil)
```
