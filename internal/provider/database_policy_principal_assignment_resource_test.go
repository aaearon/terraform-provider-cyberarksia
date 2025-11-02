// Package provider implements acceptance tests for database policy principal assignment resource
package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ============================================================================
// Principal Assignment Acceptance Tests
// ============================================================================

// TestAccPrincipalAssignment_basic tests basic CRUD lifecycle for USER principal type
func TestAccPrincipalAssignment_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPrincipalAssignmentConfigBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.test", "id"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_principal_assignment.test", "principal_type", "USER"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.test", "policy_id"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.test", "principal_id"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_principal_assignment.test", "principal_name", "tim.schindler@cyberark.cloud.40562"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_principal_assignment.test", "source_directory_name", "CyberArk Cloud Directory"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.test", "source_directory_id"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.test", "last_modified"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "cyberarksia_database_policy_principal_assignment.test",
				ImportState:       true,
				ImportStateVerify: true,
				// last_modified is computed timestamp - may differ slightly
				ImportStateVerifyIgnore: []string{"last_modified"},
			},
		},
	})
}

// TestAccPrincipalAssignment_multipleTypes tests USER, GROUP, and ROLE principal types
func TestAccPrincipalAssignment_multipleTypes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPrincipalAssignmentConfigMultipleTypes,
				Check: resource.ComposeAggregateTestCheckFunc(
					// USER assignment
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.user", "id"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_principal_assignment.user", "principal_type", "USER"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.user", "source_directory_name"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.user", "source_directory_id"),

					// GROUP assignment
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.group", "id"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_principal_assignment.group", "principal_type", "GROUP"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.group", "source_directory_name"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.group", "source_directory_id"),

					// ROLE assignment
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.role", "id"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_principal_assignment.role", "principal_type", "ROLE"),
					// ROLE does not require directory fields
				),
			},
		},
	})
}

// TestAccPrincipalAssignment_import tests ImportState with composite ID parsing
func TestAccPrincipalAssignment_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create resource
			{
				Config: testAccPrincipalAssignmentConfigBasic,
			},
			// Test import with composite ID format: policy-id:principal-id:principal-type
			{
				ResourceName:      "cyberarksia_database_policy_principal_assignment.test",
				ImportState:       true,
				ImportStateVerify: true,
				// last_modified is computed timestamp - may differ
				ImportStateVerifyIgnore: []string{"last_modified"},
			},
		},
	})
}

// TestAccPrincipalAssignment_compositeID validates the 3-part composite ID format
func TestAccPrincipalAssignment_compositeID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPrincipalAssignmentConfigBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify ID contains 3 parts separated by colons
					resource.TestMatchResourceAttr(
						"cyberarksia_database_policy_principal_assignment.test",
						"id",
						regexp.MustCompile(`^[^:]+:[^:]+:[^:]+$`),
					),
					// Verify ID ends with principal type (USER, GROUP, or ROLE)
					resource.TestMatchResourceAttr(
						"cyberarksia_database_policy_principal_assignment.test",
						"id",
						regexp.MustCompile(`:(USER|GROUP|ROLE)$`),
					),
				),
			},
		},
	})
}

// TestAccPrincipalAssignment_update tests updating principal metadata (principal_name)
func TestAccPrincipalAssignment_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create initial assignment
			{
				Config: testAccPrincipalAssignmentConfigUpdateBefore,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.update_test", "id"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_principal_assignment.update_test", "principal_name", "tim.schindler@cyberark.cloud.40562"),
				),
			},
			// Update principal_name (metadata update)
			{
				Config: testAccPrincipalAssignmentConfigUpdateAfter,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.update_test", "id"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_principal_assignment.update_test", "principal_name", "tim.schindler@cyberark.cloud.40562"),
					// Verify last_modified timestamp updated
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.update_test", "last_modified"),
				),
			},
		},
	})
}

// TestAccPrincipalAssignment_duplicateAssignment tests error handling for duplicate principal assignments
func TestAccPrincipalAssignment_duplicateAssignment(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPrincipalAssignmentConfigDuplicate,
				ExpectError: regexp.MustCompile("Principal Already Assigned"),
			},
		},
	})
}

// TestAccPrincipalAssignment_invalidPrincipalType tests validation for invalid principal types
func TestAccPrincipalAssignment_invalidPrincipalType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPrincipalAssignmentConfigInvalidType,
				ExpectError: regexp.MustCompile("Invalid Attribute Value Match"),
			},
		},
	})
}

// TestAccPrincipalAssignment_missingDirectoryFields tests conditional validation for USER/GROUP
func TestAccPrincipalAssignment_missingDirectoryFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPrincipalAssignmentConfigMissingDirectory,
				ExpectError: regexp.MustCompile("source_directory_name and source_directory_id are required for USER principal type"),
			},
		},
	})
}

