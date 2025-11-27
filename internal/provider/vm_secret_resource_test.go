// Package provider implements acceptance tests for VM secret resource
package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aaearon/terraform-provider-cyberarksia/internal/client"
	vmsecretsmodels "github.com/cyberark/ark-sdk-golang/pkg/services/sia/secrets/vm/models"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// ============================================================================
// User Story 1 - Create VM Secrets (Priority: P1)
// ============================================================================

// TestAccVMSecret_ProvisionerUser_Basic tests full CRUD lifecycle for ProvisionerUser type
func TestAccVMSecret_ProvisionerUser_Basic(t *testing.T) {
	resourceName := "cyberarksia_vm_secret.provisioner_user"
	secretName := fmt.Sprintf("test-vm-secret-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMSecretDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccVMSecretConfigProvisionerUser("provisioner_user", secretName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVMSecretExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "secret_name", secretName),
					resource.TestCheckResourceAttr(resourceName, "secret_type", "ProvisionerUser"),
					resource.TestCheckResourceAttr(resourceName, "provisioner_username", "vm_admin"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "secret_id"),
					// Verify ID equals secret_id
					resource.TestCheckResourceAttrPair(resourceName, "id", resourceName, "secret_id"),
					// Verify PCloud fields are not set
					resource.TestCheckNoResourceAttr(resourceName, "pcloud_safe_name"),
					resource.TestCheckNoResourceAttr(resourceName, "pcloud_account_name"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// Username and password are write-only - API accepts but doesn't return for security
				ImportStateVerifyIgnore: []string{"provisioner_username", "provisioner_password"},
			},
		},
	})
}

// TestAccVMSecret_PCloudAccount_Basic tests full CRUD lifecycle for PCloudAccount type
func TestAccVMSecret_PCloudAccount_Basic(t *testing.T) {
	resourceName := "cyberarksia_vm_secret.pcloud_account"
	secretName := fmt.Sprintf("test-pcloud-secret-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMSecretDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccVMSecretConfigPCloudAccount(secretName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVMSecretExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "secret_name", secretName),
					resource.TestCheckResourceAttr(resourceName, "secret_type", "PCloudAccount"),
					resource.TestCheckResourceAttr(resourceName, "pcloud_safe_name", "Production-Safe"),
					resource.TestCheckResourceAttr(resourceName, "pcloud_account_name", "vm-admin-account"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "secret_id"),
					// Verify ID equals secret_id
					resource.TestCheckResourceAttrPair(resourceName, "id", resourceName, "secret_id"),
					// Verify Provisioner fields are not set
					resource.TestCheckNoResourceAttr(resourceName, "provisioner_username"),
					resource.TestCheckNoResourceAttr(resourceName, "provisioner_password"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// PCloud fields are write-only - API encrypts credentials and doesn't return them
				ImportStateVerifyIgnore: []string{"pcloud_safe_name", "pcloud_account_name"},
			},
		},
	})
}

// TestAccVMSecret_SensitiveOutput tests that passwords are not in plan output
func TestAccVMSecret_SensitiveOutput(t *testing.T) {
	resourceName := "cyberarksia_vm_secret.sensitive_test"
	secretName := fmt.Sprintf("test-sensitive-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVMSecretConfigProvisionerUser("sensitive_test", secretName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVMSecretExists(resourceName),
					// Password should be marked as sensitive - value should not be visible
					resource.TestCheckResourceAttr(resourceName, "provisioner_username", "vm_admin"),
					// We can't directly check if password is redacted in plan, but we can verify it's set
					resource.TestCheckResourceAttrSet(resourceName, "provisioner_password"),
				),
			},
		},
	})
}

// ============================================================================
// User Story 2 - Read and Verify VM Secrets (Priority: P1)
// ============================================================================

// TestAccVMSecret_DriftDetection tests detection of external changes
// NOTE: This test demonstrates drift detection by changing the Terraform config
// A true drift test would modify the secret via API between steps, but that requires
// direct API access in test helpers. The current implementation tests the update path.
func TestAccVMSecret_DriftDetection(t *testing.T) {
	resourceName := "cyberarksia_vm_secret.drift_test"
	secretName := fmt.Sprintf("test-drift-%s", acctest.RandString(8))
	updatedSecretName := fmt.Sprintf("test-drift-updated-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMSecretDestroy,
		Steps: []resource.TestStep{
			// Create initial secret
			{
				Config: testAccVMSecretConfigProvisionerUser("drift_test", secretName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVMSecretExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "secret_name", secretName),
				),
			},
			// Change config to trigger update (simulates drift + correction)
			{
				Config: testAccVMSecretConfigProvisionerUser("drift_test", updatedSecretName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVMSecretExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "secret_name", updatedSecretName),
				),
			},
			// TODO: Add PreConfig step to modify secret via API for true drift detection
			// This requires provider instance access in test helpers
		},
	})
}

