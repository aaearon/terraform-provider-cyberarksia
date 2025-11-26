# GCP VM Access Policy Example
#
# This example demonstrates VM access policies targeting GCP infrastructure.
#
# PREREQUISITES:
# - GCP project ID (not project number)
# - (Optional) VPC network paths for more granular targeting
#
# USAGE:
#   terraform apply -var="principal_name=admin@example.com" -var="gcp_project_id=my-project-123"

# =============================================================================
# VARIABLES
# =============================================================================

variable "gcp_project_id" {
  description = "GCP project ID (not project number)"
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.gcp_project_id))
    error_message = "GCP project ID must be 6-30 lowercase letters, digits, or hyphens, starting with a letter"
  }
}

variable "gcp_vpc_network_names" {
  description = "Optional list of VPC network names (will be converted to full paths)"
  type        = list(string)
  default     = []
}

# =============================================================================
# LOCAL VALUES
# =============================================================================

locals {
  # Convert VPC network names to full resource paths
  gcp_vpc_network_paths = [
    for vpc in var.gcp_vpc_network_names :
    "projects/${var.gcp_project_id}/global/networks/${vpc}"
  ]
}

# =============================================================================
# GCP VM POLICY
# =============================================================================

resource "cyberarksia_vm_policy" "gcp_ssh_access" {
  name          = "gcp-ssh-access"
  description   = "SSH access to GCP Compute Engine instances"
  status        = "active"
  location_type = "GCP"
  protocols     = ["SSH"]

  # Target GCP resources
  gcp_targets {
    project_ids     = [var.gcp_project_id]
    vpc_network_ids = local.gcp_vpc_network_paths
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

output "gcp_policy_id" {
  description = "ID of the created GCP VM policy"
  value       = cyberarksia_vm_policy.gcp_ssh_access.id
}

output "gcp_vpc_network_paths" {
  description = "Full VPC network paths used for targeting"
  value       = local.gcp_vpc_network_paths
}
