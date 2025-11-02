# Example: Strong Account with Local Authentication
# This example demonstrates creating a strong account using local (username/password) authentication
# for a PostgreSQL database target.

# Provider configuration
terraform {
  required_providers {
    cyberarksia = {
      source  = "aaearon/cyberarksia"
      version = ">= 0.1.0"
    }
  }
}

provider "cyberarksia" {
  # Authentication credentials should be set via environment variables:
  # export CYBERARK_USERNAME="service-account@cyberark.cloud.XXXXX"
  # export CYBERARK_CLIENT_SECRET="your-client-secret"
  # Optional (for GovCloud or custom deployments):
  # export CYBERARK_IDENTITY_URL="https://your-tenant.cyberarkgov.cloud"
}

# On-premise database target
resource "cyberarksia_database_workspace" "postgres" {
  name                  = "production-postgres"
  database_type         = "postgresql"
  address               = "prod-db.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  description           = "Production PostgreSQL database"

  tags = {
    environment = "production"
    team        = "platform"
  }
}

# Strong account with local authentication
# NOTE: Secrets are standalone resources - no database_workspace_id required
resource "cyberarksia_database_secret" "postgres_admin" {
  name                = "postgres-admin-account"
  authentication_type = "local"

  # Local authentication credentials
  username = "sia_admin"
  password = var.postgres_admin_password # Sensitive - should be stored in variable or secret manager
}

# Variables for sensitive data
variable "postgres_admin_password" {
  description = "Password for the PostgreSQL admin account"
  type        = string
  sensitive   = true
}

# Outputs (safe - does not expose sensitive data)
output "strong_account_id" {
  description = "ID of the created strong account"
  value       = cyberarksia_database_secret.postgres_admin.id
}

output "strong_account_name" {
  description = "Name of the strong account"
  value       = cyberarksia_database_secret.postgres_admin.name
}

output "created_at" {
  description = "Timestamp when the strong account was created"
  value       = cyberarksia_database_secret.postgres_admin.created_at
}
