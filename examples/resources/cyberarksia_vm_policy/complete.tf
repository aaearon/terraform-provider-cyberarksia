# Complete VM Policy Example
# Demonstrates all features: AWS + RDP + time windows + multiple principals

data "cyberarksia_principal" "admin" {
  name = "cloud-admin@example.com"
  type = "USER"
}

data "cyberarksia_principal" "ops_team" {
  name = "Operations-Team"
  type = "GROUP"
}

resource "cyberarksia_vm_policy" "comprehensive" {
  name          = "Comprehensive-AWS-Policy"
  location_type = "AWS"
  status        = "Active"
  description   = "Full-featured VM policy with all configuration options"
  policy_type   = "Recurring"
  time_zone     = "America/New_York"

  # Multiple initial principals (minimum 1 required)
  principals {
    principal_id          = data.cyberarksia_principal.admin.id
    principal_name        = data.cyberarksia_principal.admin.name
    principal_type        = data.cyberarksia_principal.admin.principal_type
    source_directory_name = data.cyberarksia_principal.admin.directory_name
    source_directory_id   = data.cyberarksia_principal.admin.directory_id
  }

  principals {
    principal_id          = data.cyberarksia_principal.ops_team.id
    principal_name        = data.cyberarksia_principal.ops_team.name
    principal_type        = data.cyberarksia_principal.ops_team.principal_type
    source_directory_name = data.cyberarksia_principal.ops_team.directory_name
    source_directory_id   = data.cyberarksia_principal.ops_team.directory_id
  }

  # Both SSH and RDP (supported simultaneously)
  behavior {
    ssh {
      username = "ec2-user"
    }

    rdp {
      local_ephemeral_user {
        assign_groups                   = ["Administrators", "Remote Desktop Users"]
        enable_ephemeral_user_reconnect = true
      }
    }
  }

  # AWS target criteria with multiple filters
  aws_targets {
    regions = ["us-east-1", "us-west-2", "eu-west-1"]

    # Multiple tag filters
    tags {
      key   = "Environment"
      value = ["production", "staging"]
    }

    tags {
      key   = "ManagedBy"
      value = ["Terraform"]
    }

    tags {
      key   = "Compliance"
      value = ["PCI-DSS", "SOC2"]
    }

    # VPC IDs: vpc- followed by 8 or 17 alphanumeric chars
    vpc_ids = [
      "vpc-0a1b2c3d4e5f6789a",
      "vpc-1b2c3d4e5f6789a0b",
      "vpc-2c3d4e5f6789a0b1c"
    ]

    account_ids = ["123456789012", "210987654321"]
  }

  # Session duration and idle timeout
  max_session_duration = 4  # 4 hours
  idle_time            = 15 # 15 minutes

  # Access window: Monday-Friday, business hours
  access_window {
    days_of_the_week = [1, 2, 3, 4, 5]
    from_hour        = "08:00"
    to_hour          = "18:00"
  }

  # Policy activation period (1 year)
  time_frame {
    from_time = "2025-01-01T00:00:00"
    to_time   = "2025-12-31T23:59:59"
  }

  # Comprehensive tagging
  tags = [
    "aws",
    "production",
    "compliance",
    "multi-region",
    "ssh",
    "rdp"
  ]
}

# Outputs
output "comprehensive_policy_id" {
  value       = cyberarksia_vm_policy.comprehensive.policy_id
  description = "Policy ID for reference in assignments"
}

output "comprehensive_delegation" {
  value       = cyberarksia_vm_policy.comprehensive.delegation_classification
  description = "Server-computed delegation classification"
}

output "comprehensive_created_by" {
  value = {
    user      = cyberarksia_vm_policy.comprehensive.created_by.user
    timestamp = cyberarksia_vm_policy.comprehensive.created_by.timestamp
  }
  description = "Policy creation metadata"
}

# Additional principals can be added via assignment resource
resource "cyberarksia_vm_policy_principal_assignment" "additional_admin" {
  policy_id             = cyberarksia_vm_policy.comprehensive.policy_id
  principal_id          = "another-admin-id"
  principal_name        = "backup-admin@example.com"
  principal_type        = "USER"
  source_directory_name = "CyberArk"
  source_directory_id   = "dir-456"
}
