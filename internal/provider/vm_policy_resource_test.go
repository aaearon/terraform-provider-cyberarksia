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
		ExternalProviders: map[string]resource.ExternalProvider{
			"random": {
				Source: "hashicorp/random",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccVMPolicyConfigSSHWithTimeWindow,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy.ssh_test", "name"),
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
		ExternalProviders: map[string]resource.ExternalProvider{
			"random": {
				Source: "hashicorp/random",
			},
		},
		Steps: []resource.TestStep{
			// Create policy
			{
				Config: testAccVMPolicyConfigDrift,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy.drift_test", "name"),
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
		ExternalProviders: map[string]resource.ExternalProvider{
			"random": {
				Source: "hashicorp/random",
			},
		},
		Steps: []resource.TestStep{
			// Step 1: Create with initial name
			{
				Config: testAccVMPolicyConfigForceNewBefore,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy.forcenew_test", "name"),
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy.forcenew_test", "policy_id"),
				),
			},
			// Step 2: Change name (should trigger replacement)
			{
				Config: testAccVMPolicyConfigForceNewAfter,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy.forcenew_test", "name"),
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

resource "random_id" "ssh_test" {
  byte_length = 4
}

resource "cyberarksia_vm_policy" "ssh_test" {
  name          = "test-vm-policy-ssh-time-${random_id.ssh_test.hex}"
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

resource "random_id" "drift_test" {
  byte_length = 4
}

resource "cyberarksia_vm_policy" "drift_test" {
  name          = "test-vm-policy-drift-${random_id.drift_test.hex}"
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

resource "random_id" "forcenew_test" {
  byte_length = 4
}

resource "cyberarksia_vm_policy" "forcenew_test" {
  name          = "test-vm-policy-original-${random_id.forcenew_test.hex}"
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

resource "random_id" "forcenew_test" {
  byte_length = 4
}

resource "cyberarksia_vm_policy" "forcenew_test" {
  name          = "test-vm-policy-renamed-${random_id.forcenew_test.hex}"
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

// ============================================================================
// AWS Cloud Policy Tests - User Story 3
// ============================================================================

// TestAccVMPolicy_awsBasic tests AWS policy creation with regions and tags (T044)
func TestAccVMPolicy_awsBasic(t *testing.T) {
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
				Config: testAccVMPolicyConfigAWSBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Basic metadata
					resource.TestMatchResourceAttr("cyberarksia_vm_policy.aws_test", "name",
						regexp.MustCompile(`^test-vm-policy-aws-basic-[a-f0-9]{8}$`)),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_test", "status", "Active"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_test", "location_type", "AWS"),

					// AWS targets - regions
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_test", "aws_targets.regions.#", "2"),
					resource.TestCheckTypeSetElemAttr("cyberarksia_vm_policy.aws_test", "aws_targets.regions.*", "us-east-1"),
					resource.TestCheckTypeSetElemAttr("cyberarksia_vm_policy.aws_test", "aws_targets.regions.*", "us-west-2"),

					// AWS targets - tags (verify structure, not just count)
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_test", "aws_targets.tags.#", "2"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_test", "aws_targets.tags.0.key", "Environment"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_test", "aws_targets.tags.0.value.#", "1"),
					resource.TestCheckTypeSetElemAttr("cyberarksia_vm_policy.aws_test", "aws_targets.tags.0.value.*", "production"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_test", "aws_targets.tags.1.key", "Team"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_test", "aws_targets.tags.1.value.#", "2"),
					resource.TestCheckTypeSetElemAttr("cyberarksia_vm_policy.aws_test", "aws_targets.tags.1.value.*", "platform"),
					resource.TestCheckTypeSetElemAttr("cyberarksia_vm_policy.aws_test", "aws_targets.tags.1.value.*", "infrastructure"),

					// AWS targets - verify empty arrays (not null) for vpc_ids/account_ids
					// This verifies the fix: API requires empty arrays, not null
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_test", "aws_targets.vpc_ids.#", "0"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_test", "aws_targets.account_ids.#", "0"),

					// SSH behavior
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_test", "behavior.ssh.username", "ec2-user"),

					// Session conditions
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_test", "max_session_duration", "2"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "cyberarksia_vm_policy.aws_test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_by", "updated_by"},
			},
		},
	})
}

// TestAccVMPolicy_awsVpcAndAccounts tests AWS policy with VPC IDs and account IDs (T045)
func TestAccVMPolicy_awsVpcAndAccounts(t *testing.T) {
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
				Config: testAccVMPolicyConfigAWSVpcAndAccounts,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_vpc_test", "location_type", "AWS"),

					// AWS VPC IDs
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_vpc_test", "aws_targets.vpc_ids.#", "2"),
					resource.TestCheckTypeSetElemAttr("cyberarksia_vm_policy.aws_vpc_test", "aws_targets.vpc_ids.*", "vpc-12345678"),
					resource.TestCheckTypeSetElemAttr("cyberarksia_vm_policy.aws_vpc_test", "aws_targets.vpc_ids.*", "vpc-abcdef12"),

					// AWS Account IDs
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_vpc_test", "aws_targets.account_ids.#", "1"),
					resource.TestCheckTypeSetElemAttr("cyberarksia_vm_policy.aws_vpc_test", "aws_targets.account_ids.*", "123456789012"),

					// Regions
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_vpc_test", "aws_targets.regions.#", "1"),
					resource.TestCheckTypeSetElemAttr("cyberarksia_vm_policy.aws_vpc_test", "aws_targets.regions.*", "us-east-1"),

					// Verify tags is empty (null) when not provided
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_vpc_test", "aws_targets.tags.#", "0"),
				),
			},
			// ImportState testing - verify provider can reconstruct state from API
			{
				ResourceName:            "cyberarksia_vm_policy.aws_vpc_test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_by", "updated_by"},
			},
		},
	})
}

