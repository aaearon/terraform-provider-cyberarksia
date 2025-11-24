//go:build manual
package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/cyberark/ark-sdk-golang/pkg/auth"
	"github.com/cyberark/ark-sdk-golang/pkg/models"
	authmodels "github.com/cyberark/ark-sdk-golang/pkg/models/auth"
	uapcommonmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/common/models"
	siacommonmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/common/models"
	"github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/vm"
	uapsiavmmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/vm/models"
)

// TestAzureSDK_LocationTypeCaseSensitivity tests whether the SDK works with "Azure" vs "AZURE"
// Run with: go test -tags=manual -v -run TestAzureSDK_LocationTypeCaseSensitivity ./internal/provider
func TestAzureSDK_LocationTypeCaseSensitivity(t *testing.T) {
	t.Log("=== Azure SDK Location Type Case Sensitivity Test ===\n")

	// Test 1: "Azure" (mixed case - API value)
	t.Run("Azure_MixedCase", func(t *testing.T) {
		err := testCreateAzurePolicy(t, "Azure", "test-azure-mixedcase")
		if err != nil {
			t.Logf("❌ FAILED with LocationType=\"Azure\": %v", err)
			t.Fail()
		} else {
			t.Logf("✅ SUCCESS with LocationType=\"Azure\"")
		}
	})

	// Test 2: "AZURE" (all caps - SDK constant)
	t.Run("AZURE_AllCaps", func(t *testing.T) {
		err := testCreateAzurePolicy(t, "AZURE", "test-azure-uppercase")
		if err != nil {
			t.Logf("❌ FAILED with LocationType=\"AZURE\": %v", err)
			t.Fail()
		} else {
			t.Logf("✅ SUCCESS with LocationType=\"AZURE\"")
		}
	})
}

func testCreateAzurePolicy(t *testing.T, locationType string, policyName string) error {
	// Get credentials
	username := os.Getenv("CYBERARK_USERNAME")
	password := os.Getenv("CYBERARK_PASSWORD")
	if username == "" || password == "" {
		return fmt.Errorf("CYBERARK_USERNAME and CYBERARK_PASSWORD must be set")
	}

	// Create auth profile using IdentityServiceUser method (same as provider)
	authProfile := &authmodels.ArkAuthProfile{
		Username:   username,
		AuthMethod: authmodels.IdentityServiceUser,
		AuthMethodSettings: &authmodels.IdentityServiceUserArkAuthMethodSettings{
			IdentityTenantSubdomain:          "",
			IdentityAuthorizationApplication: "__idaptive_cybr_user_oidc",
		},
	}

	// Create secret
	secret := &authmodels.ArkSecret{
		Secret: password,
	}

	// Create ISP auth (caching disabled, same as provider)
	ispAuth := auth.NewArkISPAuth(false)

	// Create in-memory profile (bypass filesystem profile loading)
	inMemoryProfile := &models.ArkProfile{
		ProfileName: "terraform-ephemeral",
		AuthProfiles: map[string]*authmodels.ArkAuthProfile{
			"isp": authProfile,
		},
	}

	// Authenticate with explicit in-memory profile
	// force=true: Always get fresh token (no cache lookups)
	_, err := ispAuth.Authenticate(inMemoryProfile, authProfile, secret, true, false)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Create VM service
	vmService, err := vm.NewArkUAPSIAVMService(ispAuth)
	if err != nil {
		return fmt.Errorf("failed to create VM service: %w", err)
	}

	// Build minimal test policy
	policy := &uapsiavmmodels.ArkUAPSIAVMAccessPolicy{}

	// Set metadata
	policy.Metadata.Name = policyName
	policy.Metadata.TimeZone = "Europe/Amsterdam"
	policy.Metadata.PolicyEntitlement.TargetCategory = "VM"
	policy.Metadata.PolicyEntitlement.LocationType = locationType // THIS IS WHAT WE'RE TESTING!
	policy.Metadata.PolicyEntitlement.PolicyType = "Recurring"
	policy.Metadata.Status.Status = "Active"
	policy.Metadata.PolicyTags = []string{}

	// Set principal (use uapcommonmodels.ArkUAPPrincipal, not siacommonmodels)
	policy.Principals = []uapcommonmodels.ArkUAPPrincipal{
		{
			ID:                  "a1cfc60d-80e1-489c-8251-c0d7bcb84fc9",
			Name:                "timtest@cyberark.cloud.40562",
			Type:                "USER",
			SourceDirectoryName: "CyberArk Cloud Directory",
			SourceDirectoryID:   "09B9A9B0-6CE8-465F-AB03-65766D33B05E",
		},
	}

	// Set conditions (Conditions is ArkUAPSIACommonConditions which embeds ArkUAPConditions)
	policy.Conditions = siacommonmodels.ArkUAPSIACommonConditions{
		ArkUAPConditions: uapcommonmodels.ArkUAPConditions{
			MaxSessionDuration: 2,
			AccessWindow: uapcommonmodels.ArkUAPTimeCondition{
				DaysOfTheWeek: []int{0, 1, 2, 3, 4, 5, 6},
			},
		},
		IdleTime: 10,
	}

	// Set Azure targets
	policy.Targets.AzureResource = &uapsiavmmodels.ArkUAPSIAVMAzureResource{
		Regions:        []string{"eastus"},
		Tags:           []uapsiavmmodels.ArkUAPSIAVMKeyValTag{},
		ResourceGroups: []string{},
		VNetIDs:        []string{},
		Subscriptions:  []string{},
	}

	// Set SSH behavior
	policy.Behavior.SSHProfile = &uapsiavmmodels.ArkUAPSSIAVMSSHProfile{
		Username: "azureuser",
	}

	t.Logf("  Testing with LocationType=%q", locationType)

	// Attempt to serialize (this is where SDK may fail for "Azure")
	serialized, err := policy.Serialize()
	if err != nil {
		return fmt.Errorf("Serialize() failed: %w", err)
	}

	// Log targets structure to see what JSON key is used
	if targetsData, ok := serialized["targets"].(map[string]interface{}); ok {
		targetsJSON, _ := json.MarshalIndent(targetsData, "    ", "  ")
		t.Logf("  Targets JSON key in serialized output:\n    %s", string(targetsJSON))
	}

	// Try to create policy via SDK
	created, err := vmService.AddPolicy(policy)
	if err != nil {
		return fmt.Errorf("AddPolicy() failed: %w", err)
	}

	t.Logf("  ✓ Created policy ID: %s", created.Metadata.PolicyID)

	// Clean up - delete the test policy
	deleteReq := &uapcommonmodels.ArkUAPDeletePolicyRequest{
		PolicyID: created.Metadata.PolicyID,
	}
	_ = vmService.DeletePolicy(deleteReq)
	t.Logf("  ✓ Cleaned up test policy")

	return nil
}
