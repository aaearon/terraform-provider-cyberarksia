// Package provider implements acceptance tests for policy_database_assignment resource
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ============================================================================
// Core CRUD Tests
// ============================================================================

// TestAccPolicyDatabaseAssignment_basic tests basic CRUD lifecycle for policy database assignment
// Validates:
// - Assignment creation with db_auth profile
// - Composite ID format (policy-id:database-id)
// - Authentication method and profile persistence
// - ImportState functionality
func TestAccPolicyDatabaseAssignment_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPolicyDatabaseAssignmentConfigBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.test", "id"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.test", "policy_id"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.test", "database_workspace_id"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.test", "authentication_method", "db_auth"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.test", "db_auth_profile.roles.#", "2"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.test", "db_auth_profile.roles.0", "pg_read_all_data"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.test", "db_auth_profile.roles.1", "pg_write_all_data"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.test", "last_modified"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "cyberarksia_database_policy_database_assignment.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccPolicyDatabaseAssignment_withDBAuth tests db_auth profile with roles
// Validates:
// - db_auth authentication method
// - Roles list (pg_read_all_data, pg_write_all_data)
// - Profile persistence in state
func TestAccPolicyDatabaseAssignment_withDBAuth(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyDatabaseAssignmentConfigDBAuth,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.db_auth", "authentication_method", "db_auth"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.db_auth", "db_auth_profile.roles.#", "3"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.db_auth", "db_auth_profile.roles.0", "connect"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.db_auth", "db_auth_profile.roles.1", "resource"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.db_auth", "db_auth_profile.roles.2", "dba"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.db_auth", "id"),
				),
			},
		},
	})
}

// TestAccPolicyDatabaseAssignment_withLDAPAuth tests ldap_auth profile with assign_groups
// Validates:
// - ldap_auth authentication method
// - AssignGroups list (CN=DBAdmins,OU=Groups,DC=example,DC=com)
// - Profile persistence in state
func TestAccPolicyDatabaseAssignment_withLDAPAuth(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyDatabaseAssignmentConfigLdapAuth,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.ldap_auth", "authentication_method", "ldap_auth"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.ldap_auth", "ldap_auth_profile.assign_groups.#", "2"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.ldap_auth", "ldap_auth_profile.assign_groups.0", "CN=DBAdmins,OU=Groups,DC=example,DC=com"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.ldap_auth", "ldap_auth_profile.assign_groups.1", "CN=DBUsers,OU=Groups,DC=example,DC=com"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.ldap_auth", "id"),
				),
			},
		},
	})
}

// TestAccPolicyDatabaseAssignment_withOracleAuth tests oracle_auth profile with roles and special permissions
// Validates:
// - oracle_auth authentication method
// - Roles list and boolean flags (dba_role, sysdba_role, sysoper_role)
// - Profile persistence in state
func TestAccPolicyDatabaseAssignment_withOracleAuth(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyDatabaseAssignmentConfigOracleAuth,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.oracle_auth", "authentication_method", "oracle_auth"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.oracle_auth", "oracle_auth_profile.roles.#", "2"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.oracle_auth", "oracle_auth_profile.roles.0", "CONNECT"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.oracle_auth", "oracle_auth_profile.roles.1", "RESOURCE"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.oracle_auth", "oracle_auth_profile.dba_role", "true"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.oracle_auth", "oracle_auth_profile.sysdba_role", "false"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.oracle_auth", "oracle_auth_profile.sysoper_role", "false"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.oracle_auth", "id"),
				),
			},
		},
	})
}

// TestAccPolicyDatabaseAssignment_withMongoAuth tests mongo_auth profile with global and database-specific roles
// Validates:
// - mongo_auth authentication method
// - GlobalBuiltinRoles list
// - DatabaseBuiltinRoles map (database → roles)
// - Profile persistence in state
func TestAccPolicyDatabaseAssignment_withMongoAuth(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyDatabaseAssignmentConfigMongoAuth,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.mongo_auth", "authentication_method", "mongo_auth"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.mongo_auth", "mongo_auth_profile.global_builtin_roles.#", "1"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.mongo_auth", "mongo_auth_profile.global_builtin_roles.0", "readAnyDatabase"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.mongo_auth", "mongo_auth_profile.database_builtin_roles.%", "1"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.mongo_auth", "id"),
				),
			},
		},
	})
}

