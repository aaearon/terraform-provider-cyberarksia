# CRUD Validation Template for VM Access Policies
# This template helps validate Create, Read, Update, Delete lifecycle
#
# Usage: Copy to /tmp, customize values, run terraform apply/destroy cycle
# See: examples/testing/TESTING-GUIDE.md for complete validation workflow

# Prerequisites:
# 1. Valid CyberArk SIA credentials (CYBERARK_USERNAME, CYBERARK_PASSWORD)
# 2. At least one principal available for assignment
# 3. Provider configured (see examples/provider/)

# Step 1: LOOKUP TEST PRINCIPAL
# Use data source to find existing principal (avoids hardcoding UUIDs)
data "cyberarksia_principal" "test_user" {
  name = "YOUR_TEST_USER@example.com" # REPLACE WITH ACTUAL USER
  type = "USER"
}

# Step 2: CREATE - Basic FQDN/IP Policy with SSH
resource "cyberarksia_vm_policy" "crud_test" {
  name          = "CRUD-Test-VM-Policy-${formatdate("YYYYMMDD-hhmm", timestamp())}"
  location_type = "FQDN/IP"
  status        = "Active"
  description   = "CRUD validation test policy"

  # Required initial principal
  principal {
    principal_id          = data.cyberarksia_principal.test_user.principal_id
    principal_name        = data.cyberarksia_principal.test_user.principal_name
    principal_type        = data.cyberarksia_principal.test_user.principal_type
    source_directory_name = data.cyberarksia_principal.test_user.source_directory_name
    source_directory_id   = data.cyberarksia_principal.test_user.source_directory_id
  }

  # SSH connection behavior
  behavior {
    ssh {
      username = "testuser"
    }
  }

  # FQDN target rules
  fqdn_ip_targets {
    fqdn_rule {
      operator             = "SUFFIX"
      computername_pattern = "-test"
      domain               = "example.com"
    }
  }

  # Initial conditions
  max_session_duration = 2 # Will update to 4 in UPDATE test
  idle_time            = 10

  tags = ["crud-test", "terraform"]
}

# Step 3: PRINCIPAL ASSIGNMENT (tests independent resource)
data "cyberarksia_principal" "test_group" {
  name = "Test-Group" # REPLACE WITH ACTUAL GROUP
  type = "GROUP"
}

resource "cyberarksia_vm_policy_principal_assignment" "crud_test_group" {
  policy_id             = cyberarksia_vm_policy.crud_test.policy_id
  principal_id          = data.cyberarksia_principal.test_group.principal_id
  principal_name        = data.cyberarksia_principal.test_group.principal_name
  principal_type        = data.cyberarksia_principal.test_group.principal_type
  source_directory_name = data.cyberarksia_principal.test_group.source_directory_name
  source_directory_id   = data.cyberarksia_principal.test_group.source_directory_id
}

# VALIDATION OUTPUTS
# These outputs help verify CRUD operations succeeded

output "validation_summary" {
  value = {
    policy_id     = cyberarksia_vm_policy.crud_test.policy_id
    policy_name   = cyberarksia_vm_policy.crud_test.name
    status        = cyberarksia_vm_policy.crud_test.status
    location_type = cyberarksia_vm_policy.crud_test.location_type

    # Verify initial principal
    initial_principal_count = length(cyberarksia_vm_policy.crud_test.principal)

    # Verify conditions
    max_session_duration = cyberarksia_vm_policy.crud_test.max_session_duration
    idle_time            = cyberarksia_vm_policy.crud_test.idle_time

    # Verify computed fields
    delegation_classification = cyberarksia_vm_policy.crud_test.delegation_classification
    created_by_user           = cyberarksia_vm_policy.crud_test.created_by.user

    # Verify assignment
    assignment_id = cyberarksia_vm_policy_principal_assignment.crud_test_group.id
  }
  description = "CRUD validation checks - verify all values match expected"
}

output "create_checklist" {
  value = <<-EOT
  CREATE Validation Checklist:
  ✓ Policy created with unique name
  ✓ policy_id populated (UUID format)
  ✓ status = "Active"
  ✓ location_type = "FQDN/IP"
  ✓ At least 1 principal assigned
  ✓ max_session_duration = 2
  ✓ delegation_classification computed
  ✓ created_by metadata populated
  ✓ Principal assignment created successfully

  Next: Run 'terraform apply' again to verify READ (no changes expected)
  EOT
}

# Step 4: UPDATE TEST (uncomment after CREATE verification)
# Modify max_session_duration from 2 to 4, add tags
# Expected: Terraform shows 2 changes, preserves principals

# resource "cyberarksia_vm_policy" "crud_test" {
#   # ... keep all fields same except:
#   max_session_duration = 4  # UPDATED
#   tags = ["crud-test", "terraform", "updated"]  # ADDED TAG
# }

output "update_checklist" {
  value = <<-EOT
  UPDATE Validation Checklist (after uncommenting changes above):
  ✓ max_session_duration changed from 2 to 4
  ✓ tags updated with "updated" tag
  ✓ All other fields preserved
  ✓ Principal count unchanged (both inline + assigned preserved)
  ✓ updated_by metadata updated

  Next: Run 'terraform destroy' to verify DELETE
  EOT
}

# Step 5: DELETE TEST
# Run: terraform destroy
# Expected: Both policy and assignment deleted cleanly
# Verify: No errors, state file empty after destroy

output "delete_checklist" {
  value = <<-EOT
  DELETE Validation Checklist (after destroy):
  ✓ Principal assignment deleted first (dependency ordering)
  ✓ VM policy deleted successfully
  ✓ No orphaned resources
  ✓ terraform.tfstate shows empty resources
  ✓ No errors in output

  CRUD cycle complete!
  EOT
}