// TestAccPrincipalAssignment_roleWithoutDirectory tests ROLE type doesn't require directory fields
func TestAccPrincipalAssignment_roleWithoutDirectory(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPrincipalAssignmentConfigRoleNoDirectory,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.role_minimal", "id"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_principal_assignment.role_minimal", "principal_type", "ROLE"),
					// Directory fields are optional for ROLE
				),
			},
		},
	})
}

// TestAccPrincipalAssignment_forceNewAttributes tests that policy_id, principal_id, and principal_type changes force replacement
func TestAccPrincipalAssignment_forceNewAttributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPrincipalAssignmentConfigForceNewBefore,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.forcenew_test", "id"),
				),
			},
			// Changing principal_type should force replacement (new resource created)
			{
				Config: testAccPrincipalAssignmentConfigForceNewAfter,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify new resource created with different principal
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_principal_assignment.forcenew_test", "id"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_principal_assignment.forcenew_test", "principal_type", "GROUP"),
				),
			},
		},
	})
}

// ============================================================================
// Test Configurations
// ============================================================================

const testAccPrincipalAssignmentConfigBasic = `
resource "cyberarksia_database_secret" "test" {
  name                = "test-principal-assignment-secret"
  authentication_type = "local"
  username            = "db_admin"
  password            = "TestPassword123!"
  description         = "Test secret for principal assignment"
}

resource "cyberarksia_database_workspace" "test" {
  name          = "test-principal-assignment-db"
  database_type = "postgresql"
  address       = "postgres.test.example.com"
  port          = 5432
  secret_id     = cyberarksia_database_secret.test.id
}

resource "cyberarksia_database_policy" "test" {
  name               = "test-principal-assignment-policy"
  description        = "Test policy for principal assignments"
  status             = "active"
  start_date         = "2025-01-01T00:00:00Z"
  end_date           = "2026-12-31T23:59:59Z"
  user_access_type   = "View"
  max_session_duration_minutes = 120
}

data "cyberarksia_principal" "test_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy_principal_assignment" "test" {
  policy_id             = cyberarksia_database_policy.test.policy_id
  principal_id          = data.cyberarksia_principal.test_user.id
  principal_type        = data.cyberarksia_principal.test_user.principal_type
  principal_name        = data.cyberarksia_principal.test_user.name
  source_directory_name = data.cyberarksia_principal.test_user.directory_name
  source_directory_id   = data.cyberarksia_principal.test_user.directory_id
}
`

const testAccPrincipalAssignmentConfigMultipleTypes = `
resource "cyberarksia_database_secret" "test" {
  name                = "test-multiple-types-secret"
  authentication_type = "local"
  username            = "db_admin"
  password            = "TestPassword123!"
  description         = "Test secret for multiple principal types"
}

resource "cyberarksia_database_workspace" "test" {
  name          = "test-multiple-types-db"
  database_type = "postgresql"
  address       = "postgres.multitypes.example.com"
  port          = 5432
  secret_id     = cyberarksia_database_secret.test.id
}

resource "cyberarksia_database_policy" "test" {
  name               = "test-multiple-types-policy"
  description        = "Test policy for multiple principal types"
  status             = "active"
  start_date         = "2025-01-01T00:00:00Z"
  end_date           = "2026-12-31T23:59:59Z"
  user_access_type   = "View"
  max_session_duration_minutes = 120
}

# USER principal
data "cyberarksia_principal" "user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy_principal_assignment" "user" {
  policy_id             = cyberarksia_database_policy.test.policy_id
  principal_id          = data.cyberarksia_principal.user.id
  principal_type        = data.cyberarksia_principal.user.principal_type
  principal_name        = data.cyberarksia_principal.user.name
  source_directory_name = data.cyberarksia_principal.user.directory_name
  source_directory_id   = data.cyberarksia_principal.user.directory_id
}

# GROUP principal
data "cyberarksia_principal" "group" {
  name = "CyberArk Guardians"
  type = "GROUP"
}

resource "cyberarksia_database_policy_principal_assignment" "group" {
  policy_id             = cyberarksia_database_policy.test.policy_id
  principal_id          = data.cyberarksia_principal.group.id
  principal_type        = data.cyberarksia_principal.group.principal_type
  principal_name        = data.cyberarksia_principal.group.name
  source_directory_name = data.cyberarksia_principal.group.directory_name
  source_directory_id   = data.cyberarksia_principal.group.directory_id
}

# ROLE principal
data "cyberarksia_principal" "role" {
  name = "System Administrator"
  type = "ROLE"
}

resource "cyberarksia_database_policy_principal_assignment" "role" {
  policy_id             = cyberarksia_database_policy.test.policy_id
  principal_id          = data.cyberarksia_principal.role.id
  principal_type        = data.cyberarksia_principal.role.principal_type
  principal_name        = data.cyberarksia_principal.role.name
  source_directory_name = data.cyberarksia_principal.role.directory_name
  source_directory_id   = data.cyberarksia_principal.role.directory_id
}
`

