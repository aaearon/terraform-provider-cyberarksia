# Provider configuration for multi-cloud examples
#
# PREREQUISITES:
# 1. CyberArk Identity tenant with SIA enabled
# 2. Service account with VM policy management permissions
# 3. At least one user/group configured in your identity directory

terraform {
  required_providers {
    cyberarksia = {
      source  = "aaearon/cyberarksia"
      version = "~> 0.2"
    }
  }
}

provider "cyberarksia" {
  # Credentials from environment variables:
  # export CYBERARK_USERNAME="your-service-account@cyberark.cloud.12345"
  # export CYBERARK_PASSWORD="your-password"
}

# =============================================================================
# VARIABLES - Customize these for your environment
# =============================================================================

variable "principal_name" {
  description = "Name of user or group to grant access (e.g., 'admin@example.com' or 'cloud-admins')"
  type        = string
}

variable "principal_type" {
  description = "Type of principal: USER, GROUP, or ROLE"
  type        = string
  default     = "USER"

  validation {
    condition     = contains(["USER", "GROUP", "ROLE"], var.principal_type)
    error_message = "principal_type must be USER, GROUP, or ROLE"
  }
}

# =============================================================================
# PRINCIPAL LOOKUP
# =============================================================================

# Look up the principal (user/group/role) that will be granted access
# This must exist in your CyberArk Identity directory
data "cyberarksia_principal" "target_principal" {
  name = var.principal_name
  type = var.principal_type
}

# =============================================================================
# OUTPUTS
# =============================================================================

output "principal_id" {
  description = "ID of the looked-up principal"
  value       = data.cyberarksia_principal.target_principal.id
}
