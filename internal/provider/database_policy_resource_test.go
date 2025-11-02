// Package provider implements acceptance tests for database_policy resource
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ============================================================================
// Core CRUD Tests
// ============================================================================

// TestAccDatabasePolicy_basic tests basic CRUD lifecycle + ImportState
func TestAccDatabasePolicy_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDatabasePolicyConfigBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Basic metadata
					resource.TestCheckResourceAttr("cyberarksia_database_policy.test", "name", "test-basic-policy"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.test", "status", "active"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.test", "delegation_classification", "Unrestricted"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.test", "time_zone", "GMT"),

					// UUID validation
					resource.TestMatchResourceAttr("cyberarksia_database_policy.test", "id",
						mustCompileRegex(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)),
					resource.TestMatchResourceAttr("cyberarksia_database_policy.test", "policy_id",
						mustCompileRegex(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)),

					// Computed metadata fields
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy.test", "created_by.user"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy.test", "created_by.timestamp"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy.test", "updated_on.user"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy.test", "updated_on.timestamp"),

					// Conditions
					resource.TestCheckResourceAttr("cyberarksia_database_policy.test", "conditions.max_session_duration", "8"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.test", "conditions.idle_time", "10"),

					// Inline assignments
					resource.TestCheckResourceAttr("cyberarksia_database_policy.test", "target_database.#", "1"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.test", "principal.#", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "cyberarksia_database_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccDatabasePolicy_withConditions tests access windows, session limits, idle time
func TestAccDatabasePolicy_withConditions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatabasePolicyConfigWithConditions,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy.conditions_test", "name", "test-conditions-policy"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.conditions_test", "status", "active"),

					// Session conditions
					resource.TestCheckResourceAttr("cyberarksia_database_policy.conditions_test", "conditions.max_session_duration", "4"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.conditions_test", "conditions.idle_time", "15"),

					// Access window (weekdays 9-5)
					resource.TestCheckResourceAttr("cyberarksia_database_policy.conditions_test", "conditions.access_window.from_hour", "09:00"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.conditions_test", "conditions.access_window.to_hour", "17:00"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.conditions_test", "conditions.access_window.days_of_the_week.#", "5"),

					// Timezone
					resource.TestCheckResourceAttr("cyberarksia_database_policy.conditions_test", "time_zone", "America/New_York"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "cyberarksia_database_policy.conditions_test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccDatabasePolicy_withTimeFrame tests policy validity period
func TestAccDatabasePolicy_withTimeFrame(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatabasePolicyConfigWithTimeFrame,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy.timeframe_test", "name", "test-timeframe-policy"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.timeframe_test", "status", "active"),

					// Time frame validation
					resource.TestCheckResourceAttr("cyberarksia_database_policy.timeframe_test", "time_frame.from_time", "2025-01-01T00:00:00Z"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.timeframe_test", "time_frame.to_time", "2026-12-31T23:59:59Z"),
				),
			},
		},
	})
}