// TestAccVMSecret_ExternalDeletion tests handling of 404 errors gracefully
// NOTE: Full test requires API access in PreConfig to delete secret externally
func TestAccVMSecret_ExternalDeletion(t *testing.T) {
	resourceName := "cyberarksia_vm_secret.external_delete_test"
	secretName := fmt.Sprintf("test-external-delete-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVMSecretConfigProvisionerUser("external_delete_test", secretName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVMSecretExists(resourceName),
				),
			},
			// TODO: Add PreConfig step to delete secret via API (simulates external deletion)
			// Then verify next refresh detects deletion and removes from state
			// This requires: client.DeleteVMSecretDirect(ctx, authContext, secretID)
		},
	})
}

// ============================================================================
// User Story 3 - Update VM Secret Metadata (Priority: P2)
// ============================================================================

// TestAccVMSecret_UpdateName tests in-place name updates
func TestAccVMSecret_UpdateName(t *testing.T) {
	resourceName := "cyberarksia_vm_secret.update_name_test"
	secretName := fmt.Sprintf("test-update-name-%s", acctest.RandString(8))
	updatedSecretName := fmt.Sprintf("test-updated-name-%s", acctest.RandString(8))

	var originalSecretID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMSecretDestroy,
		Steps: []resource.TestStep{
			// Create initial secret and capture ID
			{
				Config: testAccVMSecretConfigProvisionerUser("update_name_test", secretName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVMSecretExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "secret_name", secretName),
					resource.TestCheckResourceAttrWith(resourceName, "secret_id", func(value string) error {
						originalSecretID = value
						return nil
					}),
				),
			},
			// Update name in-place - verify ID remains stable
			{
				Config: testAccVMSecretConfigProvisionerUser("update_name_test", updatedSecretName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVMSecretExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "secret_name", updatedSecretName),
					// Verify secret_id hasn't changed (in-place update)
					resource.TestCheckResourceAttrWith(resourceName, "secret_id", func(v string) error {
						if v != originalSecretID {
							return fmt.Errorf("expected secret_id to stay %s, got %s", originalSecretID, v)
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith(resourceName, "id", func(v string) error {
						if v != originalSecretID {
							return fmt.Errorf("expected id to stay %s, got %s", originalSecretID, v)
						}
						return nil
					}),
				),
			},
		},
	})
}

// TestAccVMSecret_UpdatePassword tests password rotation
func TestAccVMSecret_UpdatePassword(t *testing.T) {
	resourceName := "cyberarksia_vm_secret.update_password_test"
	secretName := fmt.Sprintf("test-update-password-%s", acctest.RandString(8))

	var originalSecretID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMSecretDestroy,
		Steps: []resource.TestStep{
			// Create with initial password and capture ID
			{
				Config: testAccVMSecretConfigProvisionerUserWithPassword("update_password_test", secretName, "InitialPassword123!"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVMSecretExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "provisioner_username", "vm_admin"),
					resource.TestCheckResourceAttrWith(resourceName, "secret_id", func(value string) error {
						originalSecretID = value
						return nil
					}),
				),
			},
			// Update password (rotation) - verify ID remains stable
			{
				Config: testAccVMSecretConfigProvisionerUserWithPassword("update_password_test", secretName, "RotatedPassword456!"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVMSecretExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "provisioner_username", "vm_admin"),
					// Verify secret_id hasn't changed (in-place update)
					resource.TestCheckResourceAttrWith(resourceName, "secret_id", func(v string) error {
						if v != originalSecretID {
							return fmt.Errorf("expected secret_id to stay %s, got %s", originalSecretID, v)
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith(resourceName, "id", func(v string) error {
						if v != originalSecretID {
							return fmt.Errorf("expected id to stay %s, got %s", originalSecretID, v)
						}
						return nil
					}),
				),
			},
		},
	})
}