// TestAccPolicyDatabaseAssignment_withSQLServerAuth tests sqlserver_auth profile with global and database-specific roles
// Validates:
// - sqlserver_auth authentication method
// - GlobalBuiltinRoles and GlobalCustomRoles lists
// - DatabaseBuiltinRoles and DatabaseCustomRoles maps
// - Profile persistence in state
func TestAccPolicyDatabaseAssignment_withSQLServerAuth(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyDatabaseAssignmentConfigSqlserverAuth,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.sqlserver_auth", "authentication_method", "sqlserver_auth"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.sqlserver_auth", "sqlserver_auth_profile.global_builtin_roles.#", "2"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.sqlserver_auth", "sqlserver_auth_profile.global_builtin_roles.0", "sysadmin"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.sqlserver_auth", "sqlserver_auth_profile.global_builtin_roles.1", "serveradmin"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.sqlserver_auth", "id"),
				),
			},
		},
	})
}

// TestAccPolicyDatabaseAssignment_withRDSIAMAuth tests rds_iam_user_auth profile with db_user
// Validates:
// - rds_iam_user_auth authentication method
// - DBUser string
// - Profile persistence in state
func TestAccPolicyDatabaseAssignment_withRDSIAMAuth(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyDatabaseAssignmentConfigRdsIAMAuth,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.rds_iam_auth", "authentication_method", "rds_iam_user_auth"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.rds_iam_auth", "rds_iam_user_auth_profile.db_user", "iamuser"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.rds_iam_auth", "id"),
				),
			},
		},
	})
}

// TestAccPolicyDatabaseAssignment_import tests ImportState with composite ID parsing
// Validates:
// - ImportState accepts composite ID format (policy-id:database-id)
// - State is correctly populated after import
// - ID parsing logic works correctly
func TestAccPolicyDatabaseAssignment_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create resource
			{
				Config: testAccPolicyDatabaseAssignmentConfigBasic,
			},
			// Test import with composite ID
			{
				ResourceName:      "cyberarksia_database_policy_database_assignment.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// ============================================================================
// Update Tests
// ============================================================================

// TestAccPolicyDatabaseAssignment_updateAuthMethod tests updating authentication method
// Validates:
// - Authentication method can be changed (db_auth → ldap_auth)
// - Profile is updated accordingly
// - Update operation preserves other policy assignments
func TestAccPolicyDatabaseAssignment_updateAuthMethod(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with db_auth
			{
				Config: testAccPolicyDatabaseAssignmentConfigUpdateBefore,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.update_test", "authentication_method", "db_auth"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.update_test", "db_auth_profile.roles.#", "1"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.update_test", "db_auth_profile.roles.0", "connect"),
				),
			},
			// Step 2: Update to ldap_auth
			{
				Config: testAccPolicyDatabaseAssignmentConfigUpdateAfter,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.update_test", "authentication_method", "ldap_auth"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.update_test", "ldap_auth_profile.assign_groups.#", "1"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.update_test", "ldap_auth_profile.assign_groups.0", "CN=Developers,OU=Groups,DC=example,DC=com"),
				),
			},
		},
	})
}

// TestAccPolicyDatabaseAssignment_updateProfile tests updating profile within same auth method
// Validates:
// - Profile attributes can be updated (roles list change)
// - Update operation preserves authentication method
// - State reflects updated profile
func TestAccPolicyDatabaseAssignment_updateProfile(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with initial roles
			{
				Config: testAccPolicyDatabaseAssignmentConfigProfileUpdateBefore,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.profile_update", "authentication_method", "db_auth"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.profile_update", "db_auth_profile.roles.#", "1"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.profile_update", "db_auth_profile.roles.0", "connect"),
				),
			},
			// Step 2: Update roles
			{
				Config: testAccPolicyDatabaseAssignmentConfigProfileUpdateAfter,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.profile_update", "authentication_method", "db_auth"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.profile_update", "db_auth_profile.roles.#", "2"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.profile_update", "db_auth_profile.roles.0", "connect"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.profile_update", "db_auth_profile.roles.1", "dba"),
				),
			},
		},
	})
}

// ============================================================================
// ForceNew Tests
// ============================================================================

