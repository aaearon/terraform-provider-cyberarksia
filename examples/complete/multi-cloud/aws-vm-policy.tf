# AWS VM Access Policy Example
#
# This example demonstrates VM access policies targeting AWS infrastructure.
#
# PREREQUISITES:
# - AWS account IDs you want to target
# - (Optional) VPC IDs for more granular targeting
#
# USAGE:
#   terraform apply -var="principal_name=admin@example.com" -var="aws_account_id=123456789012"

# =============================================================================
# VARIABLES
# =============================================================================

variable "aws_account_id" {
  description = "AWS account ID (12 digits) to target"
  type        = string

  validation {
    condition     = can(regex("^[0-9]{12}$", var.aws_account_id))
    error_message = "AWS account ID must be exactly 12 digits"
  }
}

variable "aws_vpc_ids" {
  description = "Optional list of VPC IDs to target (leave empty for all VPCs in account)"
  type        = list(string)
  default     = []
}

# =============================================================================
# AWS VM POLICY
# =============================================================================

resource "cyberarksia_vm_policy" "aws_ssh_access" {
  name          = "aws-ssh-access"
  description   = "SSH access to AWS EC2 instances"
  status        = "active"
  location_type = "AWS"
  protocols     = ["SSH"]

  # Target AWS resources
  aws_targets {
    account_ids = [var.aws_account_id]
    vpc_ids     = var.aws_vpc_ids
  }

  # At least one principal is required at creation
  principals {
    id   = data.cyberarksia_principal.target_principal.id
    type = var.principal_type
  }

  session_timeout = 60 # minutes
}

# =============================================================================
# OUTPUTS
# =============================================================================

output "aws_policy_id" {
  description = "ID of the created AWS VM policy"
  value       = cyberarksia_vm_policy.aws_ssh_access.id
}

output "aws_policy_name" {
  description = "Name of the created AWS VM policy"
  value       = cyberarksia_vm_policy.aws_ssh_access.name
}
