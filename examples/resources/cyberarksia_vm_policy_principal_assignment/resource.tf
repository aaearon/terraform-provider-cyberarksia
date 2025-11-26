# VM Policy Principal Assignment Example
# Demonstrates adding additional principals to an existing VM policy

# Reference to existing VM policy (created elsewhere)
resource "cyberarksia_vm_policy" "shared_servers" {
  name          = "Shared-Development-Servers"
  location_type = "FQDN/IP"
  status        = "Active"

  # Required initial principal
  principal {
    principal_id          = "initial-user-id"
    principal_name        = "dev-lead@example.com"
    principal_type        = "USER"
    source_directory_name = "CyberArk"
    source_directory_id   = "dir-123"
  }

  behavior {
    ssh {
      username = "developer"
    }
  }

  fqdn_ip_targets {
    fqdn_rule {
      operator             = "CONTAINS"
      computername_pattern = "dev"
    }
  }

  max_session_duration = 8
}

# Add individual user
data "cyberarksia_principal" "jane" {
  name = "jane.doe@example.com"
  type = "USER"
}

resource "cyberarksia_vm_policy_principal_assignment" "jane_access" {
  policy_id             = cyberarksia_vm_policy.shared_servers.policy_id
  principal_id          = data.cyberarksia_principal.jane.principal_id
  principal_name        = data.cyberarksia_principal.jane.principal_name
  principal_type        = data.cyberarksia_principal.jane.principal_type
  source_directory_name = data.cyberarksia_principal.jane.source_directory_name
  source_directory_id   = data.cyberarksia_principal.jane.source_directory_id
}

# Add group
data "cyberarksia_principal" "dev_team" {
  name = "Development-Team"
  type = "GROUP"
}

resource "cyberarksia_vm_policy_principal_assignment" "dev_team_access" {
  policy_id             = cyberarksia_vm_policy.shared_servers.policy_id
  principal_id          = data.cyberarksia_principal.dev_team.principal_id
  principal_name        = data.cyberarksia_principal.dev_team.principal_name
  principal_type        = data.cyberarksia_principal.dev_team.principal_type
  source_directory_name = data.cyberarksia_principal.dev_team.source_directory_name
  source_directory_id   = data.cyberarksia_principal.dev_team.source_directory_id
}

# Add role (no source directory required for roles)
data "cyberarksia_principal" "ops_role" {
  name = "Operations-Role"
  type = "ROLE"
}

resource "cyberarksia_vm_policy_principal_assignment" "ops_role_access" {
  policy_id      = cyberarksia_vm_policy.shared_servers.policy_id
  principal_id   = data.cyberarksia_principal.ops_role.principal_id
  principal_name = data.cyberarksia_principal.ops_role.principal_name
  principal_type = data.cyberarksia_principal.ops_role.principal_type
}

# Bulk assignment using for_each
variable "additional_users" {
  type = map(object({
    name                  = string
    type                  = string
    source_directory_name = optional(string)
    source_directory_id   = optional(string)
  }))
  default = {
    bob = {
      name                  = "bob.smith@example.com"
      type                  = "USER"
      source_directory_name = "CyberArk"
      source_directory_id   = "dir-123"
    }
    alice = {
      name                  = "alice.jones@example.com"
      type                  = "USER"
      source_directory_name = "CyberArk"
      source_directory_id   = "dir-123"
    }
  }
}

data "cyberarksia_principal" "bulk_users" {
  for_each = var.additional_users
  name     = each.value.name
  type     = each.value.type
}

resource "cyberarksia_vm_policy_principal_assignment" "bulk_assignments" {
  for_each = data.cyberarksia_principal.bulk_users

  policy_id             = cyberarksia_vm_policy.shared_servers.policy_id
  principal_id          = each.value.principal_id
  principal_name        = each.value.principal_name
  principal_type        = each.value.principal_type
  source_directory_name = each.value.source_directory_name
  source_directory_id   = each.value.source_directory_id
}

# Output the composite IDs for import reference
output "jane_assignment_id" {
  value       = cyberarksia_vm_policy_principal_assignment.jane_access.id
  description = "Composite ID format: policy-id:principal-id:principal-type"
}

output "all_assignment_ids" {
  value = {
    jane     = cyberarksia_vm_policy_principal_assignment.jane_access.id
    dev_team = cyberarksia_vm_policy_principal_assignment.dev_team_access.id
    ops_role = cyberarksia_vm_policy_principal_assignment.ops_role_access.id
  }
  description = "All principal assignment composite IDs"
}