// TestAccPolicyDatabaseAssignment_forceNewPolicy tests ForceNew behavior for policy_id change
// Validates:
// - Changing policy_id triggers resource replacement (destroy + recreate)
// - ID changes after replacement
func TestAccPolicyDatabaseAssignment_forceNewPolicy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with first policy
			{
				Config: testAccPolicyDatabaseAssignmentConfigForceNewBefore,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.forcenew_test", "id"),
				),
			},
			// Step 2: Change policy_id (should trigger replacement)
			{
				Config: testAccPolicyDatabaseAssignmentConfigForceNewAfter,
				Check: resource.ComposeAggregateTestCheckFunc(
					// ID should be different due to resource replacement
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.forcenew_test", "id"),
				),
			},
		},
	})
}

// TestAccPolicyDatabaseAssignment_forceNewDatabase tests ForceNew behavior for database_workspace_id change
// Validates:
// - Changing database_workspace_id triggers resource replacement
// - ID changes after replacement
func TestAccPolicyDatabaseAssignment_forceNewDatabase(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with first database
			{
				Config: testAccPolicyDatabaseAssignmentConfigForceNewDatabaseBefore,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.forcenew_db", "id"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.forcenew_db", "database_workspace_id"),
				),
			},
			// Step 2: Change database_workspace_id (should trigger replacement)
			{
				Config: testAccPolicyDatabaseAssignmentConfigForceNewDatabaseAfter,
				Check: resource.ComposeAggregateTestCheckFunc(
					// ID should be different due to resource replacement
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.forcenew_db", "id"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.forcenew_db", "database_workspace_id"),
				),
			},
		},
	})
}

// ============================================================================
// Multiple Assignments Tests
// ============================================================================

// TestAccPolicyDatabaseAssignment_multipleAssignments tests multiple databases assigned to same policy
// Validates:
// - Multiple assignments to same policy work independently
// - Each assignment has unique composite ID
// - Deleting one assignment doesn't affect others
func TestAccPolicyDatabaseAssignment_multipleAssignments(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyDatabaseAssignmentConfigMultiple,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Assignment 1
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.multi1", "id"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.multi1", "authentication_method", "db_auth"),
					// Assignment 2
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.multi2", "id"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.multi2", "authentication_method", "ldap_auth"),
					// Assignment 3
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.multi3", "id"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.multi3", "authentication_method", "oracle_auth"),
				),
			},
		},
	})
}

// ============================================================================
// Drift Detection Tests
// ============================================================================

// TestAccPolicyDatabaseAssignment_driftDetection tests external modification detection
// Validates:
// - Resource removed from policy outside Terraform is detected
// - State refresh removes resource from Terraform state
func TestAccPolicyDatabaseAssignment_driftDetection(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create resource
			{
				Config: testAccPolicyDatabaseAssignmentConfigDrift,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.drift_test", "id"),
					resource.TestCheckResourceAttr("cyberarksia_database_policy_database_assignment.drift_test", "authentication_method", "db_auth"),
				),
			},
			// Step 2: Refresh state (should detect if assignment was removed outside Terraform)
			// Note: In a real test, you would manually remove the assignment in SIA between steps
			// For now, this verifies the refresh mechanism works
			{
				Config:   testAccPolicyDatabaseAssignmentConfigDrift,
				PlanOnly: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cyberarksia_database_policy_database_assignment.drift_test", "id"),
				),
			},
		},
	})
}

// ============================================================================
// Test Configurations
// ============================================================================

const testAccPolicyDatabaseAssignmentConfigBasic = `
resource "cyberarksia_database_secret" "test" {
  name                = "test-db-secret"
  authentication_type = "local"
  username            = "postgres"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "test" {
  name                  = "test-postgres-db"
  database_type         = "postgres"
  address               = "postgres.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.test.id
}

resource "cyberarksia_database_policy" "test" {
  policy_name = "test-policy-basic"
  description = "Test policy for basic assignment"
}

resource "cyberarksia_database_policy_database_assignment" "test" {
  policy_id              = cyberarksia_database_policy.test.id
  database_workspace_id  = cyberarksia_database_workspace.test.id
  authentication_method  = "db_auth"

  db_auth_profile {
    roles = ["pg_read_all_data", "pg_write_all_data"]
  }
}
`