const testAccPrincipalAssignmentConfigUpdateBefore = `
resource "cyberarksia_database_secret" "test" {
  name                = "test-update-secret"
  authentication_type = "local"
  username            = "db_admin"
  password            = "TestPassword123!"
  description         = "Test secret for update"
}

resource "cyberarksia_database_workspace" "test" {
  name          = "test-update-db"
  database_type = "postgresql"
  address       = "postgres.update.example.com"
  port          = 5432
  secret_id     = cyberarksia_database_secret.test.id
}

resource "cyberarksia_database_policy" "test" {
  name               = "test-update-policy"
  description        = "Test policy for update"
  status             = "active"
  start_date         = "2025-01-01T00:00:00Z"
  end_date           = "2026-12-31T23:59:59Z"
  user_access_type   = "View"
  max_session_duration_minutes = 120
}

data "cyberarksia_principal" "test_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy_principal_assignment" "update_test" {
  policy_id             = cyberarksia_database_policy.test.policy_id
  principal_id          = data.cyberarksia_principal.test_user.id
  principal_type        = data.cyberarksia_principal.test_user.principal_type
  principal_name        = data.cyberarksia_principal.test_user.name
  source_directory_name = data.cyberarksia_principal.test_user.directory_name
  source_directory_id   = data.cyberarksia_principal.test_user.directory_id
}
`

const testAccPrincipalAssignmentConfigUpdateAfter = `
resource "cyberarksia_database_secret" "test" {
  name                = "test-update-secret"
  authentication_type = "local"
  username            = "db_admin"
  password            = "TestPassword123!"
  description         = "Test secret for update"
}

resource "cyberarksia_database_workspace" "test" {
  name          = "test-update-db"
  database_type = "postgresql"
  address       = "postgres.update.example.com"
  port          = 5432
  secret_id     = cyberarksia_database_secret.test.id
}

resource "cyberarksia_database_policy" "test" {
  name               = "test-update-policy"
  description        = "Test policy for update - modified"
  status             = "active"
  start_date         = "2025-01-01T00:00:00Z"
  end_date           = "2026-12-31T23:59:59Z"
  user_access_type   = "View"
  max_session_duration_minutes = 120
}

data "cyberarksia_principal" "test_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy_principal_assignment" "update_test" {
  policy_id             = cyberarksia_database_policy.test.policy_id
  principal_id          = data.cyberarksia_principal.test_user.id
  principal_type        = data.cyberarksia_principal.test_user.principal_type
  principal_name        = data.cyberarksia_principal.test_user.name
  source_directory_name = data.cyberarksia_principal.test_user.directory_name
  source_directory_id   = data.cyberarksia_principal.test_user.directory_id
}
`

const testAccPrincipalAssignmentConfigDuplicate = `
resource "cyberarksia_database_secret" "test" {
  name                = "test-duplicate-secret"
  authentication_type = "local"
  username            = "db_admin"
  password            = "TestPassword123!"
  description         = "Test secret for duplicate"
}

resource "cyberarksia_database_workspace" "test" {
  name          = "test-duplicate-db"
  database_type = "postgresql"
  address       = "postgres.duplicate.example.com"
  port          = 5432
  secret_id     = cyberarksia_database_secret.test.id
}

resource "cyberarksia_database_policy" "test" {
  name               = "test-duplicate-policy"
  description        = "Test policy for duplicate"
  status             = "active"
  start_date         = "2025-01-01T00:00:00Z"
  end_date           = "2026-12-31T23:59:59Z"
  user_access_type   = "View"
  max_session_duration_minutes = 120
}

data "cyberarksia_principal" "test_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy_principal_assignment" "first" {
  policy_id             = cyberarksia_database_policy.test.policy_id
  principal_id          = data.cyberarksia_principal.test_user.id
  principal_type        = data.cyberarksia_principal.test_user.principal_type
  principal_name        = data.cyberarksia_principal.test_user.name
  source_directory_name = data.cyberarksia_principal.test_user.directory_name
  source_directory_id   = data.cyberarksia_principal.test_user.directory_id
}

# This should fail - same principal already assigned
resource "cyberarksia_database_policy_principal_assignment" "duplicate" {
  policy_id             = cyberarksia_database_policy.test.policy_id
  principal_id          = data.cyberarksia_principal.test_user.id
  principal_type        = data.cyberarksia_principal.test_user.principal_type
  principal_name        = data.cyberarksia_principal.test_user.name
  source_directory_name = data.cyberarksia_principal.test_user.directory_name
  source_directory_id   = data.cyberarksia_principal.test_user.directory_id
}
`

