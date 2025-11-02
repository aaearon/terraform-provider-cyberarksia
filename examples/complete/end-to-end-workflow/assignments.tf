# NOTE: This file demonstrates the MODULAR assignment pattern
#
# The policies.tf file uses INLINE assignments (principals and target_database blocks)
# This file shows how to manage assignments SEPARATELY using dedicated resources
#
# Use INLINE when: You want to manage the entire policy configuration in one place
# Use MODULAR when: Different teams manage principals (security) vs databases (app teams)

# Example: Adding another developer group to existing policy (modular pattern)
data "cyberarksia_principal" "senior_developers" {
  name = "senior-developers@example.com"
  type = "GROUP"
}

resource "cyberarksia_database_policy_principal_assignment" "senior_devs" {
  policy_id             = cyberarksia_database_policy.developer_access.policy_id
  principal_id          = data.cyberarksia_principal.senior_developers.principal_id
  principal_name        = data.cyberarksia_principal.senior_developers.principal_name
  principal_type        = data.cyberarksia_principal.senior_developers.principal_type
  source_directory_id   = data.cyberarksia_principal.senior_developers.source_directory_id
  source_directory_name = data.cyberarksia_principal.senior_developers.source_directory_name
}

# Example: Adding another database to existing policy (modular pattern)
resource "cyberarksia_database_workspace" "staging_postgres" {
  name           = "staging-customers"
  database_type  = "postgres"
  address        = "staging-db.internal.example.com"
  port           = 5432
  secret_id      = cyberarksia_database_secret.postgres_admin.id
  cloud_provider = "on_premise"

  tags = {
    environment = "staging"
    application = "customer-management"
  }
}

resource "cyberarksia_database_policy_database_assignment" "staging_dev_access" {
  policy_id             = cyberarksia_database_policy.developer_access.policy_id
  database_workspace_id = cyberarksia_database_workspace.staging_postgres.id

  authentication_method = "db_auth"

  db_auth_profile {
    roles = ["staging_developer"]
  }
}