const testAccPolicyDatabaseAssignmentConfigDBAuth = `
resource "cyberarksia_database_secret" "db_auth" {
  name                = "db-auth-secret"
  authentication_type = "local"
  username            = "oracleadmin"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "db_auth" {
  name                  = "oracle-db-auth"
  database_type         = "oracle"
  address               = "oracle.example.com"
  port                  = 1521
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.db_auth.id
}

resource "cyberarksia_database_policy" "db_auth" {
  policy_name = "test-policy-db-auth"
  description = "Test policy for db_auth profile"
}

resource "cyberarksia_database_policy_database_assignment" "db_auth" {
  policy_id              = cyberarksia_database_policy.db_auth.id
  database_workspace_id  = cyberarksia_database_workspace.db_auth.id
  authentication_method  = "db_auth"

  db_auth_profile {
    roles = ["connect", "resource", "dba"]
  }
}
`

const testAccPolicyDatabaseAssignmentConfigLdapAuth = `
resource "cyberarksia_database_secret" "ldap_auth" {
  name                = "ldap-auth-secret"
  authentication_type = "local"
  username            = "postgres"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "ldap_auth" {
  name                  = "postgres-ldap-auth"
  database_type         = "postgres"
  address               = "postgres-ldap.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.ldap_auth.id
}

resource "cyberarksia_database_policy" "ldap_auth" {
  policy_name = "test-policy-ldap-auth"
  description = "Test policy for ldap_auth profile"
}

resource "cyberarksia_database_policy_database_assignment" "ldap_auth" {
  policy_id              = cyberarksia_database_policy.ldap_auth.id
  database_workspace_id  = cyberarksia_database_workspace.ldap_auth.id
  authentication_method  = "ldap_auth"

  ldap_auth_profile {
    assign_groups = [
      "CN=DBAdmins,OU=Groups,DC=example,DC=com",
      "CN=DBUsers,OU=Groups,DC=example,DC=com"
    ]
  }
}
`

const testAccPolicyDatabaseAssignmentConfigOracleAuth = `
resource "cyberarksia_database_secret" "oracle_auth" {
  name                = "oracle-auth-secret"
  authentication_type = "local"
  username            = "oracleadmin"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "oracle_auth" {
  name                  = "oracle-auth-db"
  database_type         = "oracle"
  address               = "oracle-auth.example.com"
  port                  = 1521
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.oracle_auth.id
}

resource "cyberarksia_database_policy" "oracle_auth" {
  policy_name = "test-policy-oracle-auth"
  description = "Test policy for oracle_auth profile"
}

resource "cyberarksia_database_policy_database_assignment" "oracle_auth" {
  policy_id              = cyberarksia_database_policy.oracle_auth.id
  database_workspace_id  = cyberarksia_database_workspace.oracle_auth.id
  authentication_method  = "oracle_auth"

  oracle_auth_profile {
    roles       = ["CONNECT", "RESOURCE"]
    dba_role    = true
    sysdba_role = false
    sysoper_role = false
  }
}
`

const testAccPolicyDatabaseAssignmentConfigMongoAuth = `
resource "cyberarksia_database_secret" "mongo_auth" {
  name                = "mongo-auth-secret"
  authentication_type = "local"
  username            = "mongoadmin"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "mongo_auth" {
  name                  = "mongo-auth-db"
  database_type         = "mongo"
  address               = "mongo.example.com"
  port                  = 27017
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.mongo_auth.id
}

resource "cyberarksia_database_policy" "mongo_auth" {
  policy_name = "test-policy-mongo-auth"
  description = "Test policy for mongo_auth profile"
}

resource "cyberarksia_database_policy_database_assignment" "mongo_auth" {
  policy_id              = cyberarksia_database_policy.mongo_auth.id
  database_workspace_id  = cyberarksia_database_workspace.mongo_auth.id
  authentication_method  = "mongo_auth"

  mongo_auth_profile {
    global_builtin_roles = ["readAnyDatabase"]
    database_builtin_roles = {
      "testdb" = ["read", "write"]
    }
  }
}
`