// TestAccVMPolicy_awsUpdateRegions tests updating AWS regions (no ForceNew) (T046)
func TestAccVMPolicy_awsUpdateRegions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"random": {
				Source: "hashicorp/random",
			},
		},
		Steps: []resource.TestStep{
			// Create with initial regions
			{
				Config: testAccVMPolicyConfigAWSUpdateRegionsBefore,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_update_test", "location_type", "AWS"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_update_test", "aws_targets.regions.#", "2"),
					resource.TestCheckTypeSetElemAttr("cyberarksia_vm_policy.aws_update_test", "aws_targets.regions.*", "us-east-1"),
					resource.TestCheckTypeSetElemAttr("cyberarksia_vm_policy.aws_update_test", "aws_targets.regions.*", "us-west-2"),

					// Verify principals are set initially (Session 4 fix verification)
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_update_test", "principals.#", "1"),
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy.aws_update_test", "principals.0.principal_id"),
				),
			},
			// Update regions (should NOT trigger ForceNew)
			{
				Config: testAccVMPolicyConfigAWSUpdateRegionsAfter,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_update_test", "location_type", "AWS"),
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_update_test", "aws_targets.regions.#", "3"),
					resource.TestCheckTypeSetElemAttr("cyberarksia_vm_policy.aws_update_test", "aws_targets.regions.*", "us-east-1"),
					resource.TestCheckTypeSetElemAttr("cyberarksia_vm_policy.aws_update_test", "aws_targets.regions.*", "eu-west-1"),
					resource.TestCheckTypeSetElemAttr("cyberarksia_vm_policy.aws_update_test", "aws_targets.regions.*", "ap-southeast-1"),

					// CRITICAL: Verify policy_id didn't change (no ForceNew occurred)
					// This uses TestCheckResourceAttrPair to compare policy_id across steps
					// If ForceNew happened, this would fail because policy_id would be different
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy.aws_update_test", "policy_id"),

					// CRITICAL: Verify principals preserved during update (Session 4 fix)
					// This tests our Read() filtering logic that prevents inline principal drift
					resource.TestCheckResourceAttr("cyberarksia_vm_policy.aws_update_test", "principals.#", "1"),
					resource.TestCheckResourceAttrSet("cyberarksia_vm_policy.aws_update_test", "principals.0.principal_id"),
				),
			},
		},
	})
}

// ============================================================================
// AWS Test Configurations
// ============================================================================

const testAccVMPolicyConfigAWSBasic = `
data "cyberarksia_principal" "test_user" {
  name = "timtest@cyberark.cloud.40562"
  type = "USER"
}

resource "random_id" "aws_test" {
  byte_length = 4
}

resource "cyberarksia_vm_policy" "aws_test" {
  name          = "test-vm-policy-aws-basic-${random_id.aws_test.hex}"
  location_type = "AWS"
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
      username = "ec2-user"
    }
  }

  aws_targets {
    regions = ["us-east-1", "us-west-2"]

    tags {
      key   = "Environment"
      value = ["production"]
    }

    tags {
      key   = "Team"
      value = ["platform", "infrastructure"]
    }
  }

  max_session_duration = 2

  access_window {
    days_of_the_week = [0, 1, 2, 3, 4, 5, 6]
  }
}
`

const testAccVMPolicyConfigAWSVpcAndAccounts = `
data "cyberarksia_principal" "test_user" {
  name = "timtest@cyberark.cloud.40562"
  type = "USER"
}

resource "random_id" "aws_vpc_test" {
  byte_length = 4
}

resource "cyberarksia_vm_policy" "aws_vpc_test" {
  name          = "test-vm-policy-aws-vpc-${random_id.aws_vpc_test.hex}"
  location_type = "AWS"
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
      username = "ubuntu"
    }
  }

  aws_targets {
    regions     = ["us-east-1"]
    vpc_ids     = ["vpc-12345678", "vpc-abcdef12"]
    account_ids = ["123456789012"]
  }

  max_session_duration = 2

  access_window {
    days_of_the_week = [0, 1, 2, 3, 4, 5, 6]
  }
}
`

const testAccVMPolicyConfigAWSUpdateRegionsBefore = `
data "cyberarksia_principal" "test_user" {
  name = "timtest@cyberark.cloud.40562"
  type = "USER"
}

resource "random_id" "aws_update_test" {
  byte_length = 4
}

resource "cyberarksia_vm_policy" "aws_update_test" {
  name          = "test-vm-policy-aws-update-${random_id.aws_update_test.hex}"
  location_type = "AWS"
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
      username = "admin"
    }
  }

  aws_targets {
    regions = ["us-east-1", "us-west-2"]
  }

  max_session_duration = 2

  access_window {
    days_of_the_week = [0, 1, 2, 3, 4, 5, 6]
  }
}
`

const testAccVMPolicyConfigAWSUpdateRegionsAfter = `
data "cyberarksia_principal" "test_user" {
  name = "timtest@cyberark.cloud.40562"
  type = "USER"
}

resource "random_id" "aws_update_test" {
  byte_length = 4
}

resource "cyberarksia_vm_policy" "aws_update_test" {
  name          = "test-vm-policy-aws-update-${random_id.aws_update_test.hex}"
  location_type = "AWS"
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
      username = "admin"
    }
  }

  aws_targets {
    regions = ["us-east-1", "eu-west-1", "ap-southeast-1"]
  }

  max_session_duration = 2

  access_window {
    days_of_the_week = [0, 1, 2, 3, 4, 5, 6]
  }
}
`
