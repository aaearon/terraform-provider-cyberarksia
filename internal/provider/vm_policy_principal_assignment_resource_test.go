// Package provider implements acceptance tests for VM policy principal assignment resource
package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ============================================================================
// VM Policy Principal Assignment Acceptance Tests
// ============================================================================

// TestAccVMPolicyPrincipalAssignment_basic tests CRUD lifecycle with ImportState and ID validation (T036, T037, T039, T040)
// This consolidated test verifies:
// - Assignment resource creation and Read
// - Composite ID format (3-part: policy-id:principal-id:principal-type)
// - Inline principal preservation (Session 4 fix)
// - ImportState functionality
func TestAccVMPolicyPrincipalAssignment_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"random": {
				Source: "hashicorp/random",
			},
		},
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccVMPolicyPrincipalAssignmentConfigBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Assignment resource attributes
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy_principal_assignment.test", "id"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy_principal_assignment.test", "principal_type", "GROUP"),
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy_principal_assignment.test", "policy_id"),
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy_principal_assignment.test", "principal_id"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy_principal_assignment.test", "principal_name", "CyberArk Guardians"),
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy_principal_assignment.test", "source_directory_name"),
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy_principal_assignment.test", "source_directory_id"),

					// Verify composite ID format: policy-id:principal-id:principal-type
					resource.TestMatchResourceAttr(
						"cyberarksia_vm_policy_principal_assignment.test",
						"id",
						regexp.MustCompile(`^[^:]+:[^:]+:[^:]+$`),
					),
					// Verify ID ends with valid principal type
					resource.TestMatchResourceAttr(
						"cyberarksia_vm_policy_principal_assignment.test",
						"id",
						regexp.MustCompile(`:(USER|GROUP|ROLE)$`),
					),

					// CRITICAL: Test Session 4 fix - inline principals preserved
					// Policy should still have exactly 1 inline principal (from policy definition)
					// The assigned principal is managed separately, not in policy's principals attribute
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "principals.#", "1"),
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy.test", "principals.0.principal_id"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "principals.0.principal_type", "USER"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "principals.0.principal_name", "timtest@cyberark.cloud.40562"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "cyberarksia_vm_policy_principal_assignment.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccVMPolicyPrincipalAssignment_duplicateDetection tests duplicate assignment handling (T038)
func TestAccVMPolicyPrincipalAssignment_duplicateDetection(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"random": {
				Source: "hashicorp/random",
			},
		},
		Steps: []resource.TestStep{
			{
				Config:      testAccVMPolicyPrincipalAssignmentConfigDuplicate,
				ExpectError: regexp.MustCompile("Principal Already Assigned"),
			},
		},
	})
}

// ============================================================================
// Test Configurations
// ============================================================================

const testAccVMPolicyPrincipalAssignmentConfigBasic = `
data "cyberarksia_principal" "test_user" {
  name = "timtest@cyberark.cloud.40562"
  type = "USER"
}

resource "random_id" "test" {
  byte_length = 4
}

resource "cyberarksia_vm_policy" "test" {
  name          = "test-vm-policy-principal-${random_id.test.hex}"
  location_type = "FQDN/IP"
  status        = "Active"

  principals {
    principal_id          = data.cyberarksia_principal.test_user.id
    principal_name        = data.cyberarksia_principal.test_user.name
    principal_type        = data.cyberarksia_principal.test_user.principal_type
    source_directory_name = data.cyberarksia_principal.test_user.directory_name
    source_directory_id   = data.cyberarksia_principal.test_user.directory_id
  }

  behavior {
    ssh {
      username = "testuser"
    }
  }

  fqdn_ip_targets {
    fqdn_rule {
      operator             = "SUFFIX"
      computername_pattern = "-test"
    }
  }

  max_session_duration = 2

  access_window {
    days_of_the_week = [0, 1, 2, 3, 4, 5, 6]
  }
}

# Second principal to assign via assignment resource (use a GROUP instead to avoid duplicate)
data "cyberarksia_principal" "assigned_group" {
  name = "CyberArk Guardians"
  type = "GROUP"
}

resource "cyberarksia_vm_policy_principal_assignment" "test" {
  policy_id             = cyberarksia_vm_policy.test.policy_id
  principal_id          = data.cyberarksia_principal.assigned_group.id
  principal_name        = data.cyberarksia_principal.assigned_group.name
  principal_type        = data.cyberarksia_principal.assigned_group.principal_type
  source_directory_name = data.cyberarksia_principal.assigned_group.directory_name
  source_directory_id   = data.cyberarksia_principal.assigned_group.directory_id
}
`

const testAccVMPolicyPrincipalAssignmentConfigDuplicate = `
data "cyberarksia_principal" "test_user" {
  name = "timtest@cyberark.cloud.40562"
  type = "USER"
}

resource "random_id" "dup_test" {
  byte_length = 4
}

resource "cyberarksia_vm_policy" "dup_test" {
  name          = "test-vm-policy-dup-${random_id.dup_test.hex}"
  location_type = "FQDN/IP"
  status        = "Active"

  principals {
    principal_id          = data.cyberarksia_principal.test_user.id
    principal_name        = data.cyberarksia_principal.test_user.name
    principal_type        = data.cyberarksia_principal.test_user.principal_type
    source_directory_name = data.cyberarksia_principal.test_user.directory_name
    source_directory_id   = data.cyberarksia_principal.test_user.directory_id
  }

  behavior {
    ssh {
      username = "dupuser"
    }
  }

  fqdn_ip_targets {
    fqdn_rule {
      operator             = "SUFFIX"
      computername_pattern = "-dup"
    }
  }

  max_session_duration = 2

  access_window {
    days_of_the_week = [0, 1, 2, 3, 4, 5, 6]
  }
}

# Try to assign the same principal that's already in inline principals (should fail)
resource "cyberarksia_vm_policy_principal_assignment" "dup_test" {
  policy_id             = cyberarksia_vm_policy.dup_test.policy_id
  principal_id          = data.cyberarksia_principal.test_user.id
  principal_name        = data.cyberarksia_principal.test_user.name
  principal_type        = data.cyberarksia_principal.test_user.principal_type
  source_directory_name = data.cyberarksia_principal.test_user.directory_name
  source_directory_id   = data.cyberarksia_principal.test_user.directory_id
}
`
