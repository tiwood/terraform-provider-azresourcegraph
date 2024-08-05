# Authentication using Client Credentials
provider "azresourcegraph" {
  tenant_id     = var.tenant_id
  client_id     = var.client_id
  client_secret = var.client_secret
}

# Authentication using Azure Default Credential
# 1. Environment Variables
# 2. Managed Identity
# 3. Azure CLI
# 4. Azure Developer CLI
provider "azresourcegraph" {
  use_azure_default_credential = true
}
