# On-Premises (FQDN/IP) VM Access Policy Example
#
# This example demonstrates VM access policies targeting on-premises servers
# or any infrastructure accessible via FQDN or IP address.
#
# SUPPORTED FORMATS:
# - Specific hostnames: "server1.example.com"
# - Wildcards: "*.prod.example.com"
# - IP addresses: "192.168.1.100"
# - CIDR ranges: "10.0.0.0/24"
#
# USAGE:
#   terraform apply \
#     -var="principal_name=admin@example.com" \
#     -var='fqdn_targets=["*.prod.example.com", "bastion.example.com"]'

# =============================================================================
# VARIABLES
# =============================================================================

variable "fqdn_targets" {
  description = "List of FQDNs, hostnames, IPs, or CIDR ranges to target"
  type        = list(string)

  validation {
    condition     = length(var.fqdn_targets) > 0
    error_message = "At least one FQDN target is required"
  }
}

variable "onprem_protocols" {
  description = "Protocols to allow: SSH, RDP, or both"
  type        = list(string)
  default     = ["SSH"]

  validation {
    condition     = alltrue([for p in var.onprem_protocols : contains(["SSH", "RDP"], p)])
    error_message = "Protocols must be SSH or RDP"
  }
}

# =============================================================================
# ON-PREMISES VM POLICY
# =============================================================================

resource "cyberarksia_vm_policy" "onprem_access" {
  name          = "onprem-server-access"
  description   = "Access to on-premises servers via FQDN/IP"
  status        = "active"
  location_type = "FQDN/IP"
  protocols     = var.onprem_protocols

  # Target specific FQDNs, hostnames, or IP ranges
  fqdn_targets = var.fqdn_targets

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

output "onprem_policy_id" {
  description = "ID of the created on-premises VM policy"
  value       = cyberarksia_vm_policy.onprem_access.id
}

output "onprem_targets" {
  description = "FQDN/IP targets configured for this policy"
  value       = var.fqdn_targets
}
