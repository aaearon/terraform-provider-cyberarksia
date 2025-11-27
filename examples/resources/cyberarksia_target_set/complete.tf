# Complete Example: Target Set with All Attributes
#
# This example demonstrates all available attributes for cyberarksia_target_set

# Prerequisite: VM Secret
resource "cyberarksia_vm_secret" "complete_example" {
  secret_name          = "complete-target-set-credentials"
  secret_type          = "ProvisionerUser"
  provisioner_username = "Administrator"
  provisioner_password = var.admin_password # Sensitive variable
}

# Target Set with All Attributes
resource "cyberarksia_target_set" "complete" {
  # ===== Required Attributes =====

  # Name: Unique identifier for the target set
  # Used in API URLs and as the resource ID
  # WARNING: Avoid forward slashes - they cause deletion issues (403 errors)
  name = "complete-example.example.com"

  # Type: Matching pattern for server selection
  # Options: "Domain", "Suffix", "Target"
  # - Domain: Matches all servers in *.complete-example.example.com
  # - Suffix: Matches servers ending with complete-example.example.com
  # - Target: Matches exact hostname complete-example.example.com
  type = "Domain"

  # Secret ID: Reference to VM credentials
  # Best practice: Use resource reference for automatic dependency ordering
  secret_id = cyberarksia_vm_secret.complete_example.id

  # Secret Type: Must match the referenced secret's type
  # Options: "ProvisionerUser", "PCloudAccount"
  secret_type = cyberarksia_vm_secret.complete_example.secret_type

  # ===== Optional Attributes =====

  # Provision Format: Template for ephemeral account names during JIT sessions
  # Placeholders:
  #   <user> - Replaced with requesting user's username (e.g., "john.doe")
  #   <session-guid> - Replaced with unique session ID (e.g., "abc123-def456")
  #
  # Default: "<user>-<session-guid>"
  #
  # Examples:
  #   "jit-<user>-<session-guid>" → "jit-john.doe-abc123-def456"
  #   "<user>-prod-<session-guid>" → "john.doe-prod-abc123-def456"
  #
  # IMPORTANT: Cannot be removed once set (maintains audit trail consistency)
  # You can update it to a different value, but cannot clear it entirely
  provision_format = "jit-<user>-<session-guid>"

  # Description: Human-readable description (not displayed in SIA UI)
  # Useful for documentation and Terraform configuration clarity
  description = "Complete example showing all target set attributes"

  # Enable Certificate Validation: TLS/SSL certificate validation for target connections
  # Default: true (secure default)
  # Set to false for development/testing with self-signed certificates
  enable_certificate_validation = true
}

# ============================================================================
# Outputs: Demonstrate all available attributes
# ============================================================================

output "target_set_complete_example" {
  value = {
    # Computed Attributes
    id = cyberarksia_target_set.complete.id # Equals name (name-as-ID pattern)

    # Required Attributes
    name        = cyberarksia_target_set.complete.name
    type        = cyberarksia_target_set.complete.type
    secret_id   = cyberarksia_target_set.complete.secret_id
    secret_type = cyberarksia_target_set.complete.secret_type

    # Optional Attributes
    provision_format              = cyberarksia_target_set.complete.provision_format
    description                   = cyberarksia_target_set.complete.description
    enable_certificate_validation = cyberarksia_target_set.complete.enable_certificate_validation

    # Validation
    id_matches_name = cyberarksia_target_set.complete.id == cyberarksia_target_set.complete.name
  }
  description = "Complete target set configuration"
}

# ============================================================================
# Behavior Notes
# ============================================================================

# 1. RENAME SUPPORT
#    - Change the 'name' attribute to rename the target set
#    - Terraform performs in-place update (NOT resource recreation)
#    - ID automatically updates to match new name
#    - Old name immediately becomes unavailable (404 on lookup)
#
#    Example:
#      name = "new-name.example.com"  # Changed from "complete-example.example.com"

# 2. TYPE CHANGES
#    - All type changes supported: Domain ↔ Suffix ↔ Target
#    - In-place update (no resource recreation)
#    - No service interruption
#
#    Example:
#      type = "Suffix"  # Changed from "Domain"

# 3. CREDENTIAL ROTATION
#    - Update secret_id and/or secret_type to rotate credentials
#    - In-place update (no resource recreation)
#    - New credentials apply to future JIT sessions
#    - Active sessions continue with old credentials until expiration
#
#    Example:
#      secret_id   = cyberarksia_vm_secret.new_admin.id
#      secret_type = cyberarksia_vm_secret.new_admin.secret_type

# 4. PROVISION FORMAT CONSTRAINTS
#    - Can be set initially or added later
#    - Can be updated to a new value
#    - CANNOT be removed once set (API PATCH semantics)
#    - Attempting to clear triggers plan-time error:
#      "Cannot Clear Attribute: The provision_format field cannot be
#       removed once set due to API limitations."
#
#    Valid:
#      provision_format = "new-<user>-<session-guid>"  # Update to new value
#
#    Invalid (will error at plan time):
#      # provision_format = ""  # Cannot clear

# 5. FORWARD SLASH WARNING
#    - Names containing forward slashes (/) trigger a WARNING
#    - API accepts them during CREATE but DELETE fails with 403 Forbidden
#    - Resource can be created but NOT destroyed via Terraform
#    - Manual cleanup via SIA UI required
#
#    Example (not recommended):
#      name = "env/prod/servers"  # WARNING: Will cause deletion issues

# 6. IMPORT EXISTING TARGET SETS
#    - Import using target set name as identifier
#    - ID automatically computed from name
#
#    Command:
#      terraform import cyberarksia_target_set.complete "complete-example.example.com"
