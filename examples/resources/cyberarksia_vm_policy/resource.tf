# Basic FQDN/IP VM access policy with SSH connection behavior
# This example demonstrates a minimal VM policy for on-premises server access

# Data source to lookup principals by name (avoids manual UUID lookup)
data "cyberarksia_principal" "vm_admin" {
  name = "admin@example.com"
  type = "USER"
}

resource "cyberarksia_vm_policy" "basic_servers" {
  name          = "Production-Servers-Access"
  location_type = "FQDN/IP"
  status        = "Active"
  description   = "Access policy for production on-premises servers"

  # Required: At least one principal assignment at policy creation
  principal {
    principal_id          = data.cyberarksia_principal.vm_admin.principal_id
    principal_name        = data.cyberarksia_principal.vm_admin.principal_name
    principal_type        = data.cyberarksia_principal.vm_admin.principal_type
    source_directory_name = data.cyberarksia_principal.vm_admin.source_directory_name
    source_directory_id   = data.cyberarksia_principal.vm_admin.source_directory_id
  }

  # Connection behavior: SSH with specific username
  behavior {
    ssh {
      username = "ec2-user"
    }
  }

  # Target servers: FQDN suffix match
  fqdn_ip_targets {
    fqdn_rule {
      operator             = "SUFFIX"
      computername_pattern = "-prod"
      domain               = "example.com"
    }
  }

  # Access conditions
  max_session_duration = 4  # 4 hours
  idle_time            = 10 # 10 minutes

  # Business hours access window
  access_window {
    days_of_the_week = [1, 2, 3, 4, 5] # Monday-Friday
    from_hour        = "09:00"
    to_hour          = "17:00"
  }

  # Policy activation period
  time_frame {
    from_time = "2025-01-01T00:00:00"
    to_time   = "2025-12-31T23:59:59"
  }

  tags = ["production", "on-premises", "ssh-access"]
}

# Output the policy ID for use in principal assignments
output "policy_id" {
  value       = cyberarksia_vm_policy.basic_servers.policy_id
  description = "The ID of the created VM policy"
}

output "delegation_classification" {
  value       = cyberarksia_vm_policy.basic_servers.delegation_classification
  description = "Server-computed delegation classification"
}
