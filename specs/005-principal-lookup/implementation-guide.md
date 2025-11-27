# Implementation Guide: Principal Lookup Data Source

**Feature**: `cyberarksia_principal` data source
**Purpose**: Step-by-step implementation guidance with cross-references

---

## Prerequisites

Before starting implementation, review:
1. **Data Model**: `data-model.md` - Entity definitions and field mappings
2. **API Contracts**: `contracts/identity-apis.md` - ARK SDK API behavior
3. **Investigation**: `docs/development/principal-lookup-investigation.md` - PoC validation
4. **Existing Pattern**: `internal/provider/access_policy_data_source.go` - Data source template

---

## Implementation Order

### Phase 1: Core Data Source (TDD)

#### Step 1.1: Create Test File First (TDD)
**File**: `internal/provider/principal_data_source_test.go`
**Reference Pattern**: `internal/provider/access_policy_data_source.go:73-88` (Configure method)

```go
package provider

import (
	"testing"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPrincipalDataSource_CloudUser(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPrincipalDataSourceConfig_CloudUser(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Schema validation - see data-model.md "Terraform Schema" section
					resource.TestCheckResourceAttrSet("data.cyberarksia_principal.test", "id"),
					resource.TestCheckResourceAttr("data.cyberarksia_principal.test", "principal_type", "USER"),
					resource.TestCheckResourceAttr("data.cyberarksia_principal.test", "directory_name", "CyberArk Cloud Directory"),
					resource.TestCheckResourceAttrSet("data.cyberarksia_principal.test", "directory_id"),
					resource.TestCheckResourceAttrSet("data.cyberarksia_principal.test", "display_name"),
					resource.TestCheckResourceAttrSet("data.cyberarksia_principal.test", "email"),
				),
			},
		},
	})
}

func testAccPrincipalDataSourceConfig_CloudUser() string {
	return `
data "cyberarksia_principal" "test" {
  name = "tim.schindler@cyberark.cloud.40562"  # Use real test principal
}
`
}

// Add tests for: FDS user, AdProxy user, GROUP, ROLE, not found, type filter
// See spec.md "User Scenarios & Testing" for complete test matrix
```

**Next**: Run test to confirm it fails (Red phase of TDD)

---

#### Step 1.2: Create Data Source Structure
**File**: `internal/provider/principal_data_source.go`
**Reference Pattern**: `internal/provider/access_policy_data_source.go:1-43`

```go
package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cyberark/ark-sdk-golang/pkg/services/identity"
	directoriesmodels "github.com/cyberark/ark-sdk-golang/pkg/services/identity/directories/models"
	usersmodels "github.com/cyberark/ark-sdk-golang/pkg/services/identity/users/models"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &PrincipalDataSource{}

func NewPrincipalDataSource() datasource.DataSource {
	return &PrincipalDataSource{}
}

// PrincipalDataSource defines the data source implementation
type PrincipalDataSource struct {
	providerData *ProviderData
}

// PrincipalDataSourceModel describes the data source data model
// Reference: data-model.md "Go Struct" section
type PrincipalDataSourceModel struct {
	// Input attributes
	Name types.String `tfsdk:"name"`  // Required: SystemName
	Type types.String `tfsdk:"type"`  // Optional: USER/GROUP/ROLE filter

	// Computed attributes
	ID            types.String `tfsdk:"id"`              // Principal UUID
	PrincipalType types.String `tfsdk:"principal_type"`  // USER/GROUP/ROLE
	DirectoryName types.String `tfsdk:"directory_name"`  // Localized directory name
	DirectoryID   types.String `tfsdk:"directory_id"`    // Directory UUID
	DisplayName   types.String `tfsdk:"display_name"`    // Human-readable name
	Email         types.String `tfsdk:"email"`           // Email (optional, users only)
	Description   types.String `tfsdk:"description"`     // Description (optional)
}
```

---

#### Step 1.3: Implement Metadata Method
**Reference Pattern**: `internal/provider/access_policy_data_source.go:40-42`

```go
func (d *PrincipalDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_principal"
}
```

