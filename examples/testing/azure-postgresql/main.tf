# Azure PostgreSQL + CyberArk SIA Integration Test
# This configuration creates an Azure PostgreSQL Flexible Server (B1ms - cheapest option)
# and onboards it to CyberArk SIA for privileged access management

terraform {
  required_version = ">= 1.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
    cyberarksia = {
      source  = "aaearon/cyberarksia"
      version = "0.1.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}

provider "azurerm" {
  features {}
  subscription_id = var.azure_subscription_id
}

provider "cyberarksia" {
  username = var.sia_username
  password = var.sia_password
}

# Generate unique suffix for resource names
resource "random_string" "suffix" {
  length  = 6
  special = false
  upper   = false
}

# Resource group for all Azure resources
resource "azurerm_resource_group" "sia_test" {
  name     = "rg-sia-test-${random_string.suffix.result}"
  location = var.azure_region

  # STATIC tags only - NO timestamp() to avoid unnecessary updates!
  tags = {
    environment = "test"
    purpose     = "sia-provider-validation"
    managed_by  = "terraform"
    auto_delete = "true"
    created_on  = "2025-10-28"
  }
}

# Azure PostgreSQL Flexible Server (B1ms - cheapest option)
resource "azurerm_postgresql_flexible_server" "sia_test" {
  name                = "psql-sia-test-${random_string.suffix.result}"
  resource_group_name = azurerm_resource_group.sia_test.name
  location            = azurerm_resource_group.sia_test.location

  # B1ms SKU - Burstable tier (1 vCore, 2GB RAM) - ~$12.41/month
  sku_name   = "B_Standard_B1ms"
  storage_mb = 32768 # 32 GB (minimum for Flexible Server)

  # PostgreSQL version
  version = "16"

  # Admin credentials
  administrator_login    = var.postgres_admin_username
  administrator_password = var.postgres_admin_password

  # Public access enabled (required for SIA connectivity)
  public_network_access_enabled = true

  # High availability disabled (not needed for testing, saves cost)
  zone = "1"

  # Backup configuration (minimum settings)
  backup_retention_days        = 7
  geo_redundant_backup_enabled = false

  tags = azurerm_resource_group.sia_test.tags
}

# Firewall rule: Allow Azure services
resource "azurerm_postgresql_flexible_server_firewall_rule" "allow_azure_services" {
  name             = "AllowAzureServices"
  server_id        = azurerm_postgresql_flexible_server.sia_test.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "0.0.0.0" # Special Azure magic IP for Azure services
}

# Firewall rule: Allow all IPs (for testing - SIA can connect from anywhere)
# WARNING: In production, restrict to SIA IP ranges only
resource "azurerm_postgresql_flexible_server_firewall_rule" "allow_all" {
  name             = "AllowAll_TestOnly"
  server_id        = azurerm_postgresql_flexible_server.sia_test.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "255.255.255.255"
}

# Database configuration (optional - ensure PostgreSQL is ready)
resource "azurerm_postgresql_flexible_server_database" "testdb" {
  name      = "testdb"
  server_id = azurerm_postgresql_flexible_server.sia_test.id
  charset   = "UTF8"
  collation = "en_US.utf8"
}

# ============================================================================
# CyberArk SIA Resources
# ============================================================================

# ============================================================================
# Policy Assignment - Azure Database to Terraform-Test-Policy
# ============================================================================
# Assigns Azure PostgreSQL database to existing "Terraform-Test-Policy"
# All databases use "FQDN/IP" target set regardless of cloud provider
# ============================================================================

data "cyberarksia_database_policy" "terraform_test" {
  name = "Terraform-Test-Policy"
}

resource "cyberarksia_database_policy_workspace_assignment" "azure_postgres_assignment" {
  policy_id             = data.cyberarksia_database_policy.terraform_test.id
  database_workspace_id = cyberarksia_database_workspace.azure_postgres.id
  authentication_method = "db_auth"

  db_auth_profile {
    roles = ["pg_read_all_settings"]
  }

  depends_on = [
    cyberarksia_database_workspace.azure_postgres
  ]
}

# Certificate: Azure PostgreSQL TLS/SSL certificate
# Azure PostgreSQL Flexible Server uses Microsoft RSA Root Certificate Authority 2017
resource "cyberarksia_certificate" "azure_postgres_cert" {
  cert_name        = "azure-postgres-ssl-cert-${random_string.suffix.result}"
  cert_description = "Microsoft RSA Root CA 2017 for Azure PostgreSQL Flexible Server TLS/SSL validation"
  cert_type        = "PEM"
  cert_body        = <<-EOT
-----BEGIN CERTIFICATE-----
MIIFqDCCA5CgAwIBAgIQHtOXCV/YtLNHcB6qvn9FszANBgkqhkiG9w0BAQwFADBl
MQswCQYDVQQGEwJVUzEeMBwGA1UEChMVTWljcm9zb2Z0IENvcnBvcmF0aW9uMTYw
NAYDVQQDEy1NaWNyb3NvZnQgUlNBIFJvb3QgQ2VydGlmaWNhdGUgQXV0aG9yaXR5
IDIwMTcwHhcNMTkxMjE4MjI1MTIyWhcNNDIwNzE4MjMwMDIzWjBlMQswCQYDVQQG
EwJVUzEeMBwGA1UEChMVTWljcm9zb2Z0IENvcnBvcmF0aW9uMTYwNAYDVQQDEy1N
aWNyb3NvZnQgUlNBIFJvb3QgQ2VydGlmaWNhdGUgQXV0aG9yaXR5IDIwMTcwggIi
MA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQDKW76UM4wplZEWCpW9R2LBifOZ
Nt9GkMml7Xhqb0eRaPgnZ1AzHaGm++DlQ6OEAlcBXZxIQIJTELy/xztokLaCLeX0
ZdDMbRnMlfl7rEqUrQ7eS0MdhweSE5CAg2Q1OQT85elss7YfUJQ4ZVBcF0a5toW1
HLUX6NZFndiyJrDKxHBKrmCk3bPZ7Pw71VdyvD/IybLeS2v4I2wDwAW9lcfNcztm
gGTjGqwu+UcF8ga2m3P1eDNbx6H7JyqhtJqRjJHTOoI+dkC0zVJhUXAoP8XFWvLJ
jEm7FFtNyP9nTUwSlq31/niol4fX/V4ggNyhSyL71Imtus5Hl0dVe49FyGcohJUc
aDDv70ngNXtk55iwlNpNhTs+VcQor1fznhPbRiefHqJeRIOkpcrVE7NLP8TjwuaG
YaRSMLl6IE9vDzhTyzMMEyuP1pq9KsgtsRx9S1HKR9FIJ3Jdh+vVReZIZZ2vUpC6
W6IYZVcSn2i51BVrlMRpIpj0M+Dt+VGOQVDJNE92kKz8OMHY4Xu54+OU4UZpyw4K
UGsTuqwPN1q3ErWQgR5WrlcihtnJ0tHXUeOrO8ZV/R4O03QK0dqq6mm4lyiPSMQH
+FJDOvTKVTUssKZqwJz58oHhEmrARdlns87/I6KJClTUFLkqqNfs+avNJVgyeY+Q
W5g5xAgGwax/Dj0ApQIDAQABo1QwUjAOBgNVHQ8BAf8EBAMCAYYwDwYDVR0TAQH/
BAUwAwEB/zAdBgNVHQ4EFgQUCctZf4aycI8awznjwNnpv7tNsiMwEAYJKwYBBAGC
NxUBBAMCAQAwDQYJKoZIhvcNAQEMBQADggIBAKyvPl3CEZaJjqPnktaXFbgToqZC
LgLNFgVZJ8og6Lq46BrsTaiXVq5lQ7GPAJtSzVXNUzltYkyLDVt8LkS/gxCP81OC
gMNPOsduET/m4xaRhPtthH80dK2Jp86519efhGSSvpWhrQlTM93uCupKUY5vVau6
tZRGrox/2KJQJWVggEbbMwSubLWYdFQl3JPk+ONVFT24bcMKpBLBaYVu32TxU5nh
SnUgnZUP5NbcA/FZGOhHibJXWpS2qdgXKxdJ5XbLwVaZOjex/2kskZGT4d9Mozd2
TaGf+G0eHdP67Pv0RR0Tbc/3WeUiJ3IrhvNXuzDtJE3cfVa7o7P4NHmJweDyAmH3
pvwPuxwXC65B2Xy9J6P9LjrRk5Sxcx0ki69bIImtt2dmefU6xqaWM/5TkshGsRGR
xpl/j8nWZjEgQRCHLQzWwa80mMpkg/sTV9HB8Dx6jKXB/ZUhoHHBk2dxEuqPiApp
GWSZI1b7rCoucL5mxAyE7+WL85MB+GqQk2dLsmijtWKP6T+MejteD+eMuMZ87zf9
dOLITzNy4ZQ5bb0Sr74MTnB8G2+NszKTc0QWbej09+CVgI+WXTik9KveCjCHk9hN
AHFiRSdLOkKEW39lt2c0Ui2cFmuqqNh7o0JMcccMyj6D5KbvtwEwXlGjefVwaaZB
RA+GsCyRxj3qrg+E
-----END CERTIFICATE-----
EOT

  labels = {
    provider    = "azure"
    database    = "postgresql"
    cert_type   = "root-ca"
    description = "microsoft-rsa-root-ca-2017"
  }
}

# Secret: Database admin credentials
resource "cyberarksia_database_secret" "postgres_admin" {
  name                = "azure-postgres-admin-${random_string.suffix.result}"
  authentication_type = "local"
  username            = var.postgres_admin_username
  password            = var.postgres_admin_password

  # Wait for PostgreSQL server to be fully provisioned
  depends_on = [
    azurerm_postgresql_flexible_server.sia_test,
    azurerm_postgresql_flexible_server_firewall_rule.allow_all
  ]
}

# Database Workspace: Azure PostgreSQL
resource "cyberarksia_database_workspace" "azure_postgres" {
  name          = "azure-postgres-test-${random_string.suffix.result}"
  database_type = "postgres-azure-managed"

  # Azure-specific settings
  cloud_provider = "azure"
  region         = var.azure_region

  # Connection details
  address = azurerm_postgresql_flexible_server.sia_test.fqdn
  port    = 5432

  # Secret reference (required for ZSP/JIT)
  secret_id = cyberarksia_database_secret.postgres_admin.id

  # Certificate validation (Azure uses valid TLS certs)
  enable_certificate_validation = false
  certificate_id                = cyberarksia_certificate.azure_postgres_cert.id

  # Metadata (UPDATED for CRUD test)
  tags = {
    environment    = "test"
    cloud_provider = "azure"
    database_type  = "postgresql"
    sku            = "B1ms"
    test_purpose   = "sia-provider-validation"
    test_phase     = "crud-update-validation"
    updated_at     = "2025-10-27"
  }

  # Ensure PostgreSQL is fully accessible before creating workspace
  depends_on = [
    azurerm_postgresql_flexible_server_database.testdb,
    cyberarksia_database_secret.postgres_admin,
    cyberarksia_certificate.azure_postgres_cert # Certificate must exist first
  ]
}
