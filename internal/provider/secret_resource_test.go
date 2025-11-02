// Package provider implements acceptance tests for secret resource
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ============================================================================
// Phase 4 (User Story 2) Tests: Strong Account CRUD Lifecycle
// ============================================================================

// TestAccSecret_basic tests basic CRUD lifecycle for strong account resource
func TestAccSecret_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSecretConfigBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_secret.test", "name", "test-strong-account"),
					resource.TestCheckResourceAttr("cyberarksia_database_secret.test", "authentication_type", "local"),
					resource.TestCheckResourceAttr("cyberarksia_database_secret.test", "username", "db_admin"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_secret.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "cyberarksia_database_secret.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Password is sensitive and won't be in state
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

// TestAccSecret_localAuth tests local authentication strong account
func TestAccSecret_localAuth(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretConfigLocalAuth,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_secret.local", "name", "local-auth-account"),
					resource.TestCheckResourceAttr("cyberarksia_database_secret.local", "authentication_type", "local"),
					resource.TestCheckResourceAttr("cyberarksia_database_secret.local", "username", "postgres_admin"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_secret.local", "id"),
				),
			},
		},
	})
}

// TestAccSecret_domainAuth tests Active Directory authentication strong account
func TestAccSecret_domainAuth(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretConfigDomainAuth,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_secret.domain", "name", "domain-auth-account"),
					resource.TestCheckResourceAttr("cyberarksia_database_secret.domain", "authentication_type", "domain"),
					resource.TestCheckResourceAttr("cyberarksia_database_secret.domain", "username", "CORP\\sqladmin"),
					resource.TestCheckResourceAttr("cyberarksia_database_secret.domain", "domain", "corp.example.com"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_secret.domain", "id"),
				),
			},
		},
	})
}

// TestAccSecret_awsIAM tests AWS IAM authentication strong account
func TestAccSecret_awsIAM(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretConfigAwsIAM,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_secret.aws_iam", "name", "aws-iam-account"),
					resource.TestCheckResourceAttr("cyberarksia_database_secret.aws_iam", "authentication_type", "aws_iam"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_secret.aws_iam", "id"),
				),
			},
		},
	})
}

// TestAccSecret_credentialUpdate tests credential rotation/update
func TestAccSecret_credentialUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with initial credentials
			{
				Config: testAccSecretConfigCredentialsBefore,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_secret.rotation_test", "name", "rotation-test-account"),
					resource.TestCheckResourceAttr("cyberarksia_database_secret.rotation_test", "username", "initial_user"),
				),
			},
			// Step 2: Update credentials (password and username)
			{
				Config: testAccSecretConfigCredentialsAfter,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_secret.rotation_test", "name", "rotation-test-account"),
					resource.TestCheckResourceAttr("cyberarksia_database_secret.rotation_test", "username", "updated_user"),
				),
			},
		},
	})
}

