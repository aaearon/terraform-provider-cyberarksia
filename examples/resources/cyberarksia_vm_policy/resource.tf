# Basic VM Access Policy with FQDN/IP targets and SSH behavior
resource "cyberarksia_vm_policy" "example" {
  name          = "production-servers-policy"
  description   = "Access policy for production web servers"
  status        = "Active"
  location_type = "FQDN/IP"

  # At least one principal required
  principals {
    principal_id            = "user-uuid-here"
    principal_name          = "admin@example.com"
    principal_type          = "USER"
    source_directory_name   = "CyberArk"
    source_directory_id     = "directory-uuid-here"
  }

  # SSH connection profile (required)
  behavior {
    ssh {
      username = "ec2-user"
    }
  }

  # FQDN/IP targets
  fqdn_ip_targets {
    fqdn_rule {
      operator            = "SUFFIX"
      computername_pattern = "-prod"
      domain              = "example.com"
    }
  }

  # Access conditions
  max_session_duration = 4  # 4 hours
  idle_time           = 30 # 30 minutes
}

# Add additional principal via assignment
resource "cyberarksia_vm_policy_principal_assignment" "example" {
  policy_id               = cyberarksia_vm_policy.example.policy_id
  principal_id            = "user2-uuid-here"
  principal_name          = "developer@example.com"
  principal_type          = "USER"
  source_directory_name   = "CyberArk"
  source_directory_id     = "directory-uuid-here"
}