---

#### Step 1.4: Implement Schema Method
**Reference**: `data-model.md` "Terraform Schema" and "Validation Rules" sections

```go
func (d *PrincipalDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a principal (user, group, or role) by name and returns all information needed for policy assignments. " +
			"Supports Cloud Directory (CDS), Federated Directory (FDS), and Active Directory (AdProxy) principals.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The principal's system name (e.g., `user@domain.com` for users, or group/role name). " +
					"Matching is case-insensitive.",
				Required: true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Optional filter by principal type: `USER`, `GROUP`, or `ROLE`. " +
					"If omitted, searches all types.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf("USER", "GROUP", "ROLE"),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The principal's UUID.",
				Computed:            true,
			},
			"principal_type": schema.StringAttribute{
				MarkdownDescription: "The principal type: `USER`, `GROUP`, or `ROLE`.",
				Computed:            true,
			},
			"directory_name": schema.StringAttribute{
				MarkdownDescription: "The localized, human-readable directory name " +
					"(e.g., 'CyberArk Cloud Directory', 'Federation with company.com', 'Active Directory (domain.com)').",
				Computed: true,
			},
			"directory_id": schema.StringAttribute{
				MarkdownDescription: "The directory service UUID.",
				Computed:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The principal's human-readable display name.",
				Computed:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The principal's email address (users only, may be empty for groups/roles).",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The principal's description (optional, may be empty).",
				Computed:            true,
			},
		},
	}
}
```

**Reference**: Add validator import:
```go
import "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
```

---

#### Step 1.5: Implement Configure Method
**Reference Pattern**: `internal/provider/access_policy_data_source.go:73-88`

```go
func (d *PrincipalDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.providerData = providerData
}
```

---

#### Step 1.6: Implement Read Method (Core Logic)
**References**:
- `contracts/identity-apis.md` - API behavior and hybrid strategy
- `data-model.md` "Field Mappings" - SDK to Terraform field mapping
- `docs/development/principal-lookup-investigation.md` - Validated implementation

