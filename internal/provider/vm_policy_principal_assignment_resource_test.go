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

// TestAccVMPolicyPrincipalAssignment_basic tests basic CRUD lifecycle (T036)
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
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy_principal_assignment.test", "id"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy_principal_assignment.test", "principal_type", "GROUP"),
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy_principal_assignment.test", "policy_id"),
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy_principal_assignment.test", "principal_id"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy_principal_assignment.test", "principal_name", "CyberArk Guardians"),
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy_principal_assignment.test", "source_directory_name"),
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy_principal_assignment.test", "source_directory_id"),
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

// TestAccVMPolicyPrincipalAssignment_crud tests full CRUD lifecycle (T037)
func TestAccVMPolicyPrincipalAssignment_crud(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"random": {
				Source: "hashicorp/random",
			},
		},
		Steps: []resource.TestStep{
			// CREATE: Assign principal
			{
				Config: testAccVMPolicyPrincipalAssignmentConfigCRUD,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy_principal_assignment.crud_test", "id"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy_principal_assignment.crud_test", "principal_type", "GROUP"),
				),
			},
			// READ: Verify assignment exists (implicit via ImportState)
			{
				ResourceName:      "cyberarksia_vm_policy_principal_assignment.crud_test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// DELETE: Remove assignment (implicit via destroy in next test or at end)
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

// TestAccVMPolicyPrincipalAssignment_importState tests composite ID import (T039)
func TestAccVMPolicyPrincipalAssignment_importState(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"random": {
				Source: "hashicorp/random",
			},
		},
		Steps: []resource.TestStep{
			// Create resource
			{
				Config: testAccVMPolicyPrincipalAssignmentConfigImport,
			},
			// Test import with composite ID format: policy-id:principal-id:principal-type
			{
				ResourceName:      "cyberarksia_vm_policy_principal_assignment.import_test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccVMPolicyPrincipalAssignment_compositeID validates the 3-part composite ID format
func TestAccVMPolicyPrincipalAssignment_compositeID(t *testing.T) {
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
				Config: testAccVMPolicyPrincipalAssignmentConfigBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify ID contains 3 parts separated by colons
					resource.TestMatchResourceAttr(
						"cyberarksia_vm_policy_principal_assignment.test",
						"id",
						regexp.MustCompile(`^[^:]+:[^:]+:[^:]+$`),
					),
					// Verify ID ends with principal type (USER, GROUP, or ROLE)
					resource.TestMatchResourceAttr(
						"cyberarksia_vm_policy_principal_assignment.test",
						"id",
						regexp.MustCompile(`:(USER|GROUP|ROLE)$`),
					),
				),
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

const testAccVMPolicyPrincipalAssignmentConfigCRUD = `
data "cyberarksia_principal" "test_user" {
  name = "timtest@cyberark.cloud.40562"
  type = "USER"
}

resource "random_id" "crud_test" {
  byte_length = 4
}

resource "cyberarksia_vm_policy" "crud_test" {
  name          = "test-vm-policy-crud-${random_id.crud_test.hex}"
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
      username = "cruduser"
    }
  }

  fqdn_ip_targets {
    fqdn_rule {
      operator             = "SUFFIX"
      computername_pattern = "-crud"
    }
  }

  max_session_duration = 2

  access_window {
    days_of_the_week = [0, 1, 2, 3, 4, 5, 6]
  }
}

data "cyberarksia_principal" "assigned_group" {
  name = "CyberArk Guardians"
  type = "GROUP"
}

resource "cyberarksia_vm_policy_principal_assignment" "crud_test" {
  policy_id             = cyberarksia_vm_policy.crud_test.policy_id
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

const testAccVMPolicyPrincipalAssignmentConfigImport = `
data "cyberarksia_principal" "test_user" {
  name = "timtest@cyberark.cloud.40562"
  type = "USER"
}

resource "random_id" "import_test" {
  byte_length = 4
}

resource "cyberarksia_vm_policy" "import_test" {
  name          = "test-vm-policy-import-${random_id.import_test.hex}"
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
      username = "importuser"
    }
  }

  fqdn_ip_targets {
    fqdn_rule {
      operator             = "SUFFIX"
      computername_pattern = "-import"
    }
  }

  max_session_duration = 2

  access_window {
    days_of_the_week = [0, 1, 2, 3, 4, 5, 6]
  }
}

data "cyberarksia_principal" "assigned_group" {
  name = "CyberArk Guardians"
  type = "GROUP"
}

resource "cyberarksia_vm_policy_principal_assignment" "import_test" {
  policy_id             = cyberarksia_vm_policy.import_test.policy_id
  principal_id          = data.cyberarksia_principal.assigned_group.id
  principal_name        = data.cyberarksia_principal.assigned_group.name
  principal_type        = data.cyberarksia_principal.assigned_group.principal_type
  source_directory_name = data.cyberarksia_principal.assigned_group.directory_name
  source_directory_id   = data.cyberarksia_principal.assigned_group.directory_id
}
`
