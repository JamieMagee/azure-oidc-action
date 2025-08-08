# Azure OIDC Action

A GitHub Action that authenticates with Azure using GitHub Actions OIDC tokens via the Azure `ClientAssertionCredential`.

## Overview

This action is designed to run in GitHub Actions workflows and uses the GitHub OIDC provider to authenticate with Azure AD without requiring long-lived secrets. It leverages the `azidentity.ClientAssertionCredential` to create a credential that can be used with any Azure SDK client.

## Usage

### Basic Usage

```yaml
- name: Authenticate with Azure
  uses: your-username/azure-oidc-action@v1
  with:
    tenant-id: ${{ secrets.AZURE_TENANT_ID }}
    client-id: ${{ secrets.AZURE_CLIENT_ID }}
```

### Advanced Usage with Token Output

⚠️ **Security Warning**: Outputting the Azure token makes it visible in workflow logs and can be accessed by subsequent steps. Only enable this if you need the raw token for custom integrations.

```yaml
- name: Authenticate with Azure (with token output)
  id: azure-auth
  uses: your-username/azure-oidc-action@v1
  with:
    tenant-id: ${{ secrets.AZURE_TENANT_ID }}
    client-id: ${{ secrets.AZURE_CLIENT_ID }}
    output-token: true  # Enable token output (security consideration)

- name: Use Azure token directly
  run: |
    echo "Token expires at: ${{ steps.azure-auth.outputs.token-expiry }}"
    # Use token for custom Azure REST API calls
    curl -H "Authorization: Bearer ${{ steps.azure-auth.outputs.azure-token }}" \
         "https://management.azure.com/subscriptions?api-version=2020-01-01"
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `tenant-id` | Azure AD tenant ID | Yes | - |
| `client-id` | Azure AD application client ID | Yes | - |
| `audience` | OIDC token audience | No | `api://AzureADTokenExchange` |
| `output-token` | Output the Azure token (security consideration) | No | `false` |

## Outputs

| Output | Description |
|--------|-------------|
| `token-expiry` | The expiry time of the Azure token (RFC3339 format) |
| `azure-token` | The Azure access token (only output if `output-token` is `true`) |

## Prerequisites

1. **Azure App Registration**: Create an Azure AD application with federated credentials configured for GitHub OIDC
2. **GitHub Actions Workflow**: The action must run within a GitHub Actions workflow with `id-token: write` permission

## Azure Setup

### 1. Create Azure App Registration

```bash
# Create the app registration
az ad app create --display-name "MyApp-GitHub-OIDC"

# Get the application ID (client ID)
az ad app list --display-name "MyApp-GitHub-OIDC" --query "[0].appId" -o tsv
```

### 2. Configure Federated Credentials

```bash
# Replace with your values
APP_ID="your-app-id"
GITHUB_ORG="your-github-org"
GITHUB_REPO="your-github-repo"

# Create federated credential for main branch
az ad app federated-credential create \
  --id $APP_ID \
  --parameters '{
    "name": "github-main",
    "issuer": "https://token.actions.githubusercontent.com",
    "subject": "repo:'$GITHUB_ORG'/'$GITHUB_REPO':ref:refs/heads/main",
    "audiences": ["api://AzureADTokenExchange"]
  }'
```

### 3. Assign Azure Permissions

```bash
# Example: Assign Contributor role to a resource group
az role assignment create \
  --assignee $APP_ID \
  --role "Contributor" \
  --scope "/subscriptions/your-subscription-id/resourceGroups/your-resource-group"
```

## GitHub Actions Workflow Example

```yaml
name: Azure OIDC Authentication

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  id-token: write    # Required for OIDC token
  contents: read     # Required for checkout

jobs:
  azure-auth:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Authenticate with Azure
        uses: your-username/azure-oidc-action@v1
        with:
          tenant-id: ${{ secrets.AZURE_TENANT_ID }}
          client-id: ${{ secrets.AZURE_CLIENT_ID }}

      # Now you can use Azure CLI or other Azure tools
      - name: Azure CLI example
        run: |
          # Install Azure CLI if needed
          curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
          
          # The credential is now available for Azure operations
          az account show
```

## CLI Tool Usage (Alternative)

If you prefer to use this as a standalone CLI tool instead of a GitHub Action:

### Build the Tool

```bash
go build -o azure-oidc-action
```

### Command Line Options

```bash
./azure-oidc-action --help
```

**Required flags:**

- `--tenant-id`: Azure AD tenant ID
- `--client-id`: Azure AD application client ID

**Optional flags:**

- `--audience`: OIDC token audience (default: `api://AzureADTokenExchange`)
- `--output-token`: Output the Azure token (default: `false`, security consideration)
- `--help`: Show help message

### Environment Variables

The tool requires these environment variables, which are automatically set by GitHub Actions:

- `ACTIONS_ID_TOKEN_REQUEST_URL`: GitHub OIDC provider URL
- `ACTIONS_ID_TOKEN_REQUEST_TOKEN`: Bearer token for OIDC provider

## Repository Secrets

Add these secrets to your GitHub repository:

- `AZURE_TENANT_ID`: Your Azure AD tenant ID
- `AZURE_CLIENT_ID`: Your Azure AD application client ID

## How It Works

1. **OIDC Token Request**: The tool requests an OIDC token from GitHub using the environment variables set by GitHub Actions
2. **Client Assertion**: The GitHub OIDC token is used as a client assertion for Azure AD authentication
3. **Azure Token**: Azure AD validates the GitHub OIDC token and issues an Azure access token
4. **Credential Creation**: A `ClientAssertionCredential` is created that can be used with Azure SDK clients

## Security Benefits

- **No long-lived secrets**: No need to store Azure credentials in GitHub secrets
- **Short-lived tokens**: OIDC tokens are short-lived and automatically rotated
- **Scoped access**: Federated credentials can be scoped to specific repositories, branches, or environments
- **Auditable**: All authentication events are logged in Azure AD

## Subject Claim Examples

Configure different subject claims for different scenarios:

```bash
# Specific branch
"subject": "repo:org/repo:ref:refs/heads/main"

# Specific environment
"subject": "repo:org/repo:environment:production"

# Pull requests
"subject": "repo:org/repo:pull_request"

# Specific tag
"subject": "repo:org/repo:ref:refs/tags/v1.0.0"
```

## Troubleshooting

### Common Issues

1. **Missing OIDC environment variables**
   - Ensure `id-token: write` permission is set in the workflow
   - Check that the workflow is running on a supported runner

2. **Azure authentication failures**
   - Verify the app registration and federated credential configuration
   - Check that the subject claim matches your repository/branch
   - Ensure the audience matches (`api://AzureADTokenExchange`)

3. **Token validation errors**
   - Verify the tenant ID and client ID are correct
   - Check that the app has the necessary Azure permissions

### Debug Mode

You can add debug output by examining the OIDC token claims:

```bash
# Install jq for JSON parsing
sudo apt-get install jq

# Decode the JWT token (without verification)
echo "JWT_TOKEN_HERE" | cut -d. -f2 | base64 -d | jq .
```

## Dependencies

- `github.com/Azure/azure-sdk-for-go/sdk/azidentity`: Azure identity library
- `github.com/Azure/azure-sdk-for-go/sdk/azcore`: Azure core library

## License

This project is licensed under the MIT License.