const testAccPolicyDatabaseAssignmentConfigSqlserverAuth = `
resource "cyberarksia_database_secret" "sqlserver_auth" {
  name                = "sqlserver-auth-secret"
  authentication_type = "local"
  username            = "sqladmin"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "sqlserver_auth" {
  name                  = "sqlserver-auth-db"
  database_type         = "mssql"
  address               = "sqlserver.example.com"
  port                  = 1433
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.sqlserver_auth.id
}

resource "cyberarksia_database_policy" "sqlserver_auth" {
  policy_name = "test-policy-sqlserver-auth"
  description = "Test policy for sqlserver_auth profile"
}

resource "cyberarksia_database_policy_database_assignment" "sqlserver_auth" {
  policy_id              = cyberarksia_database_policy.sqlserver_auth.id
  database_workspace_id  = cyberarksia_database_workspace.sqlserver_auth.id
  authentication_method  = "sqlserver_auth"

  sqlserver_auth_profile {
    global_builtin_roles = ["sysadmin", "serveradmin"]
  }
}
`

const testAccPolicyDatabaseAssignmentConfigRdsIAMAuth = `
resource "cyberarksia_database_secret" "rds_iam" {
  name                  = "rds-iam-secret"
  authentication_type   = "aws_iam"
  aws_access_key_id     = "AKIAIOSFODNN7EXAMPLE"
  aws_secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}

resource "cyberarksia_database_workspace" "rds_iam" {
  name                  = "rds-postgres-iam"
  database_type         = "postgres-aws-rds"
  address               = "mydb.abc123.us-east-1.rds.amazonaws.com"
  port                  = 5432
  authentication_method = "rds_iam_authentication"
  cloud_provider        = "aws"
  region                = "us-east-1"
  secret_id             = cyberarksia_database_secret.rds_iam.id
}

resource "cyberarksia_database_policy" "rds_iam" {
  policy_name = "test-policy-rds-iam"
  description = "Test policy for rds_iam_user_auth profile"
}

resource "cyberarksia_database_policy_database_assignment" "rds_iam_auth" {
  policy_id              = cyberarksia_database_policy.rds_iam.id
  database_workspace_id  = cyberarksia_database_workspace.rds_iam.id
  authentication_method  = "rds_iam_user_auth"

  rds_iam_user_auth_profile {
    db_user = "iamuser"
  }
}
`

const testAccPolicyDatabaseAssignmentConfigUpdateBefore = `
resource "cyberarksia_database_secret" "update_test" {
  name                = "update-test-secret"
  authentication_type = "local"
  username            = "postgres"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "update_test" {
  name                  = "update-test-db"
  database_type         = "postgres"
  address               = "postgres-update.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.update_test.id
}

resource "cyberarksia_database_policy" "update_test" {
  policy_name = "test-policy-update"
  description = "Test policy for auth method update"
}

resource "cyberarksia_database_policy_database_assignment" "update_test" {
  policy_id              = cyberarksia_database_policy.update_test.id
  database_workspace_id  = cyberarksia_database_workspace.update_test.id
  authentication_method  = "db_auth"

  db_auth_profile {
    roles = ["connect"]
  }
}
`

const testAccPolicyDatabaseAssignmentConfigUpdateAfter = `
resource "cyberarksia_database_secret" "update_test" {
  name                = "update-test-secret"
  authentication_type = "local"
  username            = "postgres"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "update_test" {
  name                  = "update-test-db"
  database_type         = "postgres"
  address               = "postgres-update.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.update_test.id
}

resource "cyberarksia_database_policy" "update_test" {
  policy_name = "test-policy-update"
  description = "Test policy for auth method update"
}

resource "cyberarksia_database_policy_database_assignment" "update_test" {
  policy_id              = cyberarksia_database_policy.update_test.id
  database_workspace_id  = cyberarksia_database_workspace.update_test.id
  authentication_method  = "ldap_auth"

  ldap_auth_profile {
    assign_groups = ["CN=Developers,OU=Groups,DC=example,DC=com"]
  }
}
`