```go
func (d *PrincipalDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PrincipalDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	principalName := data.Name.ValueString()
	principalType := data.Type.ValueString()

	tflog.Debug(ctx, "Looking up principal", map[string]interface{}{
		"name": principalName,
		"type": principalType,
	})

	// Initialize Identity API
	// Reference: contracts/identity-apis.md "API 1: UserByName()"
	identityAPI, err := identity.NewArkIdentityAPI(d.providerData.AuthContext.ISPAuth)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to initialize Identity API",
			fmt.Sprintf("Unable to create Identity API client: %v", err),
		)
		return
	}

	// Step 1: Get directory mapping (required for Phase 2)
	// Reference: contracts/identity-apis.md "API 3: ListDirectories()"
	directories, err := identityAPI.Directories().ListDirectories(&directoriesmodels.ArkIdentityListDirectories{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to list directories",
			fmt.Sprintf("Unable to retrieve directory mappings: %v", err),
		)
		return
	}

	dirMap := buildDirectoryMap(directories)
	tflog.Debug(ctx, "Built directory mapping", map[string]interface{}{
		"directory_count": len(dirMap),
	})

	// PHASE 1: Try UserByName() for fast user lookup
	// Reference: contracts/identity-apis.md "API 1: UserByName()"
	// Reference: docs/development/principal-lookup-investigation.md "PoC 4: Optimal Hybrid Solution"
	if principalType == "" || principalType == "USER" {
		user, err := identityAPI.Users().UserByName(&usersmodels.ArkIdentityUserByName{
			Username: principalName,
		})

		if err == nil && user != nil {
			tflog.Debug(ctx, "User found via UserByName() fast path", map[string]interface{}{
				"uuid":         user.UserID,
				"display_name": user.DisplayName,
			})

			// Get directory info by matching UUID in Phase 2
			// Reference: data-model.md "From ARK SDK ArkIdentityUser"
			principal, err := d.getDirectoryInfoByUUID(ctx, identityAPI, user.UserID, dirMap)
			if err == nil {
				populateDataModel(&data, principal)
				tflog.Info(ctx, "Principal lookup completed", map[string]interface{}{
					"name": principalName,
					"type": principal.Type,
					"path": "phase1_fast",
				})
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}

			// Fallback: Return user without directory info (edge case)
			tflog.Warn(ctx, "User found but directory info unavailable", map[string]interface{}{
				"uuid": user.UserID,
			})
			data.ID = types.StringValue(user.UserID)
			data.PrincipalType = types.StringValue("USER")
			data.DisplayName = types.StringValue(user.DisplayName)
			data.Email = types.StringValue(user.Email)
			data.DirectoryName = types.StringNull()
			data.DirectoryID = types.StringNull()
			data.Description = types.StringNull()
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	// PHASE 2: Fallback to ListDirectoriesEntities() with client-side filtering
	// Reference: contracts/identity-apis.md "API 2: ListDirectoriesEntities()"
	tflog.Debug(ctx, "Falling back to ListDirectoriesEntities() scan", map[string]interface{}{
		"reason": "User not found in Phase 1 or type filter is GROUP/ROLE",
	})

	entityTypes := []string{directoriesmodels.User, directoriesmodels.Group, directoriesmodels.Role}
	if principalType != "" {
		entityTypes = []string{principalType}
	}

	entitiesChan, err := identityAPI.Directories().ListDirectoriesEntities(
		&directoriesmodels.ArkIdentityListDirectoriesEntities{
			Search:      "", // CRITICAL: Empty search = get all entities
			EntityTypes: entityTypes,
			PageSize:    10000,
			Limit:       10000,
		},
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to list directory entities",
			fmt.Sprintf("Unable to search for principal: %v", err),
		)
		return
	}

	// Client-side filter for exact SystemName match
	// Reference: data-model.md "Field Mappings" - entity.Name is SystemName
	entitiesScanned := 0
	for page := range entitiesChan {
		for _, entity := range page.Items {
			entitiesScanned++
			entityPtr := *entity

			// Extract principal data from entity
			// Reference: data-model.md "From ARK SDK ArkIdentityUserEntity/GroupEntity/RoleEntity"
			principal := extractPrincipalFromEntity(entityPtr, dirMap)
			if principal == nil {
				continue
			}

			// Case-insensitive exact match on SystemName
			if strings.EqualFold(principal.Name, principalName) {
				tflog.Info(ctx, "Principal lookup completed", map[string]interface{}{
					"name":             principalName,
					"type":             principal.Type,
					"path":             "phase2_fallback",
					"entities_scanned": entitiesScanned,
				})
				populateDataModel(&data, principal)
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
		}
	}

	// Principal not found
	tflog.Error(ctx, "Principal not found", map[string]interface{}{
		"name":             principalName,
		"entities_scanned": entitiesScanned,
	})
	resp.Diagnostics.AddError(
		"Principal Not Found",
		fmt.Sprintf("Principal '%s' not found in any directory. "+
			"Verify the principal name spelling and ensure the principal exists in one of the configured directories.", principalName),
	)
}
```

---

### Phase 2: Helper Functions

#### Step 2.1: Create Helper File
**File**: `internal/provider/helpers/principal_lookup.go`

```go
package helpers

import (
	directoriesmodels "github.com/cyberark/ark-sdk-golang/pkg/services/identity/directories/models"
)

// BuildDirectoryMap creates a mapping from directory type to UUID
// Reference: data-model.md "Directory UUID Mapping Function"
func BuildDirectoryMap(directories []*directoriesmodels.ArkIdentityDirectory) map[string]string {
	dirMap := make(map[string]string)
	for _, dir := range directories {
		dirMap[dir.Directory] = dir.DirectoryServiceUUID
	}
	return dirMap
}
```

**Or** keep it in the data source file as a private function (simpler):

```go
// buildDirectoryMap creates a mapping from directory type (CDS/FDS/AdProxy) to UUID
// Reference: data-model.md "Directory UUID Mapping Function"
func buildDirectoryMap(directories []*directoriesmodels.ArkIdentityDirectory) map[string]string {
	dirMap := make(map[string]string)
	for _, dir := range directories {
		dirMap[dir.Directory] = dir.DirectoryServiceUUID
	}
	return dirMap
}
```