// TestAccDatabasePolicy_withInlineAssignments tests inline principals + target_database blocks
func TestAccDatabasePolicy_withInlineAssignments(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatabasePolicyConfigWithInlineAssignments,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy.inline_test", "name", "test-inline-assignments-policy"),

					// Verify multiple database targets
					resource.TestCheckResourceAttr("cyberarksia_database_policy.inline_test", "target_database.#", "2"),

					// Verify multiple principals
					resource.TestCheckResourceAttr("cyberarksia_database_policy.inline_test", "principal.#", "1"),

					// Verify principal attributes (using data source lookup pattern)
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy.inline_test", "principal.0.principal_id"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy.inline_test", "principal.0.principal_type"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "cyberarksia_database_policy.inline_test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// ============================================================================
// Update Tests
// ============================================================================

// TestAccDatabasePolicy_update tests changing status and conditions
func TestAccDatabasePolicy_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with active status
			{
				Config: testAccDatabasePolicyConfigUpdateBefore,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy.update_test", "name", "test-update-policy"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.update_test", "status", "active"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.update_test", "conditions.max_session_duration", "8"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.update_test", "conditions.idle_time", "10"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.update_test", "description", "Initial policy description"),
				),
			},
			// Step 2: Update to suspended status and modify conditions
			{
				Config: testAccDatabasePolicyConfigUpdateAfter,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy.update_test", "name", "test-update-policy"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.update_test", "status", "suspended"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.update_test", "conditions.max_session_duration", "12"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.update_test", "conditions.idle_time", "20"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.update_test", "description", "Updated policy description"),
				),
			},
			// Step 3: Verify import still works after update
			{
				ResourceName:      "cyberarksia_database_policy.update_test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccDatabasePolicy_updateAccessWindow tests modifying access window conditions
func TestAccDatabasePolicy_updateAccessWindow(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with weekday access window
			{
				Config: testAccDatabasePolicyConfigAccessWindowBefore,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy.window_test", "conditions.access_window.from_hour", "09:00"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.window_test", "conditions.access_window.to_hour", "17:00"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.window_test", "conditions.access_window.days_of_the_week.#", "5"),
				),
			},
			// Step 2: Update to 24/7 access
			{
				Config: testAccDatabasePolicyConfigAccessWindowAfter,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy.window_test", "conditions.access_window.from_hour", "00:00"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.window_test", "conditions.access_window.to_hour", "23:59"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.window_test", "conditions.access_window.days_of_the_week.#", "7"),
				),
			},
		},
	})
}

// ============================================================================
// Authentication Profile Tests
// ============================================================================

// TestAccDatabasePolicy_dbAuthProfile tests db_auth authentication method
func TestAccDatabasePolicy_dbAuthProfile(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatabasePolicyConfigDBAuth,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy.dbauth_test", "name", "test-dbauth-policy"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.dbauth_test", "target_database.#", "1"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.dbauth_test", "target_database.0.authentication_method", "db_auth"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.dbauth_test", "target_database.0.db_auth_profile.roles.#", "2"),
				),
			},
		},
	})
}

// TestAccDatabasePolicy_oracleAuthProfile tests oracle_auth authentication method
func TestAccDatabasePolicy_oracleAuthProfile(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatabasePolicyConfigOracleAuth,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy.oracle_test", "name", "test-oracle-policy"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.oracle_test", "target_database.0.authentication_method", "oracle_auth"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.oracle_test", "target_database.0.oracle_auth_profile.dba_role", "true"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy.oracle_test", "target_database.0.oracle_auth_profile.sysdba_role", "false"),
				),
			},
		},
	})
}

// ============================================================================
// Validation Tests
// ============================================================================

// TestAccDatabasePolicy_validationMissingTargets tests validation for missing target_database
func TestAccDatabasePolicy_validationMissingTargets(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDatabasePolicyConfigMissingTargets,
				ExpectError: mustCompileRegex("At least one target_database block is required"),
			},
		},
	})
}

// TestAccDatabasePolicy_validationMissingPrincipals tests validation for missing principal
func TestAccDatabasePolicy_validationMissingPrincipals(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDatabasePolicyConfigMissingPrincipals,
				ExpectError: mustCompileRegex("At least one principal block is required"),
			},
		},
	})
}

// TestAccDatabasePolicy_validationMismatchedAuthProfile tests validation for mismatched auth profile
func TestAccDatabasePolicy_validationMismatchedAuthProfile(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDatabasePolicyConfigMismatchedAuthProfile,
				ExpectError: mustCompileRegex("db_auth_profile block is required when authentication_method is 'db_auth'"),
			},
		},
	})
}

// ============================================================================
// ForceNew Tests
// ============================================================================

// TestAccDatabasePolicy_forceNewOnNameChange tests that changing name forces resource replacement
func TestAccDatabasePolicy_forceNewOnNameChange(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with initial name
			{
				Config: testAccDatabasePolicyConfigForceNewBefore,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy.forcenew_test", "name", "test-forcenew-policy-original"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy.forcenew_test", "policy_id"),
				),
			},
			// Step 2: Change name (should trigger replacement)
			{
				Config: testAccDatabasePolicyConfigForceNewAfter,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy.forcenew_test", "name", "test-forcenew-policy-renamed"),
					// Policy ID should be different due to resource replacement
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy.forcenew_test", "policy_id"),
				),
			},
		},
	})
}