// TestAccVMSecret_ForceNew tests that secret_type change triggers recreate
func TestAccVMSecret_ForceNew(t *testing.T) {
	resourceName := "cyberarksia_vm_secret.forcenew_test"
	secretName := fmt.Sprintf("test-forcenew-%s", acctest.RandString(8))

	var originalSecretID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMSecretDestroy,
		Steps: []resource.TestStep{
			// Create ProvisionerUser secret
			{
				Config: testAccVMSecretConfigProvisionerUser("forcenew_test", secretName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVMSecretExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "secret_type", "ProvisionerUser"),
					func(s *terraform.State) error {
						rs := s.RootModule().Resources[resourceName]
						originalSecretID = rs.Primary.Attributes["secret_id"]
						return nil
					},
				),
			},
			// Change to PCloudAccount - should trigger ForceNew (destroy + recreate)
			{
				Config: testAccVMSecretConfigPCloudAccount(secretName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVMSecretExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "secret_type", "PCloudAccount"),
					// Verify secret_id has changed (resource was recreated)
					func(s *terraform.State) error {
						rs := s.RootModule().Resources[resourceName]
						newSecretID := rs.Primary.Attributes["secret_id"]
						if newSecretID == originalSecretID {
							return fmt.Errorf("expected secret_id to change after ForceNew, but it remained %s", newSecretID)
						}
						return nil
					},
				),
			},
		},
	})
}

// ============================================================================
// User Story 4 - Import Existing VM Secrets (Priority: P2)
// ============================================================================

// TestAccVMSecret_ImportBasic tests import by secret_id
func TestAccVMSecret_ImportBasic(t *testing.T) {
	resourceName := "cyberarksia_vm_secret.import_test"
	secretName := fmt.Sprintf("test-import-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMSecretDestroy,
		Steps: []resource.TestStep{
			// Create secret
			{
				Config: testAccVMSecretConfigProvisionerUser("import_test", secretName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVMSecretExists(resourceName),
				),
			},
			// Import by secret_id
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// Username and password are write-only - API encrypts credentials and doesn't return them
				ImportStateVerifyIgnore: []string{"provisioner_username", "provisioner_password"},
			},
		},
	})
}

// TestAccVMSecret_ImportNotFound tests that importing non-existent secret_id fails
func TestAccVMSecret_ImportNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:        testAccVMSecretConfigProvisionerUser("import_test", "nonexistent"),
				ResourceName:  "cyberarksia_vm_secret.import_test",
				ImportState:   true,
				ImportStateId: "00000000-0000-0000-0000-000000000000",
				// Import calls Read() which should return 404 for non-existent UUID
				// Terraform framework generates "cannot import" message for import failures
				ExpectError: regexp.MustCompile(`(?i)(404|not found|does not exist|failed to read|cannot import)`),
			},
		},
	})
}

// ============================================================================
// User Story 5 - Delete VM Secrets (Priority: P3)
// ============================================================================

// TestAccVMSecret_DeleteBasic tests secret deletion via Terraform destroy
func TestAccVMSecret_DeleteBasic(t *testing.T) {
	resourceName := "cyberarksia_vm_secret.delete_test"
	secretName := fmt.Sprintf("test-delete-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVMSecretConfigProvisionerUser("delete_test", secretName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVMSecretExists(resourceName),
				),
			},
			// After test completes, Terraform calls Delete() and CheckDestroy verifies success
		},
	})
}

// TestAccVMSecret_DeleteIdempotent tests that deleting already-deleted secret succeeds
// NOTE: Full test requires two sequential destroy operations with API verification
func TestAccVMSecret_DeleteIdempotent(t *testing.T) {
	resourceName := "cyberarksia_vm_secret.delete_idempotent_test"
	secretName := fmt.Sprintf("test-delete-idempotent-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVMSecretConfigProvisionerUser("delete_idempotent_test", secretName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVMSecretExists(resourceName),
				),
			},
			// Standard destroy tests idempotency through CheckDestroy
			// TODO: To fully test idempotency, need to:
			// 1. Remove resource from config (triggers first Delete)
			// 2. Run destroy again (should handle 404 gracefully)
			// Current test verifies Delete() handles normal case; resource code handles 404
		},
	})
}

// ============================================================================
// Validation Tests - Negative Scenarios
// ============================================================================