---

#### Step 2.2: Extract Principal Helper
**Reference**: `data-model.md` "Field Mappings" section

```go
// Principal holds the complete principal data for Terraform
type Principal struct {
	ID            string
	Name          string
	Type          string
	DirectoryName string
	DirectoryID   string
	DisplayName   string
	Email         string
	Description   string
}

// extractPrincipalFromEntity converts SDK entity to Principal struct
// Reference: data-model.md "From ARK SDK ArkIdentityUserEntity/GroupEntity/RoleEntity"
func extractPrincipalFromEntity(entity interface{}, dirMap map[string]string) *Principal {
	switch e := entity.(type) {
	case *directoriesmodels.ArkIdentityUserEntity:
		return &Principal{
			ID:            e.ID,
			Name:          e.Name,
			Type:          e.EntityType,
			DirectoryName: e.ServiceInstanceLocalized, // CRITICAL: Use localized name
			DirectoryID:   dirMap[e.DirectoryServiceType], // Map type to UUID
			DisplayName:   e.DisplayName,
			Email:         e.Email,
			Description:   e.Description,
		}
	case *directoriesmodels.ArkIdentityGroupEntity:
		return &Principal{
			ID:            e.ID,
			Name:          e.Name,
			Type:          e.EntityType,
			DirectoryName: e.ServiceInstanceLocalized,
			DirectoryID:   dirMap[e.DirectoryServiceType],
			DisplayName:   e.DisplayName,
			Email:         "", // Groups don't have email
			Description:   e.Description,
		}
	case *directoriesmodels.ArkIdentityRoleEntity:
		return &Principal{
			ID:            e.ID,
			Name:          e.Name,
			Type:          e.EntityType,
			DirectoryName: e.ServiceInstanceLocalized,
			DirectoryID:   dirMap[e.DirectoryServiceType],
			DisplayName:   e.DisplayName,
			Email:         "", // Roles don't have email
			Description:   e.Description,
		}
	default:
		return nil
	}
}
```

---

#### Step 2.3: Populate Data Model Helper
**Reference**: `data-model.md` "Output Validation" section

```go
// populateDataModel fills Terraform model from Principal data
// Reference: data-model.md "Output Validation" - handle optional fields
func populateDataModel(data *PrincipalDataSourceModel, principal *Principal) {
	data.ID = types.StringValue(principal.ID)
	data.PrincipalType = types.StringValue(principal.Type)
	data.DirectoryName = types.StringValue(principal.DirectoryName)
	data.DirectoryID = types.StringValue(principal.DirectoryID)
	data.DisplayName = types.StringValue(principal.DisplayName)

	// Optional fields - use null if empty
	if principal.Email != "" {
		data.Email = types.StringValue(principal.Email)
	} else {
		data.Email = types.StringNull()
	}

	if principal.Description != "" {
		data.Description = types.StringValue(principal.Description)
	} else {
		data.Description = types.StringNull()
	}
}
```

---

#### Step 2.4: Get Directory Info by UUID Helper
**Reference**: `docs/development/principal-lookup-investigation.md` "PoC 4" section

```go
// getDirectoryInfoByUUID matches a user UUID to get directory information
// This is used after Phase 1 UserByName() to enrich user data with directory info
func (d *PrincipalDataSource) getDirectoryInfoByUUID(
	ctx context.Context,
	api *identity.ArkIdentityAPI,
	userUUID string,
	dirMap map[string]string,
) (*Principal, error) {

	entitiesChan, err := api.Directories().ListDirectoriesEntities(
		&directoriesmodels.ArkIdentityListDirectoriesEntities{
			Search:      "",
			EntityTypes: []string{directoriesmodels.User},
			PageSize:    10000,
			Limit:       10000,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	for page := range entitiesChan {
		for _, entity := range page.Items {
			e, ok := (*entity).(*directoriesmodels.ArkIdentityUserEntity)
			if !ok {
				continue
			}
			if e.ID == userUUID {
				return &Principal{
					ID:            e.ID,
					Name:          e.Name,
					Type:          e.EntityType,
					DirectoryName: e.ServiceInstanceLocalized,
					DirectoryID:   dirMap[e.DirectoryServiceType],
					DisplayName:   e.DisplayName,
					Email:         e.Email,
					Description:   e.Description,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("directory info not found for UUID: %s", userUUID)
}
```

