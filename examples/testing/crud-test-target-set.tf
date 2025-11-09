# CRUD Test Template for cyberarksia_target_set
# This template validates CREATE → READ → UPDATE → DELETE workflow

# ⚠️ IMPORTANT: This template requires an existing VM secret
# Update the secret reference below with your actual secret ID or resource reference

terraform {
  required_providers {
    cyberarksia = {
      source  = "aaearon/cyberarksia"
      version = "0.1.0"
    }
  }
}

# ============================================================================
# CONFIGURATION - Define stable test suffix
# ============================================================================
# Change this suffix manually to create a new test instance
# DO NOT use timestamp() - it forces replacement on every plan
locals {
  test_suffix = "01" # Increment manually for each test run (01, 02, 03, etc.)
}

# ============================================================================
# STEP 0: PREREQUISITE - VM Secret (if not using existing)
# ============================================================================
# Uncomment this block if you want to create a test secret:
#
# resource "cyberarksia_virtual_machine_secret" "test" {
#   secret_name          = "crud-test-secret-${local.test_suffix}"
#   secret_type          = "ProvisionerUser"
#   provisioner_username = "testadmin"
#   provisioner_password = "TestPassword123!"
# }

# ============================================================================
# STEP 1: CREATE - Create a domain-based target set
# ============================================================================

resource "cyberarksia_target_set" "test" {
  name        = "crud-test-${local.test_suffix}.example.com"
  type        = "Domain"
  secret_id   = "YOUR_SECRET_ID_HERE" # REQUIRED: Update this
  secret_type = "ProvisionerUser"     # REQUIRED: Update if using PCloudAccount

  # Optional attributes
  description                   = "CRUD validation test target set"
  provision_format              = "<user>-test-<session-guid>"
  enable_certificate_validation = true
}

# Alternative: Reference the test secret created above
# Uncomment if using the test secret from STEP 0:
# resource "cyberarksia_target_set" "test" {
#   name        = "crud-test-${local.test_suffix}.example.com"
#   type        = "Domain"
#   secret_id   = cyberarksia_virtual_machine_secret.test.id
#   secret_type = cyberarksia_virtual_machine_secret.test.secret_type
#
#   description                   = "CRUD validation test target set"
#   provision_format              = "<user>-test-<session-guid>"
#   enable_certificate_validation = true
# }

# ============================================================================
# STEP 2: READ - Verify state matches configuration
# ============================================================================

output "create_validation" {
  value = {
    id                            = cyberarksia_target_set.test.id
    name                          = cyberarksia_target_set.test.name
    type                          = cyberarksia_target_set.test.type
    secret_id                     = cyberarksia_target_set.test.secret_id
    secret_type                   = cyberarksia_target_set.test.secret_type
    provision_format              = cyberarksia_target_set.test.provision_format
    description                   = cyberarksia_target_set.test.description
    enable_certificate_validation = cyberarksia_target_set.test.enable_certificate_validation
    id_matches_name               = cyberarksia_target_set.test.id == cyberarksia_target_set.test.name
  }
  description = "CREATE validation - Verify all attributes populated correctly"
}

# ============================================================================
# VALIDATION CHECKLIST - CREATE
# ============================================================================
# [ ] id equals name (name-as-ID pattern)
# [ ] name matches input (contains "crud-test-")
# [ ] type is "Domain"
# [ ] secret_id is populated
# [ ] secret_type is "ProvisionerUser" (or "PCloudAccount" if updated)
# [ ] provision_format is "<user>-test-<session-guid>"
# [ ] description is "CRUD validation test target set"
# [ ] enable_certificate_validation is true
# [ ] terraform plan shows no changes (state matches config)

# ============================================================================
# STEP 3: UPDATE - Modify attributes
# ============================================================================
# After initial creation, update the resource block above with these changes:
#
# 1. Change type from "Domain" to "Suffix"
# 2. Update description
# 3. Change provision_format
# 4. Toggle enable_certificate_validation
#
# Example updated resource:
# resource "cyberarksia_target_set" "test" {
#   name        = "crud-test-${local.test_suffix}.example.com"
#   type        = "Suffix"  # CHANGED from "Domain"
#   secret_id   = "YOUR_SECRET_ID_HERE"
#   secret_type = "ProvisionerUser"
#
#   description                   = "UPDATED: CRUD validation test - modified"  # CHANGED
#   provision_format              = "jit-<user>-<session-guid>"                  # CHANGED
#   enable_certificate_validation = false                                        # CHANGED
# }