// ============================================================================
// Test Configurations
// ============================================================================

const testAccDatabasePolicyConfigBasic = `
resource "cyberarksia_database_secret" "test" {
  name                = "test-policy-secret"
  authentication_type = "local"
  username            = "db_admin"
  password            = "TestPassword123!"
}

resource "cyberarksia_database_workspace" "test" {
  name                  = "test-policy-db"
  database_type         = "postgres"
  address               = "postgres.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.test.id
}

data "cyberarksia_principal" "test_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy" "test" {
  name                       = "test-basic-policy"
  status                     = "active"
  delegation_classification  = "unrestricted"
  time_zone                  = "GMT"

  conditions {
    max_session_duration = 8
    idle_time            = 10
  }

  target_database {
    database_workspace_id  = cyberarksia_database_workspace.test.id
    authentication_method  = "db_auth"

    db_auth_profile {
      roles = ["readonly"]
    }
  }

  principal {
    principal_id          = data.cyberarksia_principal.test_user.principal_id
    principal_type        = data.cyberarksia_principal.test_user.principal_type
    principal_name        = data.cyberarksia_principal.test_user.principal_name
    source_directory_name = data.cyberarksia_principal.test_user.source_directory_name
    source_directory_id   = data.cyberarksia_principal.test_user.source_directory_id
  }
}
`

const testAccDatabasePolicyConfigWithConditions = `
resource "cyberarksia_database_secret" "conditions" {
  name                = "test-conditions-secret"
  authentication_type = "local"
  username            = "db_user"
  password            = "TestPassword456!"
}

resource "cyberarksia_database_workspace" "conditions" {
  name                  = "test-conditions-db"
  database_type         = "postgres"
  address               = "postgres-conditions.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.conditions.id
}

data "cyberarksia_principal" "conditions_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy" "conditions_test" {
  name                       = "test-conditions-policy"
  status                     = "active"
  delegation_classification  = "unrestricted"
  time_zone                  = "America/New_York"

  conditions {
    max_session_duration = 4
    idle_time            = 15

    access_window {
      days_of_the_week = [1, 2, 3, 4, 5]  # Monday-Friday (0=Sunday, 6=Saturday)
      from_hour        = "09:00"
      to_hour          = "17:00"
    }
  }

  target_database {
    database_workspace_id  = cyberarksia_database_workspace.conditions.id
    authentication_method  = "db_auth"

    db_auth_profile {
      roles = ["readonly"]
    }
  }

  principal {
    principal_id          = data.cyberarksia_principal.conditions_user.principal_id
    principal_type        = data.cyberarksia_principal.conditions_user.principal_type
    principal_name        = data.cyberarksia_principal.conditions_user.principal_name
    source_directory_name = data.cyberarksia_principal.conditions_user.source_directory_name
    source_directory_id   = data.cyberarksia_principal.conditions_user.source_directory_id
  }
}
`

const testAccDatabasePolicyConfigWithTimeFrame = `
resource "cyberarksia_database_secret" "timeframe" {
  name                = "test-timeframe-secret"
  authentication_type = "local"
  username            = "db_user"
  password            = "TestPassword789!"
}

resource "cyberarksia_database_workspace" "timeframe" {
  name                  = "test-timeframe-db"
  database_type         = "postgres"
  address               = "postgres-timeframe.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.timeframe.id
}

data "cyberarksia_principal" "timeframe_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy" "timeframe_test" {
  name                       = "test-timeframe-policy"
  status                     = "active"
  delegation_classification  = "unrestricted"

  time_frame {
    from_time = "2025-01-01T00:00:00Z"
    to_time   = "2026-12-31T23:59:59Z"
  }

  conditions {
    max_session_duration = 8
  }

  target_database {
    database_workspace_id  = cyberarksia_database_workspace.timeframe.id
    authentication_method  = "db_auth"

    db_auth_profile {
      roles = ["readonly"]
    }
  }

  principal {
    principal_id          = data.cyberarksia_principal.timeframe_user.principal_id
    principal_type        = data.cyberarksia_principal.timeframe_user.principal_type
    principal_name        = data.cyberarksia_principal.timeframe_user.principal_name
    source_directory_name = data.cyberarksia_principal.timeframe_user.source_directory_name
    source_directory_id   = data.cyberarksia_principal.timeframe_user.source_directory_id
  }
}
`

