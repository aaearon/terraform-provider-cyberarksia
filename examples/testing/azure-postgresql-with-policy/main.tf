# ============================================================================
# Terraform Configuration
# ============================================================================

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
    cyberarksia = {
      source  = "aaearon/cyberarksia"
      version = "0.1.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

# ============================================================================
# Provider Configuration
# ============================================================================

provider "azurerm" {
  features {}
  subscription_id = var.azure_subscription_id
}

provider "cyberarksia" {
  username = var.sia_username
  password = var.sia_password
}

# ============================================================================
# Random Suffix for Unique Naming
# ============================================================================

resource "random_string" "suffix" {
  length  = 6
  special = false
  upper   = false
}

# ============================================================================
# Azure Infrastructure
# ============================================================================

resource "azurerm_resource_group" "sia_test" {
  name     = var.resource_group_name
  location = var.azure_region

  # STATIC tags only - NO timestamp() to avoid unnecessary updates!
  tags = {
    environment = "test"
    purpose     = "sia-policy-validation"
    managed_by  = "terraform"
    created_on  = "2025-10-28"
  }
}

resource "azurerm_postgresql_flexible_server" "sia_test" {
  name                = "psql-sia-policy-${random_string.suffix.result}"
  resource_group_name = azurerm_resource_group.sia_test.name
  location            = azurerm_resource_group.sia_test.location

  # B1ms - cheapest option (~$0.017/hour)
  sku_name   = "B_Standard_B1ms"
  storage_mb = 32768 # 32 GB minimum
  version    = "16"

  administrator_login    = var.postgres_admin_username
  administrator_password = var.postgres_admin_password

  # Required for SIA connectivity
  public_network_access_enabled = true

  # Backup configuration
  backup_retention_days        = 7
  geo_redundant_backup_enabled = false

  # STATIC tags only - NO timestamp() to avoid unnecessary updates!
  tags = {
    environment = "test"
    purpose     = "sia-policy-validation"
    managed_by  = "terraform"
    created_on  = "2025-10-28"
  }

  lifecycle {
    ignore_changes = [zone]
  }
}

# Allow Azure services to connect
resource "azurerm_postgresql_flexible_server_firewall_rule" "allow_azure_services" {
  name             = "AllowAzureServices"
  server_id        = azurerm_postgresql_flexible_server.sia_test.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "0.0.0.0"
}

# Allow all IPs for testing (remove in production!)
resource "azurerm_postgresql_flexible_server_firewall_rule" "allow_all" {
  name             = "AllowAll"
  server_id        = azurerm_postgresql_flexible_server.sia_test.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "255.255.255.255"
}

# Create test database
resource "azurerm_postgresql_flexible_server_database" "testdb" {
  name      = "testdb"
  server_id = azurerm_postgresql_flexible_server.sia_test.id
  charset   = "UTF8"
  collation = "en_US.utf8"
}

# ============================================================================
# CyberArk SIA Resources
# ============================================================================

# Secret for PostgreSQL admin credentials
resource "cyberarksia_database_secret" "admin" {
  name                = "azure-postgres-admin-${random_string.suffix.result}"
  authentication_type = "local"
  username            = var.postgres_admin_username
  password            = var.postgres_admin_password

  # STATIC tags only - NO timestamp() to avoid unnecessary updates!
  tags = {
    environment = "test"
    managed_by  = "terraform"
    purpose     = "sia-policy-validation"
  }
}

# Database workspace for Azure PostgreSQL (NO certificate validation)
resource "cyberarksia_database_workspace" "azure_postgres" {
  name           = "azure-postgres-policy-test-${random_string.suffix.result}"
  database_type  = "postgres-azure-managed"
  cloud_provider = "azure"
  region         = var.azure_region

  # Connection details
  address = azurerm_postgresql_flexible_server.sia_test.fqdn
  port    = 5432

  # Credentials
  secret_id = cyberarksia_database_secret.admin.id

  # Certificate validation DISABLED as requested
  enable_certificate_validation = false
  # certificate_id is omitted (skipping certificate onboarding)

  # STATIC tags only - NO timestamp() to avoid unnecessary updates!
  tags = {
    environment = "test"
    managed_by  = "terraform"
    purpose     = "sia-policy-validation"
  }

  depends_on = [
    azurerm_postgresql_flexible_server_firewall_rule.allow_azure_services,
    azurerm_postgresql_flexible_server_firewall_rule.allow_all,
    azurerm_postgresql_flexible_server_database.testdb
  ]
}

# ============================================================================
# Database Access Policy (with inline assignments)
# ============================================================================

resource "cyberarksia_database_policy" "test" {
  # IMPORTANT: Using static name without timestamp() to avoid updates
  name                      = "SIA-Policy-Test-${random_string.suffix.result}"
  description               = "Database access policy for Azure PostgreSQL testing"
  status                    = "active"
  delegation_classification = "unrestricted"
  time_zone                 = "GMT"

  # Access conditions
  conditions {
    max_session_duration = 4  # 4 hours
    idle_time            = 10 # 10 minutes

    access_window {
      days_of_the_week = [1, 2, 3, 4, 5] # Monday-Friday
      from_hour        = "09:00"
      to_hour          = "17:00"
    }
  }

  # ============================================================================
  # INLINE TARGET DATABASE - Required (at least 1)
  # ============================================================================
  # Azure PostgreSQL database assigned inline

  target_database {
    database_workspace_id = cyberarksia_database_workspace.azure_postgres.id
    authentication_method = "db_auth"

    db_auth_profile {
      roles = ["pg_read_all_settings"]
    }
  }

  # ============================================================================
  # INLINE PRINCIPAL - Required (at least 1)
  # ============================================================================
  # Service account user assigned inline

  principal {
    principal_id          = var.service_account_principal_id
    principal_type        = "USER"
    principal_name        = var.sia_username
    source_directory_name = "CyberArk Cloud Directory"
    source_directory_id   = "09B9A9B0-6CE8-465F-AB03-65766D33B05E"
  }

  # STATIC tags only - NO timestamp() to avoid unnecessary updates!
  policy_tags = [
    "test:crud",
    "environment:test",
    "managed-by:terraform",
    "purpose:policy-validation"
  ]

  # Allow separate principal assignment resource to manage additional principals
  lifecycle {
    ignore_changes = [
      principal # Hybrid pattern: manage additional principals via assignment resources
    ]
  }

  depends_on = [cyberarksia_database_workspace.azure_postgres]
}

# ============================================================================
# Additional Principal Assignment (Tim Schindler)
# ============================================================================

resource "cyberarksia_database_policy_principal_assignment" "tim_schindler" {
  policy_id             = cyberarksia_database_policy.test.policy_id
  principal_id          = var.test_principal_id
  principal_type        = "USER" # CyberArk Cloud Directory users use USER, not ROLE
  principal_name        = var.test_principal_email
  source_directory_name = "CyberArk Cloud Directory"
  source_directory_id   = var.cyberark_cloud_directory_id

  depends_on = [cyberarksia_database_policy.test]
}
