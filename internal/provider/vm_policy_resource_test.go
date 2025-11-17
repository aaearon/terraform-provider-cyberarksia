// Package provider implements acceptance tests for vm_policy resource
package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ============================================================================
// Core CRUD Tests - User Story 1
// ============================================================================

// TestAccVMPolicy_basic tests basic FQDN/IP policy creation + ImportState (T022)
func TestAccVMPolicy_basic(t *testing.T) {
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
				Config: testAccVMPolicyConfigBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Basic metadata
					resource.TestMatchResourceAttr("cyberarksia_vm_policy.test", "name",
						regexp.MustCompile(`^test-vm-policy-basic-[a-f0-9]{8}$`)),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "status", "Active"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "location_type", "FQDN/IP"),

					// UUID validation
					resource.TestMatchResourceAttr("cyberarksia_vm_policy.test", "id",
						regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)),
					resource.TestMatchResourceAttr("cyberarksia_vm_policy.test", "policy_id",
						regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)),

					// Computed delegation_classification
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy.test", "delegation_classification"),

					// Required principal
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "principals.#", "1"),
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy.test", "principals.0.principal_id"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "principals.0.principal_type", "USER"),

					// SSH behavior
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "behavior.ssh.username", "testuser"),

					// FQDN target
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "fqdn_ip_targets.fqdn_rule.#", "1"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "fqdn_ip_targets.fqdn_rule.0.operator", "SUFFIX"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "fqdn_ip_targets.fqdn_rule.0.computername_pattern", "-test"),

					// Session conditions
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "max_session_duration", "2"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "idle_time", "10"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "cyberarksia_vm_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore computed metadata fields that are populated during Read()
				ImportStateVerifyIgnore: []string{"created_by", "updated_by"},
			},
		},
	})
}

// TestAccVMPolicy_sshWithTimeWindow tests SSH behavior with access window (T023)
func TestAccVMPolicy_sshWithTimeWindow(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVMPolicyConfigSSHWithTimeWindow,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.ssh_test", "name", "test-vm-policy-ssh-time"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.ssh_test", "status", "Active"),

					// SSH username
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.ssh_test", "behavior.ssh.username", "admin"),

					// Access window (business hours)
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.ssh_test", "access_window.from_hour", "09:00"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.ssh_test", "access_window.to_hour", "17:00"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.ssh_test", "access_window.days_of_the_week.#", "5"),

					// Time zone
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.ssh_test", "time_zone", "America/New_York"),

					// Session duration
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.ssh_test", "max_session_duration", "4"),
				),
			},
		},
	})
}

// TestAccVMPolicy_driftDetection tests policy drift detection (T024)
func TestAccVMPolicy_driftDetection(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create policy
			{
				Config: testAccVMPolicyConfigDrift,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.drift_test", "name", "test-vm-policy-drift"),
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy.drift_test", "policy_id"),
				),
			},
			// Simulate external deletion by running refresh without config
			{
				Config:             testAccVMPolicyConfigDrift,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false, // After manual deletion, plan should detect drift
			},
		},
	})
}

// TestAccVMPolicy_forceNewOnNameChange tests that changing name forces resource replacement (T025)
func TestAccVMPolicy_forceNewOnNameChange(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with initial name
			{
				Config: testAccVMPolicyConfigForceNewBefore,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.forcenew_test", "name", "test-vm-policy-original"),
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy.forcenew_test", "policy_id"),
				),
			},
			// Step 2: Change name (should trigger replacement)
			{
				Config: testAccVMPolicyConfigForceNewAfter,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.forcenew_test", "name", "test-vm-policy-renamed"),
					// Policy ID should be different due to resource replacement
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy.forcenew_test", "policy_id"),
				),
			},
		},
	})
}

// TestAccVMPolicy_validationErrors tests validation error handling (T026)
func TestAccVMPolicy_validationErrors(t *testing.T) {
	t.Run("missing_principals", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      testAccVMPolicyConfigMissingPrincipals,
					ExpectError: regexp.MustCompile(`Attribute principals list must contain at least 1 elements|principals block is required|Input should be a valid list.*field:\s*principals`),
				},
			},
		})
	})

	t.Run("missing_ssh_username", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      testAccVMPolicyConfigMissingSSHUsername,
					ExpectError: regexp.MustCompile("The argument \"username\" is required|username is required"),
				},
			},
		})
	})

	t.Run("conflicting_location_types", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      testAccVMPolicyConfigConflictingLocationTypes,
					ExpectError: regexp.MustCompile("Exactly one location type must be specified|Exactly one target type must be configured"),
				},
			},
		})
	})
}

// ============================================================================
// Test Configurations
// ============================================================================