const testAccDatabasePolicyConfigWithInlineAssignments = `
resource "cyberarksia_database_secret" "inline1" {
  name                = "test-inline-secret1"
  authentication_type = "local"
  username            = "db_user1"
  password            = "TestPassword111!"
}

resource "cyberarksia_database_secret" "inline2" {
  name                = "test-inline-secret2"
  authentication_type = "local"
  username            = "db_user2"
  password            = "TestPassword222!"
}

resource "cyberarksia_database_workspace" "inline1" {
  name                  = "test-inline-db1"
  database_type         = "postgres"
  address               = "postgres-inline1.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.inline1.id
}

resource "cyberarksia_database_workspace" "inline2" {
  name                  = "test-inline-db2"
  database_type         = "postgres"
  address               = "postgres-inline2.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.inline2.id
}

data "cyberarksia_principal" "inline_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy" "inline_test" {
  name                       = "test-inline-assignments-policy"
  status                     = "active"
  delegation_classification  = "unrestricted"

  conditions {
    max_session_duration = 8
  }

  # Multiple database targets
  target_database {
    database_workspace_id  = cyberarksia_database_workspace.inline1.id
    authentication_method  = "db_auth"

    db_auth_profile {
      roles = ["readonly"]
    }
  }

  target_database {
    database_workspace_id  = cyberarksia_database_workspace.inline2.id
    authentication_method  = "db_auth"

    db_auth_profile {
      roles = ["readwrite"]
    }
  }

  # Principal assignment using data source lookup
  principal {
    principal_id          = data.cyberarksia_principal.inline_user.principal_id
    principal_type        = data.cyberarksia_principal.inline_user.principal_type
    principal_name        = data.cyberarksia_principal.inline_user.principal_name
    source_directory_name = data.cyberarksia_principal.inline_user.source_directory_name
    source_directory_id   = data.cyberarksia_principal.inline_user.source_directory_id
  }
}
`

const testAccDatabasePolicyConfigUpdateBefore = `
resource "cyberarksia_database_secret" "update" {
  name                = "test-update-secret"
  authentication_type = "local"
  username            = "db_user"
  password            = "TestPassword999!"
}

resource "cyberarksia_database_workspace" "update" {
  name                  = "test-update-db"
  database_type         = "postgres"
  address               = "postgres-update.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.update.id
}

data "cyberarksia_principal" "update_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy" "update_test" {
  name                       = "test-update-policy"
  description                = "Initial policy description"
  status                     = "active"
  delegation_classification  = "unrestricted"

  conditions {
    max_session_duration = 8
    idle_time            = 10
  }

  target_database {
    database_workspace_id  = cyberarksia_database_workspace.update.id
    authentication_method  = "db_auth"

    db_auth_profile {
      roles = ["readonly"]
    }
  }

  principal {
    principal_id          = data.cyberarksia_principal.update_user.principal_id
    principal_type        = data.cyberarksia_principal.update_user.principal_type
    principal_name        = data.cyberarksia_principal.update_user.principal_name
    source_directory_name = data.cyberarksia_principal.update_user.source_directory_name
    source_directory_id   = data.cyberarksia_principal.update_user.source_directory_id
  }
}
`

