# AWS cloud VM access policy example
# Demonstrates AWS-specific target criteria: regions, tags, VPC IDs, account IDs

data "cyberarksia_principal" "cloud_admin" {
  name = "cloud-admin@example.com"
  type = "USER"
}

resource "cyberarksia_vm_policy" "aws_production" {
  name          = "AWS-Production-VMs"
  location_type = "AWS"
  status        = "Active"
  description   = "Access policy for AWS production EC2 instances"

  # Initial principal assignment
  principal {
    principal_id          = data.cyberarksia_principal.cloud_admin.principal_id
    principal_name        = data.cyberarksia_principal.cloud_admin.principal_name
    principal_type        = data.cyberarksia_principal.cloud_admin.principal_type
    source_directory_name = data.cyberarksia_principal.cloud_admin.source_directory_name
    source_directory_id   = data.cyberarksia_principal.cloud_admin.source_directory_id
  }

  # SSH connection behavior
  behavior {
    ssh {
      username = "ec2-user"
    }
  }

  # AWS target criteria
  aws_targets {
    regions = ["us-east-1", "us-west-2"]

    # Resource tags
    tags {
      key   = "Environment"
      value = ["production"]
    }

    tags {
      key   = "Team"
      value = ["platform", "infrastructure"]
    }

    vpc_ids     = ["vpc-12345678", "vpc-abcdef12"]
    account_ids = ["123456789012"]
  }

  # Access conditions
  max_session_duration = 2  # 2 hours
  idle_time            = 15 # 15 minutes

  # 24/7 access for cloud operations
  time_zone = "UTC"

  tags = ["aws", "production", "cloud"]
}

output "aws_policy_id" {
  value       = cyberarksia_vm_policy.aws_production.policy_id
  description = "AWS VM policy ID"
}