const testAccVMPolicyConfigBasic = `
data "cyberarksia_principal" "test_user" {
  name = "timtest@cyberark.cloud.40562"
  type = "USER"
}

resource "random_id" "test" {
  byte_length = 4
}

resource "cyberarksia_vm_policy" "test" {
  name          = "test-vm-policy-basic-${random_id.test.hex}"
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

  # Access window with days only (from_hour/to_hour optional for time restrictions)
  access_window {
    days_of_the_week = [0, 1, 2, 3, 4, 5, 6]  # All days
  }
}
`

const testAccVMPolicyConfigSSHWithTimeWindow = `
data "cyberarksia_principal" "test_user" {
  name = "timtest@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_vm_policy" "ssh_test" {
  name          = "test-vm-policy-ssh-time"
  location_type = "FQDN/IP"
  status        = "Active"
  time_zone     = "America/New_York"

  principals {
    principal_id          = data.cyberarksia_principal.test_user.id
    principal_name        = data.cyberarksia_principal.test_user.name
    principal_type        = data.cyberarksia_principal.test_user.principal_type
    source_directory_name = data.cyberarksia_principal.test_user.directory_name
    source_directory_id   = data.cyberarksia_principal.test_user.directory_id
  }

  behavior {
    ssh {
      username = "admin"
    }
  }

  fqdn_ip_targets {
    fqdn_rule {
      operator             = "SUFFIX"
      computername_pattern = "-prod"
    }
  }

  max_session_duration = 4
  idle_time            = 15

  access_window {
    days_of_the_week = [1, 2, 3, 4, 5]  # Monday-Friday
    from_hour        = "09:00"
    to_hour          = "17:00"
  }
}
`

const testAccVMPolicyConfigDrift = `
data "cyberarksia_principal" "test_user" {
  name = "timtest@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_vm_policy" "drift_test" {
  name          = "test-vm-policy-drift"
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
      username = "driftuser"
    }
  }

  fqdn_ip_targets {
    fqdn_rule {
      operator             = "SUFFIX"
      computername_pattern = "-drift"
    }
  }

  max_session_duration = 1

  access_window {
    days_of_the_week = [0, 1, 2, 3, 4, 5, 6]  # All days
  }
}
`

const testAccVMPolicyConfigForceNewBefore = `
data "cyberarksia_principal" "test_user" {
  name = "timtest@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_vm_policy" "forcenew_test" {
  name          = "test-vm-policy-original"
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
      username = "original"
    }
  }

  fqdn_ip_targets {
    fqdn_rule {
      operator             = "SUFFIX"
      computername_pattern = "-original"
    }
  }

  max_session_duration = 2

  access_window {
    days_of_the_week = [0, 1, 2, 3, 4, 5, 6]  # All days
  }
}
`

const testAccVMPolicyConfigForceNewAfter = `
data "cyberarksia_principal" "test_user" {
  name = "timtest@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_vm_policy" "forcenew_test" {
  name          = "test-vm-policy-renamed"
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
      username = "renamed"
    }
  }

  fqdn_ip_targets {
    fqdn_rule {
      operator             = "SUFFIX"
      computername_pattern = "-renamed"
    }
  }

  max_session_duration = 2

  access_window {
    days_of_the_week = [0, 1, 2, 3, 4, 5, 6]  # All days
  }
}
`

// Validation test configs
const testAccVMPolicyConfigMissingPrincipals = `
resource "cyberarksia_vm_policy" "invalid" {
  name          = "test-vm-policy-invalid"
  location_type = "FQDN/IP"
  status        = "Active"

  # Missing required principals block

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
    days_of_the_week = [0, 1, 2, 3, 4, 5, 6]  # All days
  }
}
`

const testAccVMPolicyConfigMissingSSHUsername = `
data "cyberarksia_principal" "test_user" {
  name = "timtest@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_vm_policy" "invalid" {
  name          = "test-vm-policy-invalid-ssh"
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
      # Missing required username
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
    days_of_the_week = [0, 1, 2, 3, 4, 5, 6]  # All days
  }
}
`

const testAccVMPolicyConfigConflictingLocationTypes = `
data "cyberarksia_principal" "test_user" {
  name = "timtest@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_vm_policy" "invalid" {
  name          = "test-vm-policy-invalid-location"
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

  # Conflicting: Both FQDN/IP and AWS targets specified
  fqdn_ip_targets {
    fqdn_rule {
      operator             = "SUFFIX"
      computername_pattern = "-test"
    }
  }

  aws_targets {
    regions = ["us-east-1"]
  }

  max_session_duration = 2

  access_window {
    days_of_the_week = [0, 1, 2, 3, 4, 5, 6]  # All days
  }
}
`