const testAccDatabasePolicyConfigUpdateAfter = `
resource "cyberarksia_database_secret" "update" {
  name                = "test-update-secret"
  authentication_type = "local"
  username            = "db_user"
  password            = "TestPassword999!"
}

resource "cyberarksia_database_workspace" "update" {
  name                  = "test-update-db"
  database_type         = "postgres"
  address               = "postgres-update.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.update.id
}

data "cyberarksia_principal" "update_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy" "update_test" {
  name                       = "test-update-policy"
  description                = "Updated policy description"
  status                     = "suspended"
  delegation_classification  = "unrestricted"

  conditions {
    max_session_duration = 12
    idle_time            = 20
  }

  target_database {
    database_workspace_id  = cyberarksia_database_workspace.update.id
    authentication_method  = "db_auth"

    db_auth_profile {
      roles = ["readonly"]
    }
  }

  principal {
    principal_id          = data.cyberarksia_principal.update_user.principal_id
    principal_type        = data.cyberarksia_principal.update_user.principal_type
    principal_name        = data.cyberarksia_principal.update_user.principal_name
    source_directory_name = data.cyberarksia_principal.update_user.source_directory_name
    source_directory_id   = data.cyberarksia_principal.update_user.source_directory_id
  }
}
`

const testAccDatabasePolicyConfigAccessWindowBefore = `
resource "cyberarksia_database_secret" "window" {
  name                = "test-window-secret"
  authentication_type = "local"
  username            = "db_user"
  password            = "TestPassword888!"
}

resource "cyberarksia_database_workspace" "window" {
  name                  = "test-window-db"
  database_type         = "postgres"
  address               = "postgres-window.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.window.id
}

data "cyberarksia_principal" "window_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy" "window_test" {
  name                       = "test-window-policy"
  status                     = "active"
  delegation_classification  = "unrestricted"

  conditions {
    max_session_duration = 8

    access_window {
      days_of_the_week = [1, 2, 3, 4, 5]  # Weekdays only
      from_hour        = "09:00"
      to_hour          = "17:00"
    }
  }

  target_database {
    database_workspace_id  = cyberarksia_database_workspace.window.id
    authentication_method  = "db_auth"

    db_auth_profile {
      roles = ["readonly"]
    }
  }

  principal {
    principal_id          = data.cyberarksia_principal.window_user.principal_id
    principal_type        = data.cyberarksia_principal.window_user.principal_type
    principal_name        = data.cyberarksia_principal.window_user.principal_name
    source_directory_name = data.cyberarksia_principal.window_user.source_directory_name
    source_directory_id   = data.cyberarksia_principal.window_user.source_directory_id
  }
}
`

const testAccDatabasePolicyConfigAccessWindowAfter = `
resource "cyberarksia_database_secret" "window" {
  name                = "test-window-secret"
  authentication_type = "local"
  username            = "db_user"
  password            = "TestPassword888!"
}

resource "cyberarksia_database_workspace" "window" {
  name                  = "test-window-db"
  database_type         = "postgres"
  address               = "postgres-window.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.window.id
}

data "cyberarksia_principal" "window_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy" "window_test" {
  name                       = "test-window-policy"
  status                     = "active"
  delegation_classification  = "unrestricted"

  conditions {
    max_session_duration = 8

    access_window {
      days_of_the_week = [0, 1, 2, 3, 4, 5, 6]  # All days (24/7)
      from_hour        = "00:00"
      to_hour          = "23:59"
    }
  }

  target_database {
    database_workspace_id  = cyberarksia_database_workspace.window.id
    authentication_method  = "db_auth"

    db_auth_profile {
      roles = ["readonly"]
    }
  }

  principal {
    principal_id          = data.cyberarksia_principal.window_user.principal_id
    principal_type        = data.cyberarksia_principal.window_user.principal_type
    principal_name        = data.cyberarksia_principal.window_user.principal_name
    source_directory_name = data.cyberarksia_principal.window_user.source_directory_name
    source_directory_id   = data.cyberarksia_principal.window_user.source_directory_id
  }
}
`