---

### Phase 3: Provider Registration

**File**: `internal/provider/provider.go`
**Reference Pattern**: `internal/provider/provider.go` (existing DataSources() method)

```go
func (p *CyberArkSIAProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAccessPolicyDataSource, // Existing
		NewPrincipalDataSource,    // ADD THIS LINE
	}
}
```

---

### Phase 4: Complete Test Suite

**File**: `internal/provider/principal_data_source_test.go`
**Reference**: `spec.md` "User Scenarios & Testing" for complete test matrix

Add tests for:
1. ✅ TestAccPrincipalDataSource_CloudUser (already created)
2. TestAccPrincipalDataSource_FederatedUser
3. TestAccPrincipalDataSource_ADUser
4. TestAccPrincipalDataSource_Group
5. TestAccPrincipalDataSource_Role
6. TestAccPrincipalDataSource_NotFound (error case)
7. TestAccPrincipalDataSource_TypeFilter
8. TestAccPrincipalDataSource_WithPolicyAssignment (integration test)

---

### Phase 5: Documentation

#### Step 5.1: Create Data Source Documentation
**File**: `docs/data-sources/principal.md`
**Reference**: `quickstart.md` for examples

#### Step 5.2: Create Examples
**File**: `examples/data-sources/cyberarksia_principal/data-source.tf`
**Reference**: `quickstart.md` "Complete Example" section

---

## Validation Checklist

Before marking implementation complete:

- [ ] All tests pass (TF_ACC=1 go test)
- [ ] Run against all 3 directory types (CDS, FDS, AdProxy)
- [ ] Run against all 3 principal types (USER, GROUP, ROLE)
- [ ] User lookup < 1 second (Phase 1 fast path)
- [ ] Group/role lookup < 2 seconds (Phase 2 fallback)
- [ ] Principal not found returns clear error
- [ ] No sensitive data logged (check tflog output)
- [ ] Documentation includes all examples from quickstart.md
- [ ] Code follows existing provider patterns (compare to access_policy_data_source.go)

---

## Common Issues & Solutions

### Issue 1: Type Assertion Panics
**Symptom**: Runtime panic when type asserting entity interface
**Solution**: Use comma-ok pattern:
```go
e, ok := (*entity).(*directoriesmodels.ArkIdentityUserEntity)
if !ok {
    continue // Skip this entity
}
```

### Issue 2: Empty Directory Info
**Symptom**: DirectoryName or DirectoryID is empty
**Solution**: Verify ServiceInstanceLocalized is being used (NOT DirectoryServiceType)
**Reference**: `data-model.md` "Directory Name Mapping (CLARIFIED)" section

### Issue 3: Case Sensitivity Failures
**Symptom**: Principal not found when name casing differs
**Solution**: Use `strings.EqualFold()` for case-insensitive matching
**Reference**: `data-model.md` "Input Validation" section

---

## Quick Reference

**Key Files**:
- Data Model: `data-model.md`
- API Contracts: `contracts/identity-apis.md`
- Investigation: `docs/development/principal-lookup-investigation.md`
- Quick Start: `quickstart.md`
- Pattern: `internal/provider/access_policy_data_source.go`

**Key Concepts**:
- Hybrid Strategy: Phase 1 (UserByName) → Phase 2 (ListDirectoriesEntities)
- SystemName: Unique principal identifier (e.g., "user@domain.com")
- ServiceInstanceLocalized: Human-readable directory name (use for source_directory_name)
- DirectoryServiceType: SDK type code (use for UUID mapping only)