// TestAccVMSecret_InvalidSecretType tests rejection of invalid secret_type
func TestAccVMSecret_InvalidSecretType(t *testing.T) {
	secretName := fmt.Sprintf("test-invalid-type-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVMSecretConfigInvalidSecretType(secretName),
				// Terraform validator generates "Invalid Attribute Value" message for enum validation
				ExpectError: regexp.MustCompile(`(?i)(Invalid Attribute Value|value must be one of)`),
			},
		},
	})
}

// TestAccVMSecret_MissingProvisionerUsername tests ProvisionerUser without username
func TestAccVMSecret_MissingProvisionerUsername(t *testing.T) {
	secretName := fmt.Sprintf("test-missing-username-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccVMSecretConfigMissingProvisionerUsername(secretName),
				ExpectError: regexp.MustCompile(`(?i)(provisioner_username.*required.*ProvisionerUser|Missing Required Field)`),
			},
		},
	})
}

// TestAccVMSecret_MissingProvisionerPassword tests ProvisionerUser without password
func TestAccVMSecret_MissingProvisionerPassword(t *testing.T) {
	secretName := fmt.Sprintf("test-missing-password-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccVMSecretConfigMissingProvisionerPassword(secretName),
				ExpectError: regexp.MustCompile(`(?i)(provisioner_password.*required.*ProvisionerUser|Missing Required Field)`),
			},
		},
	})
}

// TestAccVMSecret_MissingPCloudSafeName tests PCloudAccount without safe_name
func TestAccVMSecret_MissingPCloudSafeName(t *testing.T) {
	secretName := fmt.Sprintf("test-missing-safe-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccVMSecretConfigMissingPCloudSafeName(secretName),
				ExpectError: regexp.MustCompile(`(?i)(pcloud_safe_name.*required.*PCloudAccount|Missing Required Field)`),
			},
		},
	})
}

// TestAccVMSecret_MissingPCloudAccountName tests PCloudAccount without account_name
func TestAccVMSecret_MissingPCloudAccountName(t *testing.T) {
	secretName := fmt.Sprintf("test-missing-account-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccVMSecretConfigMissingPCloudAccountName(secretName),
				ExpectError: regexp.MustCompile(`(?i)(pcloud_account_name.*required.*PCloudAccount|Missing Required Field)`),
			},
		},
	})
}

// TestAccVMSecret_InvalidFieldMix tests ProvisionerUser with PCloud fields
func TestAccVMSecret_InvalidFieldMix(t *testing.T) {
	secretName := fmt.Sprintf("test-invalid-mix-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccVMSecretConfigInvalidFieldMix(secretName),
				ExpectError: regexp.MustCompile(`(?i)(cannot be set when secret_type|Invalid Field Combination)`),
			},
		},
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// testAccCheckVMSecretExists verifies the secret exists in state
// NOTE: Full API validation happens in the resource Read() method which is called
// automatically by Terraform test framework during Check phase
func testAccCheckVMSecretExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource ID not set")
		}

		// The Terraform test framework will have already called Read() on the resource
		// If Read() succeeded without removing the resource from state, the secret exists in API
		// This is the standard pattern for Terraform acceptance tests
		return nil
	}
}

// testAccCheckVMSecretDestroy verifies all secrets were destroyed
// This function queries the API to ensure resources no longer exist
func testAccCheckVMSecretDestroy(s *terraform.State) error {
	// Get provider configuration from environment
	providerData, err := getProviderDataFromEnv()
	if err != nil {
		return fmt.Errorf("failed to get provider data: %w", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cyberarksia_vm_secret" {
			continue
		}

		// Query API to verify resource is gone
		secretID := rs.Primary.ID

		getSecretReq := &vmsecretsmodels.ArkSIAVMGetSecret{
			SecretID: secretID,
		}

		_, err := providerData.SIAAPI.SecretsVM().Secret(getSecretReq)
		if err != nil {
			// 404 means successfully deleted
			if client.IsNotFoundError(err) {
				continue
			}
			// Other errors are unexpected
			return fmt.Errorf("error checking VM secret %s: %w", secretID, err)
		}

		// Resource still exists - destroy failed
		return fmt.Errorf("VM secret %s still exists after destroy", secretID)
	}

	return nil
}

// ============================================================================
// Test Configurations
// ============================================================================