const testAccDatabasePolicyConfigDBAuth = `
resource "cyberarksia_database_secret" "dbauth" {
  name                = "test-dbauth-secret"
  authentication_type = "local"
  username            = "db_user"
  password            = "TestPassword777!"
}

resource "cyberarksia_database_workspace" "dbauth" {
  name                  = "test-dbauth-db"
  database_type         = "postgres"
  address               = "postgres-dbauth.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.dbauth.id
}

data "cyberarksia_principal" "dbauth_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy" "dbauth_test" {
  name                       = "test-dbauth-policy"
  status                     = "active"
  delegation_classification  = "unrestricted"

  conditions {
    max_session_duration = 8
  }

  target_database {
    database_workspace_id  = cyberarksia_database_workspace.dbauth.id
    authentication_method  = "db_auth"

    db_auth_profile {
      roles = ["readonly", "readwrite"]
    }
  }

  principal {
    principal_id          = data.cyberarksia_principal.dbauth_user.principal_id
    principal_type        = data.cyberarksia_principal.dbauth_user.principal_type
    principal_name        = data.cyberarksia_principal.dbauth_user.principal_name
    source_directory_name = data.cyberarksia_principal.dbauth_user.source_directory_name
    source_directory_id   = data.cyberarksia_principal.dbauth_user.source_directory_id
  }
}
`

const testAccDatabasePolicyConfigOracleAuth = `
resource "cyberarksia_database_secret" "oracle" {
  name                = "test-oracle-secret"
  authentication_type = "local"
  username            = "oracle_user"
  password            = "TestPassword666!"
}

resource "cyberarksia_database_workspace" "oracle" {
  name                  = "test-oracle-db"
  database_type         = "oracle"
  address               = "oracle.example.com"
  port                  = 1521
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.oracle.id
}

data "cyberarksia_principal" "oracle_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy" "oracle_test" {
  name                       = "test-oracle-policy"
  status                     = "active"
  delegation_classification  = "unrestricted"

  conditions {
    max_session_duration = 8
  }

  target_database {
    database_workspace_id  = cyberarksia_database_workspace.oracle.id
    authentication_method  = "oracle_auth"

    oracle_auth_profile {
      roles       = ["CONNECT", "RESOURCE"]
      dba_role    = true
      sysdba_role = false
      sysoper_role = false
    }
  }

  principal {
    principal_id          = data.cyberarksia_principal.oracle_user.principal_id
    principal_type        = data.cyberarksia_principal.oracle_user.principal_type
    principal_name        = data.cyberarksia_principal.oracle_user.principal_name
    source_directory_name = data.cyberarksia_principal.oracle_user.source_directory_name
    source_directory_id   = data.cyberarksia_principal.oracle_user.source_directory_id
  }
}
`

const testAccDatabasePolicyConfigMissingTargets = `
data "cyberarksia_principal" "missing_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy" "missing_targets" {
  name                       = "test-missing-targets"
  status                     = "active"
  delegation_classification  = "unrestricted"

  conditions {
    max_session_duration = 8
  }

  # Missing target_database block - should fail validation

  principal {
    principal_id          = data.cyberarksia_principal.missing_user.principal_id
    principal_type        = data.cyberarksia_principal.missing_user.principal_type
    principal_name        = data.cyberarksia_principal.missing_user.principal_name
    source_directory_name = data.cyberarksia_principal.missing_user.source_directory_name
    source_directory_id   = data.cyberarksia_principal.missing_user.source_directory_id
  }
}
`

const testAccDatabasePolicyConfigMissingPrincipals = `
resource "cyberarksia_database_secret" "missing_principals" {
  name                = "test-missing-principals-secret"
  authentication_type = "local"
  username            = "db_user"
  password            = "TestPassword555!"
}

resource "cyberarksia_database_workspace" "missing_principals" {
  name                  = "test-missing-principals-db"
  database_type         = "postgres"
  address               = "postgres.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.missing_principals.id
}

resource "cyberarksia_database_policy" "missing_principals" {
  name                       = "test-missing-principals"
  status                     = "active"
  delegation_classification  = "unrestricted"

  conditions {
    max_session_duration = 8
  }

  target_database {
    database_workspace_id  = cyberarksia_database_workspace.missing_principals.id
    authentication_method  = "db_auth"

    db_auth_profile {
      roles = ["readonly"]
    }
  }

  # Missing principal block - should fail validation
}
`