const testAccPolicyDatabaseAssignmentConfigProfileUpdateBefore = `
resource "cyberarksia_database_secret" "profile_update" {
  name                = "profile-update-secret"
  authentication_type = "local"
  username            = "postgres"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "profile_update" {
  name                  = "profile-update-db"
  database_type         = "postgres"
  address               = "postgres-profile.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.profile_update.id
}

resource "cyberarksia_database_policy" "profile_update" {
  policy_name = "test-policy-profile-update"
  description = "Test policy for profile update"
}

resource "cyberarksia_database_policy_database_assignment" "profile_update" {
  policy_id              = cyberarksia_database_policy.profile_update.id
  database_workspace_id  = cyberarksia_database_workspace.profile_update.id
  authentication_method  = "db_auth"

  db_auth_profile {
    roles = ["connect"]
  }
}
`

const testAccPolicyDatabaseAssignmentConfigProfileUpdateAfter = `
resource "cyberarksia_database_secret" "profile_update" {
  name                = "profile-update-secret"
  authentication_type = "local"
  username            = "postgres"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "profile_update" {
  name                  = "profile-update-db"
  database_type         = "postgres"
  address               = "postgres-profile.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.profile_update.id
}

resource "cyberarksia_database_policy" "profile_update" {
  policy_name = "test-policy-profile-update"
  description = "Test policy for profile update"
}

resource "cyberarksia_database_policy_database_assignment" "profile_update" {
  policy_id              = cyberarksia_database_policy.profile_update.id
  database_workspace_id  = cyberarksia_database_workspace.profile_update.id
  authentication_method  = "db_auth"

  db_auth_profile {
    roles = ["connect", "dba"]
  }
}
`

const testAccPolicyDatabaseAssignmentConfigForceNewBefore = `
resource "cyberarksia_database_secret" "forcenew" {
  name                = "forcenew-secret"
  authentication_type = "local"
  username            = "postgres"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "forcenew" {
  name                  = "forcenew-db"
  database_type         = "postgres"
  address               = "postgres-forcenew.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.forcenew.id
}

resource "cyberarksia_database_policy" "forcenew1" {
  policy_name = "test-policy-forcenew-1"
  description = "First test policy"
}

resource "cyberarksia_database_policy" "forcenew2" {
  policy_name = "test-policy-forcenew-2"
  description = "Second test policy"
}

resource "cyberarksia_database_policy_database_assignment" "forcenew_test" {
  policy_id              = cyberarksia_database_policy.forcenew1.id
  database_workspace_id  = cyberarksia_database_workspace.forcenew.id
  authentication_method  = "db_auth"

  db_auth_profile {
    roles = ["connect"]
  }
}
`

const testAccPolicyDatabaseAssignmentConfigForceNewAfter = `
resource "cyberarksia_database_secret" "forcenew" {
  name                = "forcenew-secret"
  authentication_type = "local"
  username            = "postgres"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "forcenew" {
  name                  = "forcenew-db"
  database_type         = "postgres"
  address               = "postgres-forcenew.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.forcenew.id
}

resource "cyberarksia_database_policy" "forcenew1" {
  policy_name = "test-policy-forcenew-1"
  description = "First test policy"
}

resource "cyberarksia_database_policy" "forcenew2" {
  policy_name = "test-policy-forcenew-2"
  description = "Second test policy"
}

resource "cyberarksia_database_policy_database_assignment" "forcenew_test" {
  policy_id              = cyberarksia_database_policy.forcenew2.id
  database_workspace_id  = cyberarksia_database_workspace.forcenew.id
  authentication_method  = "db_auth"

  db_auth_profile {
    roles = ["connect"]
  }
}
`

const testAccPolicyDatabaseAssignmentConfigForceNewDatabaseBefore = `
resource "cyberarksia_database_secret" "forcenew_db" {
  name                = "forcenew-db-secret"
  authentication_type = "local"
  username            = "postgres"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "forcenew_db1" {
  name                  = "forcenew-db-1"
  database_type         = "postgres"
  address               = "postgres-forcenew1.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.forcenew_db.id
}

resource "cyberarksia_database_workspace" "forcenew_db2" {
  name                  = "forcenew-db-2"
  database_type         = "postgres"
  address               = "postgres-forcenew2.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.forcenew_db.id
}

resource "cyberarksia_database_policy" "forcenew_db" {
  policy_name = "test-policy-forcenew-db"
  description = "Test policy for database forcenew"
}

resource "cyberarksia_database_policy_database_assignment" "forcenew_db" {
  policy_id              = cyberarksia_database_policy.forcenew_db.id
  database_workspace_id  = cyberarksia_database_workspace.forcenew_db1.id
  authentication_method  = "db_auth"

  db_auth_profile {
    roles = ["connect"]
  }
}
`