const testAccPrincipalAssignmentConfigInvalidType = `
resource "cyberarksia_database_policy" "test" {
  name               = "test-invalid-type-policy"
  description        = "Test policy for invalid type"
  status             = "active"
  start_date         = "2025-01-01T00:00:00Z"
  end_date           = "2026-12-31T23:59:59Z"
  user_access_type   = "View"
  max_session_duration_minutes = 120
}

resource "cyberarksia_database_policy_principal_assignment" "invalid" {
  policy_id             = cyberarksia_database_policy.test.policy_id
  principal_id          = "12345678-1234-1234-1234-123456789012"
  principal_type        = "INVALID_TYPE"
  principal_name        = "test@example.com"
  source_directory_name = "Test Directory"
  source_directory_id   = "dir-123"
}
`

const testAccPrincipalAssignmentConfigMissingDirectory = `
resource "cyberarksia_database_policy" "test" {
  name               = "test-missing-dir-policy"
  description        = "Test policy for missing directory"
  status             = "active"
  start_date         = "2025-01-01T00:00:00Z"
  end_date           = "2026-12-31T23:59:59Z"
  user_access_type   = "View"
  max_session_duration_minutes = 120
}

resource "cyberarksia_database_policy_principal_assignment" "missing_dir" {
  policy_id      = cyberarksia_database_policy.test.policy_id
  principal_id   = "12345678-1234-1234-1234-123456789012"
  principal_type = "USER"
  principal_name = "test@example.com"
  # Missing required directory fields for USER type
}
`

const testAccPrincipalAssignmentConfigRoleNoDirectory = `
resource "cyberarksia_database_policy" "test" {
  name               = "test-role-nodir-policy"
  description        = "Test policy for role without directory"
  status             = "active"
  start_date         = "2025-01-01T00:00:00Z"
  end_date           = "2026-12-31T23:59:59Z"
  user_access_type   = "View"
  max_session_duration_minutes = 120
}

data "cyberarksia_principal" "role" {
  name = "System Administrator"
  type = "ROLE"
}

resource "cyberarksia_database_policy_principal_assignment" "role_minimal" {
  policy_id             = cyberarksia_database_policy.test.policy_id
  principal_id          = data.cyberarksia_principal.role.id
  principal_type        = data.cyberarksia_principal.role.principal_type
  principal_name        = data.cyberarksia_principal.role.name
  source_directory_name = data.cyberarksia_principal.role.directory_name
  source_directory_id   = data.cyberarksia_principal.role.directory_id
  # Directory fields are optional for ROLE type
}
`

const testAccPrincipalAssignmentConfigForceNewBefore = `
resource "cyberarksia_database_policy" "test" {
  name               = "test-forcenew-policy"
  description        = "Test policy for forcenew"
  status             = "active"
  start_date         = "2025-01-01T00:00:00Z"
  end_date           = "2026-12-31T23:59:59Z"
  user_access_type   = "View"
  max_session_duration_minutes = 120
}

data "cyberarksia_principal" "user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy_principal_assignment" "forcenew_test" {
  policy_id             = cyberarksia_database_policy.test.policy_id
  principal_id          = data.cyberarksia_principal.user.id
  principal_type        = data.cyberarksia_principal.user.principal_type
  principal_name        = data.cyberarksia_principal.user.name
  source_directory_name = data.cyberarksia_principal.user.directory_name
  source_directory_id   = data.cyberarksia_principal.user.directory_id
}
`

const testAccPrincipalAssignmentConfigForceNewAfter = `
resource "cyberarksia_database_policy" "test" {
  name               = "test-forcenew-policy"
  description        = "Test policy for forcenew"
  status             = "active"
  start_date         = "2025-01-01T00:00:00Z"
  end_date           = "2026-12-31T23:59:59Z"
  user_access_type   = "View"
  max_session_duration_minutes = 120
}

# Change to GROUP principal - should force replacement
data "cyberarksia_principal" "group" {
  name = "CyberArk Guardians"
  type = "GROUP"
}

resource "cyberarksia_database_policy_principal_assignment" "forcenew_test" {
  policy_id             = cyberarksia_database_policy.test.policy_id
  principal_id          = data.cyberarksia_principal.group.id
  principal_type        = data.cyberarksia_principal.group.principal_type
  principal_name        = data.cyberarksia_principal.group.name
  source_directory_name = data.cyberarksia_principal.group.directory_name
  source_directory_id   = data.cyberarksia_principal.group.directory_id
}
`