// TestAccSecret_import tests ImportState functionality
func TestAccSecret_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create resource
			{
				Config: testAccSecretConfigBasic,
			},
			// Test import
			{
				ResourceName:      "cyberarksia_database_secret.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Password is sensitive and won't be in state
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

// ============================================================================
// Phase 5 (User Story 3) Tests: Strong Account Credential Update
// ============================================================================

// TestAccSecret_updateCredentials tests strong account credential rotation
func TestAccSecret_updateCredentials(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with initial password
			{
				Config: testAccSecretConfigUpdateBefore,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_secret.update_test", "name", "update-test-account"),
					resource.TestCheckResourceAttr("cyberarksia_database_secret.update_test", "username", "db_user"),
					resource.TestCheckResourceAttr("cyberarksia_database_secret.update_test", "description", "Initial credentials"),
				),
			},
			// Step 2: Rotate password (credential update)
			{
				Config: testAccSecretConfigUpdateAfter,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_secret.update_test", "name", "update-test-account"),
					resource.TestCheckResourceAttr("cyberarksia_database_secret.update_test", "username", "db_user"),
					resource.TestCheckResourceAttr("cyberarksia_database_secret.update_test", "description", "Rotated credentials"),
				),
			},
			// Step 3: Verify import still works after update
			{
				ResourceName:      "cyberarksia_database_secret.update_test",
				ImportState:       true,
				ImportStateVerify: true,
				// Password is sensitive and won't be in state
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

// ============================================================================
// Test Configurations
// ============================================================================

const testAccSecretConfigBasic = `
resource "cyberark_sia_database_workspace" "test" {
  name          = "test-db-for-strong-account"
  database_type = "postgresql"
  address       = "postgres.example.com"
  port          = 5432
}

resource "cyberarksia_database_secret" "test" {
  name               = "test-strong-account"
  authentication_type = "local"
  username           = "db_admin"
  password           = "InitialPassword123!"

  description = "Test strong account"

  tags = {
    Environment = "test"
    ManagedBy   = "Terraform"
  }
}
`

const testAccSecretConfigLocalAuth = `
resource "cyberark_sia_database_workspace" "postgres" {
  name          = "postgres-db-local"
  database_type = "postgresql"
  address       = "postgres-local.example.com"
  port          = 5432
}

resource "cyberarksia_database_secret" "local" {
  name               = "local-auth-account"
  authentication_type = "local"
  username           = "postgres_admin"
  password           = "SecurePassword456!"

  description = "Local authentication strong account for PostgreSQL"

  tags = {
    Environment = "test"
    AuthType    = "local"
  }
}
`

const testAccSecretConfigDomainAuth = `
resource "cyberark_sia_database_workspace" "sqlserver" {
  name          = "sqlserver-db-domain"
  database_type = "sqlserver"
  address       = "sqlserver-domain.example.com"
  port          = 1433
}

resource "cyberarksia_database_secret" "domain" {
  name               = "domain-auth-account"
  authentication_type = "domain"
  username           = "CORP\\sqladmin"
  password           = "DomainPassword789!"
  domain             = "corp.example.com"

  description = "Active Directory authentication strong account for SQL Server"

  tags = {
    Environment = "test"
    AuthType    = "domain"
  }
}
`

const testAccSecretConfigAwsIAM = `
resource "cyberark_sia_database_workspace" "rds" {
  name          = "rds-db-iam"
  database_type = "postgresql"
  address       = "mydb.abc123.us-east-1.rds.amazonaws.com"
  port          = 5432
  cloud_provider = "aws"
  region        = "us-east-1"
}

resource "cyberarksia_database_secret" "aws_iam" {
  name               = "aws-iam-account"
  authentication_type = "aws_iam"
  aws_access_key_id     = "AKIAIOSFODNN7EXAMPLE"
  aws_secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

  description = "AWS IAM authentication strong account for RDS"

  tags = {
    Environment = "test"
    AuthType    = "aws_iam"
  }
}
`

const testAccSecretConfigCredentialsBefore = `
resource "cyberark_sia_database_workspace" "rotation" {
  name          = "rotation-test-db"
  database_type = "postgresql"
  address       = "postgres-rotation.example.com"
  port          = 5432
}

resource "cyberarksia_database_secret" "rotation_test" {
  name               = "rotation-test-account"
  authentication_type = "local"
  username           = "initial_user"
  password           = "InitialPassword123!"

  description = "Account for credential rotation testing"

  tags = {
    Environment = "test"
    Phase       = "before-rotation"
  }
}
`

const testAccSecretConfigCredentialsAfter = `
resource "cyberark_sia_database_workspace" "rotation" {
  name          = "rotation-test-db"
  database_type = "postgresql"
  address       = "postgres-rotation.example.com"
  port          = 5432
}

resource "cyberarksia_database_secret" "rotation_test" {
  name               = "rotation-test-account"
  authentication_type = "local"
  username           = "updated_user"
  password           = "RotatedPassword456!"

  description = "Account for credential rotation testing"

  tags = {
    Environment = "test"
    Phase       = "after-rotation"
  }
}
`

const testAccSecretConfigUpdateBefore = `
resource "cyberark_sia_database_workspace" "update" {
  name          = "update-test-db"
  database_type = "postgresql"
  address       = "postgres-update.example.com"
  port          = 5432
}

resource "cyberarksia_database_secret" "update_test" {
  name               = "update-test-account"
  authentication_type = "local"
  username           = "db_user"
  password           = "InitialPassword123!"

  description = "Initial credentials"

  tags = {
    Environment = "test"
    Phase       = "before-update"
  }
}
`

const testAccSecretConfigUpdateAfter = `
resource "cyberark_sia_database_workspace" "update" {
  name          = "update-test-db"
  database_type = "postgresql"
  address       = "postgres-update.example.com"
  port          = 5432
}

resource "cyberarksia_database_secret" "update_test" {
  name               = "update-test-account"
  authentication_type = "local"
  username           = "db_user"
  password           = "RotatedPassword456!"

  description = "Rotated credentials"

  tags = {
    Environment = "test"
    Phase       = "after-update"
  }
}
`
