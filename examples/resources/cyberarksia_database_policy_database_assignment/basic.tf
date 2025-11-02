# Basic example: Add a database to a policy with database authentication

# Lookup existing policy by name
data "cyberarksia_database_policy" "db_admins" {
  name = "Database Administrators Policy"
}

# Create database workspace
resource "cyberarksia_database_workspace" "prod_postgres" {
  name          = "prod-postgres"
  database_type = "postgres"
  address       = "postgres.example.com"
  port          = 5432
  secret_id     = cyberarksia_database_secret.db_admin.id
}

# Create secret for database
resource "cyberarksia_database_secret" "db_admin" {
  name                = "postgres-admin"
  authentication_type = "local"
  username            = "admin"
  password            = var.db_password
}

# Add database to policy
resource "cyberarksia_database_policy_database_assignment" "prod_postgres_to_policy" {
  policy_id             = data.cyberarksia_database_policy.db_admins.id
  database_workspace_id = cyberarksia_database_workspace.prod_postgres.id
  authentication_method = "db_auth"

  db_auth_profile {
    roles = ["db_reader", "db_writer"]
  }
}

# Multiple databases can be added to the same policy
resource "cyberarksia_database_policy_database_assignment" "prod_mysql_to_policy" {
  policy_id             = data.cyberarksia_database_policy.db_admins.id
  database_workspace_id = cyberarksia_database_workspace.prod_mysql.id
  authentication_method = "db_auth"

  db_auth_profile {
    roles = ["SELECT", "INSERT", "UPDATE"]
  }
}

variable "db_password" {
  description = "Database administrator password"
  type        = string
  sensitive   = true
}
