# Lookup policy by ID
data "cyberarksia_database_policy" "by_id" {
  policy_id = "12345678-1234-1234-1234-123456789012"
}

# Lookup policy by name (more user-friendly)
data "cyberarksia_database_policy" "db_admins" {
  name = "Database Administrators Policy"
}

# Use policy in database assignment resource
resource "cyberarksia_database_policy_workspace_assignment" "prod_postgres" {
  policy_id             = data.cyberarksia_database_policy.db_admins.id
  database_workspace_id = cyberarksia_database_workspace.prod.id
  authentication_method = "db_auth"

  db_auth_profile {
    roles = ["db_reader", "db_writer"]
  }
}

# Output policy information
output "policy_details" {
  value = {
    id          = data.cyberarksia_database_policy.db_admins.id
    name        = data.cyberarksia_database_policy.db_admins.name
    description = data.cyberarksia_database_policy.db_admins.description
    status      = data.cyberarksia_database_policy.db_admins.status
  }
}
