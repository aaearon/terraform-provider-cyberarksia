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
				// Timestamps may have precision differences between API responses
				ImportStateVerifyIgnore: []string{"password", "created_at", "last_modified"},
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
					// Domain is extracted from username format DOMAIN\username → "CORP"
					resource.TestCheckResourceAttr("cyberarksia_database_secret.domain", "domain", "CORP"),
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
				// Timestamps may have precision differences between API responses
				ImportStateVerifyIgnore: []string{"password", "created_at", "last_modified"},
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
					resource.TestCheckResourceAttrSet("cyberarksia_database_secret.update_test", "id"),
				),
			},
			// Step 2: Rotate password (credential update)
			{
				Config: testAccSecretConfigUpdateAfter,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cyberarksia_database_secret.update_test", "name", "update-test-account"),
					resource.TestCheckResourceAttr("cyberarksia_database_secret.update_test", "username", "db_user"),
					resource.TestCheckResourceAttrSet("cyberarksia_database_secret.update_test", "id"),
				),
			},
			// Step 3: Verify import still works after update
			{
				ResourceName:      "cyberarksia_database_secret.update_test",
				ImportState:       true,
				ImportStateVerify: true,
				// Password is sensitive and won't be in state
				// Timestamps may have precision differences between API responses
				ImportStateVerifyIgnore: []string{"password", "created_at", "last_modified"},
			},
		},
	})
}

// ============================================================================
// Test Configurations
// ============================================================================

const testAccSecretConfigBasic = `
resource "cyberarksia_database_secret" "test" {
  name               = "test-strong-account"
  authentication_type = "local"
  username           = "db_admin"
  password           = "InitialPassword123!"
}
`

const testAccSecretConfigLocalAuth = `
resource "cyberarksia_database_secret" "local" {
  name               = "local-auth-account"
  authentication_type = "local"
  username           = "postgres_admin"
  password           = "SecurePassword456!"
}
`

const testAccSecretConfigDomainAuth = `
resource "cyberarksia_database_secret" "domain" {
  name               = "domain-auth-account"
  authentication_type = "domain"
  username           = "CORP\\sqladmin"
  password           = "DomainPassword789!"
  domain             = "CORP"
}
`

const testAccSecretConfigAwsIAM = `
resource "cyberarksia_database_secret" "aws_iam" {
  name                  = "aws-iam-account"
  authentication_type   = "aws_iam"
  aws_access_key_id     = "AKIAIOSFODNN7EXAMPLE"
  aws_secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  aws_account           = "123456789012"
  aws_username          = "sia-database-user"
}
`

const testAccSecretConfigCredentialsBefore = `
resource "cyberarksia_database_secret" "rotation_test" {
  name               = "rotation-test-account"
  authentication_type = "local"
  username           = "initial_user"
  password           = "InitialPassword123!"
}
`

const testAccSecretConfigCredentialsAfter = `
resource "cyberarksia_database_secret" "rotation_test" {
  name               = "rotation-test-account"
  authentication_type = "local"
  username           = "updated_user"
  password           = "RotatedPassword456!"
}
`

const testAccSecretConfigUpdateBefore = `
resource "cyberarksia_database_secret" "update_test" {
  name               = "update-test-account"
  authentication_type = "local"
  username           = "db_user"
  password           = "InitialPassword123!"
}
`

const testAccSecretConfigUpdateAfter = `
resource "cyberarksia_database_secret" "update_test" {
  name               = "update-test-account"
  authentication_type = "local"
  username           = "db_user"
  password           = "RotatedPassword456!"
}
`
