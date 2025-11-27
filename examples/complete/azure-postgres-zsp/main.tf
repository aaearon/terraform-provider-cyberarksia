terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
    cyberarksia = {
      source  = "aaearon/cyberarksia"
      version = "~> 0.2"
    }
  }
}

# ============================================================================
# PROVIDERS
# ============================================================================

provider "azurerm" {
  features {}
  subscription_id = var.azure_subscription_id
}

provider "cyberarksia" {
  username = var.cyberark_username
  password = var.cyberark_password
}

# ============================================================================
# VARIABLES
# ============================================================================

variable "azure_subscription_id" {
  description = "Azure subscription ID"
  type        = string
}

variable "cyberark_username" {
  description = "CyberArk service account username"
  type        = string
  sensitive   = true
}

variable "cyberark_password" {
  description = "CyberArk service account password"
  type        = string
  sensitive   = true
}

variable "postgres_admin_password" {
  description = "PostgreSQL administrator password"
  type        = string
  sensitive   = true
}

variable "location" {
  description = "Azure region"
  type        = string
  default     = "eastus"
}

variable "resource_prefix" {
  description = "Prefix for resource names"
  type        = string
  default     = "sia-demo"
}

# ============================================================================
# AZURE RESOURCES
# ============================================================================

resource "azurerm_resource_group" "main" {
  name     = "${var.resource_prefix}-rg"
  location = var.location

  tags = {
    purpose    = "sia-provider-validation"
    managed_by = "terraform"
  }
}

# Cheapest PostgreSQL Flexible Server configuration
resource "azurerm_postgresql_flexible_server" "main" {
  name                = "${var.resource_prefix}-postgres"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location

  # Admin credentials (used by SIA for ephemeral user provisioning)
  administrator_login    = "siaadmin"
  administrator_password = var.postgres_admin_password

  # Cheapest configuration
  sku_name     = "B_Standard_B1ms"
  version      = "16"
  storage_mb   = 32768
  storage_tier = "P4"

  # Public access for quick validation
  public_network_access_enabled = true

  tags = {
    purpose = "sia-provider-validation"
  }
}

# Allow all IPs (for quick validation only - NOT for production)
resource "azurerm_postgresql_flexible_server_firewall_rule" "allow_all" {
  name             = "allow-all"
  server_id        = azurerm_postgresql_flexible_server.main.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "255.255.255.255"
}

# ============================================================================
# CYBERARK SIA RESOURCES
# ============================================================================

# 1. Database Secret - credentials for ephemeral user provisioning
resource "cyberarksia_database_secret" "postgres_admin" {
  name                = "${var.resource_prefix}-postgres-admin"
  authentication_type = "local"
  username            = "siaadmin"
  password            = var.postgres_admin_password
}

# 2. Database Workspace - register the PostgreSQL database with SIA
resource "cyberarksia_database_workspace" "postgres" {
  name                  = "${var.resource_prefix}-postgres"
  database_type         = "postgres-azure-managed"
  address               = azurerm_postgresql_flexible_server.main.fqdn
  port                  = 5432
  secret_id             = cyberarksia_database_secret.postgres_admin.id
  cloud_provider        = "azure"
  authentication_method = "local_ephemeral_user"

  tags = {
    environment = "demo"
    cloud       = "azure"
  }

  # Ensure firewall is open before SIA attempts to validate connectivity
  depends_on = [azurerm_postgresql_flexible_server_firewall_rule.allow_all]
}

# 3. Look up principals by name (no manual UUID lookup needed)
data "cyberarksia_principal" "user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

data "cyberarksia_principal" "role" {
  name = "CyberIAM Guardians"
  type = "ROLE"
}

# 4. Database Policy - Zero Standing Privilege with 24/7 access
resource "cyberarksia_database_policy" "zsp" {
  name   = "${var.resource_prefix}-zsp-policy"
  status = "active"

  # Inline principal (at least one required at creation)
  principal {
    principal_id          = data.cyberarksia_principal.user.id
    principal_type        = "USER"
    principal_name        = data.cyberarksia_principal.user.display_name
    source_directory_name = data.cyberarksia_principal.user.directory_name
    source_directory_id   = data.cyberarksia_principal.user.directory_id
  }

  # Target database
  target_database {
    database_workspace_id = cyberarksia_database_workspace.postgres.id
    authentication_method = "db_auth"

    db_auth_profile {
      roles = ["pg_read_all_data", "pg_write_all_data"]
    }
  }

  # 24/7 access with reasonable session limits
  conditions {
    max_session_duration = 8
    idle_time            = 30
  }
}

# 5. Add the role as a second principal
resource "cyberarksia_database_policy_principal_assignment" "role" {
  policy_id      = cyberarksia_database_policy.zsp.policy_id
  principal_id   = data.cyberarksia_principal.role.id
  principal_type = "ROLE"
  principal_name = data.cyberarksia_principal.role.display_name
}

# ============================================================================
# OUTPUTS
# ============================================================================

output "postgres_fqdn" {
  description = "PostgreSQL server FQDN"
  value       = azurerm_postgresql_flexible_server.main.fqdn
}

output "postgres_admin_login" {
  description = "PostgreSQL admin username"
  value       = azurerm_postgresql_flexible_server.main.administrator_login
}

output "sia_workspace_id" {
  description = "SIA Database Workspace ID"
  value       = cyberarksia_database_workspace.postgres.id
}

output "sia_policy_id" {
  description = "SIA Policy ID"
  value       = cyberarksia_database_policy.zsp.policy_id
}

output "assigned_principals" {
  description = "Principals with access"
  value = [
    "${data.cyberarksia_principal.user.name} (USER)",
    "${data.cyberarksia_principal.role.name} (ROLE)"
  ]
}