output "update_validation" {
  value = {
    name_unchanged                = cyberarksia_target_set.test.name # Should NOT change
    id_still_matches_name         = cyberarksia_target_set.test.id == cyberarksia_target_set.test.name
    type                          = cyberarksia_target_set.test.type
    provision_format              = cyberarksia_target_set.test.provision_format
    description                   = cyberarksia_target_set.test.description
    enable_certificate_validation = cyberarksia_target_set.test.enable_certificate_validation
  }
  description = "UPDATE validation - Verify changes applied without recreation"
}

# ============================================================================
# VALIDATION CHECKLIST - UPDATE
# ============================================================================
# [ ] type changed to "Suffix" (no resource recreation)
# [ ] description updated to "UPDATED: CRUD validation test - modified"
# [ ] provision_format changed to "jit-<user>-<session-guid>"
# [ ] enable_certificate_validation changed to false
# [ ] id still equals name (unchanged)
# [ ] name unchanged (no rename in this test)
# [ ] terraform plan shows no changes after apply

# ============================================================================
# STEP 4: DELETE - Destroy the resource
# ============================================================================
# Run: terraform destroy
# Expected: Target set deleted successfully (HTTP 204 No Content)

# ============================================================================
# VALIDATION CHECKLIST - DELETE
# ============================================================================
# [ ] terraform destroy completes without errors
# [ ] Resource removed from state
# [ ] Manual verification: Target set no longer visible in SIA UI

# ============================================================================
# ADVANCED TEST: Rename
# ============================================================================
# After CREATE, update the name attribute:
# resource "cyberarksia_target_set" "test" {
#   name        = "crud-test-${local.test_suffix}-renamed.example.com"  # CHANGED
#   ...
# }
#
# VALIDATION CHECKLIST - RENAME:
# [ ] terraform plan shows update in-place (NOT replacement)
# [ ] After apply, id equals new name
# [ ] Old name returns 404 if queried manually
# [ ] terraform plan shows no changes after rename

# ============================================================================
# ADVANCED TEST: provision_format Clearing Prevention
# ============================================================================
# After CREATE, try to remove provision_format:
# resource "cyberarksia_target_set" "test" {
#   name        = "crud-test-${local.test_suffix}.example.com"
#   type        = "Domain"
#   secret_id   = "YOUR_SECRET_ID_HERE"
#   secret_type = "ProvisionerUser"
#   # provision_format = ""  # Try to clear - should ERROR
# }
#
# VALIDATION CHECKLIST - CLEARING PREVENTION:
# [ ] terraform plan shows error: "Cannot Clear Attribute"
# [ ] Error message explains: "cannot be removed once set due to API limitations"
# [ ] Plan does NOT proceed (blocked at plan phase)

# ============================================================================
# ADVANCED TEST: Forward Slash Warning
# ============================================================================
# Create target set with forward slash in name:
# resource "cyberarksia_target_set" "warning_test" {
#   name        = "env/test/servers"  # Contains forward slashes
#   type        = "Domain"
#   secret_id   = "YOUR_SECRET_ID_HERE"
#   secret_type = "ProvisionerUser"
# }
#
# VALIDATION CHECKLIST - FORWARD SLASH WARNING:
# [ ] terraform plan shows WARNING (not error)
# [ ] Warning mentions "forward slashes which will cause deletion failures"
# [ ] Resource CAN be created (warning doesn't block)
# [ ] terraform destroy FAILS with 403 Forbidden
# [ ] Manual deletion via SIA UI required

# ============================================================================
# SUMMARY
# ============================================================================
# This template validates:
# ✓ CREATE: Domain-based target set with all optional attributes
# ✓ READ: All attributes populated correctly
# ✓ UPDATE: Type, description, provision_format, enable_certificate_validation
# ✓ DELETE: Successful cleanup
# ✓ ADVANCED: Rename, clearing prevention, forward slash warning
#
# Expected Total Test Duration: 5-10 minutes
# Expected API Calls: ~15-20 (create, multiple reads, updates, delete)