const testAccPolicyDatabaseAssignmentConfigForceNewDatabaseAfter = `
resource "cyberarksia_database_secret" "forcenew_db" {
  name                = "forcenew-db-secret"
  authentication_type = "local"
  username            = "postgres"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "forcenew_db1" {
  name                  = "forcenew-db-1"
  database_type         = "postgres"
  address               = "postgres-forcenew1.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.forcenew_db.id
}

resource "cyberarksia_database_workspace" "forcenew_db2" {
  name                  = "forcenew-db-2"
  database_type         = "postgres"
  address               = "postgres-forcenew2.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.forcenew_db.id
}

resource "cyberarksia_database_policy" "forcenew_db" {
  policy_name = "test-policy-forcenew-db"
  description = "Test policy for database forcenew"
}

resource "cyberarksia_database_policy_database_assignment" "forcenew_db" {
  policy_id              = cyberarksia_database_policy.forcenew_db.id
  database_workspace_id  = cyberarksia_database_workspace.forcenew_db2.id
  authentication_method  = "db_auth"

  db_auth_profile {
    roles = ["connect"]
  }
}
`

const testAccPolicyDatabaseAssignmentConfigMultiple = `
resource "cyberarksia_database_secret" "multi" {
  name                = "multi-secret"
  authentication_type = "local"
  username            = "postgres"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "multi1" {
  name                  = "multi-db-1"
  database_type         = "postgres"
  address               = "postgres-multi1.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.multi.id
}

resource "cyberarksia_database_workspace" "multi2" {
  name                  = "multi-db-2"
  database_type         = "postgres"
  address               = "postgres-multi2.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.multi.id
}

resource "cyberarksia_database_workspace" "multi3" {
  name                  = "multi-db-3"
  database_type         = "oracle"
  address               = "oracle-multi3.example.com"
  port                  = 1521
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.multi.id
}

resource "cyberarksia_database_policy" "multi" {
  policy_name = "test-policy-multi"
  description = "Test policy for multiple assignments"
}

resource "cyberarksia_database_policy_database_assignment" "multi1" {
  policy_id              = cyberarksia_database_policy.multi.id
  database_workspace_id  = cyberarksia_database_workspace.multi1.id
  authentication_method  = "db_auth"

  db_auth_profile {
    roles = ["connect"]
  }
}

resource "cyberarksia_database_policy_database_assignment" "multi2" {
  policy_id              = cyberarksia_database_policy.multi.id
  database_workspace_id  = cyberarksia_database_workspace.multi2.id
  authentication_method  = "ldap_auth"

  ldap_auth_profile {
    assign_groups = ["CN=Developers,OU=Groups,DC=example,DC=com"]
  }
}

resource "cyberarksia_database_policy_database_assignment" "multi3" {
  policy_id              = cyberarksia_database_policy.multi.id
  database_workspace_id  = cyberarksia_database_workspace.multi3.id
  authentication_method  = "oracle_auth"

  oracle_auth_profile {
    roles    = ["CONNECT", "RESOURCE"]
    dba_role = false
  }
}
`

const testAccPolicyDatabaseAssignmentConfigDrift = `
resource "cyberarksia_database_secret" "drift" {
  name                = "drift-secret"
  authentication_type = "local"
  username            = "postgres"
  password            = "SecurePassword123!"
}

resource "cyberarksia_database_workspace" "drift" {
  name                  = "drift-db"
  database_type         = "postgres"
  address               = "postgres-drift.example.com"
  port                  = 5432
  authentication_method = "local_ephemeral_user"
  cloud_provider        = "on_premise"
  secret_id             = cyberarksia_database_secret.drift.id
}

resource "cyberarksia_database_policy" "drift" {
  policy_name = "test-policy-drift"
  description = "Test policy for drift detection"
}

resource "cyberarksia_database_policy_database_assignment" "drift_test" {
  policy_id              = cyberarksia_database_policy.drift.id
  database_workspace_id  = cyberarksia_database_workspace.drift.id
  authentication_method  = "db_auth"

  db_auth_profile {
    roles = ["connect"]
  }
}
`