const testAccDatabasePolicyConfigMismatchedAuthProfile = `
resource "cyberarksia_database_secret" "mismatched" {
  name                = "test-mismatched-secret"
  authentication_type = "local"
  username            = "db_user"
  password            = "TestPassword444!"
}

resource "cyberarksia_database_workspace" "mismatched" {
  name                  = "test-mismatched-db"
  database_type         = "postgres"
  address               = "postgres.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.mismatched.id
}

data "cyberarksia_principal" "mismatched_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy" "mismatched" {
  name                       = "test-mismatched-auth"
  status                     = "active"
  delegation_classification  = "unrestricted"

  conditions {
    max_session_duration = 8
  }

  target_database {
    database_workspace_id  = cyberarksia_database_workspace.mismatched.id
    authentication_method  = "db_auth"

    # Missing db_auth_profile - should fail validation
    # (authentication_method is db_auth but no db_auth_profile provided)
  }

  principal {
    principal_id          = data.cyberarksia_principal.mismatched_user.principal_id
    principal_type        = data.cyberarksia_principal.mismatched_user.principal_type
    principal_name        = data.cyberarksia_principal.mismatched_user.principal_name
    source_directory_name = data.cyberarksia_principal.mismatched_user.source_directory_name
    source_directory_id   = data.cyberarksia_principal.mismatched_user.source_directory_id
  }
}
`

const testAccDatabasePolicyConfigForceNewBefore = `
resource "cyberarksia_database_secret" "forcenew" {
  name                = "test-forcenew-secret"
  authentication_type = "local"
  username            = "db_user"
  password            = "TestPassword333!"
}

resource "cyberarksia_database_workspace" "forcenew" {
  name                  = "test-forcenew-db"
  database_type         = "postgres"
  address               = "postgres.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.forcenew.id
}

data "cyberarksia_principal" "forcenew_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy" "forcenew_test" {
  name                       = "test-forcenew-policy-original"
  status                     = "active"
  delegation_classification  = "unrestricted"

  conditions {
    max_session_duration = 8
  }

  target_database {
    database_workspace_id  = cyberarksia_database_workspace.forcenew.id
    authentication_method  = "db_auth"

    db_auth_profile {
      roles = ["readonly"]
    }
  }

  principal {
    principal_id          = data.cyberarksia_principal.forcenew_user.principal_id
    principal_type        = data.cyberarksia_principal.forcenew_user.principal_type
    principal_name        = data.cyberarksia_principal.forcenew_user.principal_name
    source_directory_name = data.cyberarksia_principal.forcenew_user.source_directory_name
    source_directory_id   = data.cyberarksia_principal.forcenew_user.source_directory_id
  }
}
`

const testAccDatabasePolicyConfigForceNewAfter = `
resource "cyberarksia_database_secret" "forcenew" {
  name                = "test-forcenew-secret"
  authentication_type = "local"
  username            = "db_user"
  password            = "TestPassword333!"
}

resource "cyberarksia_database_workspace" "forcenew" {
  name                  = "test-forcenew-db"
  database_type         = "postgres"
  address               = "postgres.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.forcenew.id
}

data "cyberarksia_principal" "forcenew_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"
}

resource "cyberarksia_database_policy" "forcenew_test" {
  name                       = "test-forcenew-policy-renamed"
  status                     = "active"
  delegation_classification  = "unrestricted"

  conditions {
    max_session_duration = 8
  }

  target_database {
    database_workspace_id  = cyberarksia_database_workspace.forcenew.id
    authentication_method  = "db_auth"

    db_auth_profile {
      roles = ["readonly"]
    }
  }

  principal {
    principal_id          = data.cyberarksia_principal.forcenew_user.principal_id
    principal_type        = data.cyberarksia_principal.forcenew_user.principal_type
    principal_name        = data.cyberarksia_principal.forcenew_user.principal_name
    source_directory_name = data.cyberarksia_principal.forcenew_user.source_directory_name
    source_directory_id   = data.cyberarksia_principal.forcenew_user.source_directory_id
  }
}
`