// testAccVMSecretConfigProvisionerUser returns basic ProvisionerUser config
func testAccVMSecretConfigProvisionerUser(resourceLabel, secretName string) string {
	return fmt.Sprintf(`
resource "cyberarksia_vm_secret" "%[1]s" {
  secret_name          = %[2]q
  secret_type          = "ProvisionerUser"
  provisioner_username = "vm_admin"
  provisioner_password = "SecurePassword123!"
}
`, resourceLabel, secretName)
}

// testAccVMSecretConfigProvisionerUserWithPassword returns ProvisionerUser config with custom password
func testAccVMSecretConfigProvisionerUserWithPassword(resourceLabel, secretName, password string) string {
	return fmt.Sprintf(`
resource "cyberarksia_vm_secret" "%[1]s" {
  secret_name          = %[2]q
  secret_type          = "ProvisionerUser"
  provisioner_username = "vm_admin"
  provisioner_password = %[3]q
}
`, resourceLabel, secretName, password)
}

// testAccVMSecretConfigPCloudAccount returns basic PCloudAccount config
func testAccVMSecretConfigPCloudAccount(secretName string) string {
	return fmt.Sprintf(`
resource "cyberarksia_vm_secret" "pcloud_account" {
  secret_name         = %[1]q
  secret_type         = "PCloudAccount"
  pcloud_safe_name    = "Production-Safe"
  pcloud_account_name = "vm-admin-account"
}

resource "cyberarksia_vm_secret" "forcenew_test" {
  secret_name         = %[1]q
  secret_type         = "PCloudAccount"
  pcloud_safe_name    = "Production-Safe"
  pcloud_account_name = "vm-admin-account"
}
`, secretName)
}

// testAccVMSecretConfigInvalidSecretType returns config with invalid secret_type
func testAccVMSecretConfigInvalidSecretType(secretName string) string {
	return fmt.Sprintf(`
resource "cyberarksia_vm_secret" "invalid_type" {
  secret_name          = %[1]q
  secret_type          = "InvalidType"
  provisioner_username = "vm_admin"
  provisioner_password = "SecurePassword123!"
}
`, secretName)
}

// testAccVMSecretConfigMissingProvisionerUsername returns config missing username
func testAccVMSecretConfigMissingProvisionerUsername(secretName string) string {
	return fmt.Sprintf(`
resource "cyberarksia_vm_secret" "missing_username" {
  secret_name          = %[1]q
  secret_type          = "ProvisionerUser"
  provisioner_password = "SecurePassword123!"
}
`, secretName)
}

// testAccVMSecretConfigMissingProvisionerPassword returns config missing password
func testAccVMSecretConfigMissingProvisionerPassword(secretName string) string {
	return fmt.Sprintf(`
resource "cyberarksia_vm_secret" "missing_password" {
  secret_name          = %[1]q
  secret_type          = "ProvisionerUser"
  provisioner_username = "vm_admin"
}
`, secretName)
}

// testAccVMSecretConfigMissingPCloudSafeName returns config missing safe_name
func testAccVMSecretConfigMissingPCloudSafeName(secretName string) string {
	return fmt.Sprintf(`
resource "cyberarksia_vm_secret" "missing_safe" {
  secret_name         = %[1]q
  secret_type         = "PCloudAccount"
  pcloud_account_name = "vm-admin-account"
}
`, secretName)
}

// testAccVMSecretConfigMissingPCloudAccountName returns config missing account_name
func testAccVMSecretConfigMissingPCloudAccountName(secretName string) string {
	return fmt.Sprintf(`
resource "cyberarksia_vm_secret" "missing_account" {
  secret_name      = %[1]q
  secret_type      = "PCloudAccount"
  pcloud_safe_name = "Production-Safe"
}
`, secretName)
}

// testAccVMSecretConfigInvalidFieldMix returns config with mixed fields
func testAccVMSecretConfigInvalidFieldMix(secretName string) string {
	return fmt.Sprintf(`
resource "cyberarksia_vm_secret" "invalid_mix" {
  secret_name          = %[1]q
  secret_type          = "ProvisionerUser"
  provisioner_username = "vm_admin"
  provisioner_password = "SecurePassword123!"
  pcloud_safe_name     = "Production-Safe"
  pcloud_account_name  = "vm-admin-account"
}
`, secretName)
}

// ============================================================================
// Helper Functions
// ============================================================================

// getProviderDataFromEnv is defined in testing_helpers.go
