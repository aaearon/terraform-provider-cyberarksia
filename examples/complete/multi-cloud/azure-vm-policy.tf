# Azure VM Access Policy Example
#
# This example demonstrates VM access policies targeting Azure infrastructure.
#
# IMPORTANT: Azure targets require FULL ARM resource paths, not just names.
# Common error: Using "my-rg" instead of "/subscriptions/.../resourceGroups/my-rg"
#
# PREREQUISITES:
# - Azure subscription ID (UUID format)
# - Resource group ARM paths (if targeting specific RGs)
#
# USAGE:
#   terraform apply \
#     -var="principal_name=admin@example.com" \
#     -var="azure_subscription_id=759a039e-dc44-4762-9f40-2696323c2fa5"

# =============================================================================
# VARIABLES
# =============================================================================

variable "azure_subscription_id" {
  description = "Azure subscription ID (UUID format)"
  type        = string

  validation {
    condition     = can(regex("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", var.azure_subscription_id))
    error_message = "Azure subscription ID must be a valid UUID"
  }
}

variable "azure_resource_group_names" {
  description = "Optional list of resource group NAMES (will be converted to ARM paths)"
  type        = list(string)
  default     = []
}

# =============================================================================
# LOCAL VALUES
# =============================================================================

locals {
  # Convert resource group names to full ARM paths
  azure_resource_group_paths = [
    for rg in var.azure_resource_group_names :
    "/subscriptions/${var.azure_subscription_id}/resourceGroups/${rg}"
  ]
}

# =============================================================================
# AZURE VM POLICY
# =============================================================================

resource "cyberarksia_vm_policy" "azure_rdp_access" {
  name          = "azure-rdp-access"
  description   = "RDP access to Azure VMs"
  status        = "active"
  location_type = "Azure"
  protocols     = ["RDP"]

  # Target Azure resources
  azure_targets {
    subscription_ids = [var.azure_subscription_id]
    resource_groups  = local.azure_resource_group_paths
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

output "azure_policy_id" {
  description = "ID of the created Azure VM policy"
  value       = cyberarksia_vm_policy.azure_rdp_access.id
}

output "azure_resource_group_paths" {
  description = "Full ARM paths used for resource group targeting"
  value       = local.azure_resource_group_paths
}
