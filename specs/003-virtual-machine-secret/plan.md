# Implementation Plan: Virtual Machine Secret Management

**Branch**: `003-virtual-machine-secret` | **Date**: 2025-11-02 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-virtual-machine-secret/spec.md`

## Summary

Implement Terraform resource `cyberarksia_virtual_machine_secret` for managing VM/server credentials in CyberArk SIA. Supports two credential types: ProvisionerUser (username/password stored in SIA) and PCloudAccount (references to PAM vault accounts). Follows established patterns from `cyberarksia_database_secret` for consistency, including ValidateConfig for conditional field validation, Delete Workarounds for SDK bugs, and RetryWithBackoff for API resilience. Full CRUD lifecycle with Terraform import support enables brownfield adoption.

## Technical Context

**Language/Version**: Go 1.25.0
**Primary Dependencies**:
- ARK SDK v1.5.0 (`github.com/cyberark/ark-sdk-golang`)
- Terraform Plugin Framework v1.16.1 (Plugin Framework v6)
- Terraform Plugin Log v0.9.0

**Storage**: N/A (stateless provider, Terraform state management)
**Testing**:
- Acceptance tests with TF_ACC=1 against real SIA API
- CRUD validation per `examples/testing/TESTING-GUIDE.md`

**Target Platform**: Linux/macOS/Windows (Terraform provider binary)
**Project Type**: Single (Terraform provider plugin)
**Performance Goals**:
- API operations: <5 seconds per call (standard Terraform provider expectations)

**Constraints**:
- ARK SDK v1.5.0 DELETE panic bug (requires workaround)
- ARK SDK v1.5.0 VM secrets filtering bug (ListSecretsBy broken)
- Passwords write-only (API never returns them)
- secret_type immutable (ForceNew required on change)
- 15-minute token expiration (SDK handles refresh automatically)

**Scale/Scope**:
- Single resource: `cyberarksia_virtual_machine_secret`
- Expected usage: 10-100 secrets per Terraform workspace
- No pagination needed (SIA API returns all secrets)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: Test-Driven Development ✅

- **Status**: COMPLIANT
- **Evidence**: Acceptance tests planned per TESTING-GUIDE.md, CRUD validation required
- **Action**: Phase 1 will include test structure in quickstart.md

### Principle II: Peer Review & Validation ✅

- **Status**: COMPLIANT
- **Evidence**: Plan will be reviewed by Codex before implementation
- **Action**: Submit plan + spec to Codex via `mcp__zen__clink` after Phase 1

### Principle III: Pattern Reuse & Consistency ✅

- **Status**: COMPLIANT
- **Evidence**:
  - Follows `internal/provider/database_secret_resource.go` (database secrets) as template
  - Uses Delete Workarounds (`internal/client/delete_workarounds.go`)
  - Uses RetryWithBackoff (`internal/client/retry.go`)
  - Uses ValidateConfig pattern for conditional required fields
  - Naming: `virtual_machine_secret` (full words, not `vm_secret`)
- **Action**: Document pattern mapping in research.md

### Principle IV: SDK Constraint Awareness ✅

- **Status**: COMPLIANT
- **Evidence**:
  - DELETE panic bug workaround planned (FR-012)
  - VM filtering bug documented (use `ListSecrets()`, not `ListSecretsBy()`)
  - Password write-only behavior explicit (FR-004)
- **Action**: Research SDK methods in Phase 0, document workarounds

### Principle V: Documentation Synchronization ✅

- **Status**: COMPLIANT
- **Evidence**: Plan includes:
  - Examples directory: `examples/resources/cyberarksia_virtual_machine_secret/`
  - TESTING-GUIDE.md updates planned
  - tfplugindocs generation in acceptance criteria
- **Action**: Phase 1 quickstart.md will document example configurations

### Principle VI: Git Workflow Discipline ✅

- **Status**: COMPLIANT
- **Evidence**: Feature branch `003-virtual-machine-secret` created, PR workflow planned
- **Action**: Create PR after implementation with Codex review

### Principle VII: Incremental Delivery ✅

- **Status**: COMPLIANT
- **Evidence**: 5 user stories prioritized (P1/P2/P3), independently testable
- **Action**: Tasks.md will break down by user story (Phase 2 - /speckit.tasks)

### Principle VIII: Security & Sensitive Data ✅

- **Status**: COMPLIANT
- **Evidence**:
  - `provisioner_password` marked `Sensitive: true` (FR-003)
  - Passwords never returned from API (FR-004)
  - No password logging planned
- **Action**: Validate sensitive attribute in acceptance tests

**Gate Result**: ✅ **PASS** - All principles compliant, proceed to Phase 0 research

## Project Structure

### Documentation (this feature)

```text
specs/003-virtual-machine-secret/
├── spec.md              # Feature specification (/speckit.specify output)
├── plan.md              # This file (/speckit.plan output)
├── research.md          # Phase 0 output (technology decisions, SDK methods)
├── data-model.md        # Phase 1 output (secret entity, attributes, validation)
├── quickstart.md        # Phase 1 output (example configurations, testing)
├── contracts/           # Phase 1 output (SDK API mappings)
│   └── sdk-methods.md   # ARK SDK SecretsVM API reference
├── checklists/          # Quality validation
│   └── requirements.md  # Spec validation checklist (already complete)
└── tasks.md             # Phase 2 output (/speckit.tasks - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/provider/
├── virtual_machine_secret_resource.go        # NEW: Main resource implementation
├── virtual_machine_secret_resource_test.go   # NEW: Acceptance tests
├── secret_resource.go                        # REFERENCE: Database secret pattern
└── provider.go                                # UPDATE: Register new resource

internal/client/
├── delete_workarounds.go                     # UPDATE: Add DeleteVMSecretDirect()
├── retry.go                                   # REUSE: RetryWithBackoff
└── errors.go                                  # REUSE: MapError

examples/resources/
└── cyberarksia_virtual_machine_secret/       # NEW: Example configurations
    ├── resource.tf                            # Basic ProvisionerUser example
    └── import.sh                              # Import example

examples/testing/
├── TESTING-GUIDE.md                          # UPDATE: Add VM secret test template
└── crud-test-vm-secret.tf                    # NEW: CRUD validation template

docs/
├── resources/
│   └── cyberarksia_virtual_machine_secret.md # NEW: Generated by tfplugindocs
└── development/
    └── vm-secret-implementation.md           # NEW: Implementation summary
```

**Structure Decision**: Single project (Terraform provider). All new code in `internal/provider/` following existing resource patterns. Examples follow established directory layout per `examples/resources/cyberarksia_database_secret/`.

## Complexity Tracking

> No violations to justify - all Constitution principles compliant.

---

## Phase 0: Research & Technology Decisions

### Research Tasks

1. **ARK SDK VM Secrets API Investigation**
   - Method signatures for `SecretsVM()` service
   - `AddSecret()`, `ChangeSecret()`, `DeleteSecret()`, `Secret()`, `ListSecrets()`
   - Data structures: `ArkSIAVMSecret`, `ArkSIAVMAddSecret`, `ArkSIAVMChangeSecret`
   - Verify DELETE panic bug presence
   - Verify ListSecretsBy() filtering bug
   - Document workarounds needed

2. **Pattern Extraction from Database Secret Resource**
   - Resource struct using `*ProviderData` (not `*client.CyberArkClient`)
   - Configure method with type assertion pattern
   - Schema structure for two secret types
   - ValidateConfig implementation for conditional required fields (implements resource.ResourceWithValidateConfig)
   - Sensitive attribute handling (`Sensitive: true` + write-only API behavior)
   - UseStateForUnknown() plan modifier for computed fields (id, secret_id)
   - Password state preservation during Read (API doesn't return them, keep existing state values)
   - ForceNew planmodifier for immutable fields
   - Import implementation pattern

3. **Delete Workaround Strategy**
   - Current workaround pattern in `delete_workarounds.go` (isp.FromISPAuth pattern)
   - API endpoint for VM secret deletion: `/api/v1/sia/secrets/vm/{secret_id}` (verify with SDK source)
   - Service name: "dpa" (same as database secrets, verify during research)
   - HTTP DELETE with empty body `map[string]string{}`
   - Error handling: 404 as success, log responses

4. **Testing Framework Setup**
   - Acceptance test structure from `secret_resource_test.go`
   - TESTING-GUIDE.md template requirements
   - Test data setup (ProvisionerUser vs PCloudAccount)
   - Cleanup strategies

### Decisions to Document in research.md

- **SDK Service**: Use `ArkSIASecretsVMService` accessed via `SecretsVM()`
- **Delete Strategy**: Implement `DeleteVMSecretDirect()` in `delete_workarounds.go`
- **Filtering Workaround**: Use `ListSecrets()` only, implement client-side filtering if needed
- **Schema Pattern**: Two secret types with conditional required fields (mirroring database secrets)
- **Testing Approach**: Real API acceptance tests, no mocking (per constitution)

---

## Phase 1: Design & Contracts

### Data Model (data-model.md)

**Primary Entity**: `VirtualMachineSecret`

**Attributes**:
- `id` (string, computed) - Terraform resource identifier (equals secret_id)
- `secret_id` (string, computed) - SIA UUID, immutable
- `secret_name` (string, required) - User-facing label, mutable
- `secret_type` (string, required, ForceNew) - Enum: "ProvisionerUser", "PCloudAccount"

**Conditional Attributes (ProvisionerUser)**:
- `provisioner_username` (string, required if secret_type==ProvisionerUser)
- `provisioner_password` (string, required if secret_type==ProvisionerUser, sensitive, write-only)

**Conditional Attributes (PCloudAccount)**:
- `pcloud_safe_name` (string, required if secret_type==PCloudAccount)
- `pcloud_account_name` (string, required if secret_type==PCloudAccount)

**Validation Rules**:
- secret_type: Must be "ProvisionerUser" or "PCloudAccount"
- provisioner_password: Min length 8 characters (assume standard password policy)
- secret_name: Max length 200 characters (SIA API constraint)
- ForceNew on secret_type change (immutable after creation)

**State Lifecycle**:
1. **Create**: POST to API → receive secret_id → store in state with all fields
2. **Read**: GET from API by secret_id → update mutable fields (secret_name), **passwords NOT returned by API** (preserve existing values in state)
3. **Update**: PUT to API by secret_id → update mutable fields, **passwords remain in state** (API never returns them)
4. **Delete**: DELETE via workaround by secret_id → remove from state
5. **Import**: GET from API by secret_id → populate state (**passwords absent**, must be manually added or updated)

**Password Handling** (FR-004):
- API is write-only for passwords: `provisioner_password` never returned
- During Read: State preserves existing password value (no API value to overwrite it)
- During Import: Password is absent from state (user must update config to add it)
- Schema uses `Sensitive: true` to hide passwords in plan/apply output

### API Contracts (contracts/sdk-methods.md)

**SDK Package**: `github.com/cyberark/ark-sdk-golang/pkg/services/sia/secrets/vm`

**Service Access**:
```go
siaAPI := arkClient.GetSIAAPI()
vmSecretsService := siaAPI.SecretsVM()  // Returns *ArkSIASecretsVMService
```

**Create Operation**:
```go
// Method: AddSecret(secret *models.ArkSIAVMAddSecret) (*models.ArkSIAVMSecret, error)
// Request: ArkSIAVMAddSecret{
//   SecretName: string
//   SecretType: "ProvisionerUser" | "PCloudAccount"
//   // ProvisionerUser fields:
//   ProvisionerUsername: string (optional, required for ProvisionerUser)
//   ProvisionerPassword: string (optional, required for ProvisionerUser)
//   // PCloudAccount fields:
//   PCloudSafeName: string (optional, required for PCloudAccount)
//   PCloudAccountName: string (optional, required for PCloudAccount)
// }
// Response: ArkSIAVMSecret{ SecretID: string, SecretName: string, SecretType: string, ... }
```

**Read Operation**:
```go
// Method: Secret(secret *models.ArkSIAVMGetSecret) (*models.ArkSIAVMSecret, error)
// Request: ArkSIAVMGetSecret{ SecretID: string }
// Response: ArkSIAVMSecret{ SecretID, SecretName, SecretType, username fields, NO passwords }
```

**Update Operation**:
```go
// Method: ChangeSecret(secret *models.ArkSIAVMChangeSecret) (*models.ArkSIAVMSecret, error)
// Request: ArkSIAVMChangeSecret{ SecretID: string, SecretName: string (optional), credentials... }
// Response: ArkSIAVMSecret (updated)
```

**Delete Operation**:
```go
// Method: DeleteSecret(secret *models.ArkSIAVMDeleteSecret) error
// ⚠️ BROKEN: Causes nil pointer panic, use workaround
// Workaround: DeleteVMSecretDirect(ctx, authContext, secretID string) error
// DELETE /api/v1/sia/secrets/vm/{secretID} with empty body {}
```

**List Operation**:
```go
// Method: ListSecrets() ([]*models.ArkSIAVMSecret, error)
// Returns: Array of all VM secrets (no filtering, use client-side if needed)
// Note: ListSecretsBy() is broken (filtering non-functional), do not use
```

### Quickstart (quickstart.md)

**Example 1: ProvisionerUser Secret**
```hcl
resource "cyberarksia_virtual_machine_secret" "app_server" {
  secret_name = "app-server-admin"
  secret_type = "ProvisionerUser"

  provisioner_username = "admin"
  provisioner_password = "SecurePassword123!"
}

output "secret_id" {
  value = cyberarksia_virtual_machine_secret.app_server.secret_id
}
```

**Example 2: PCloudAccount Secret**
```hcl
resource "cyberarksia_virtual_machine_secret" "vault_ref" {
  secret_name = "production-db-admin"
  secret_type = "PCloudAccount"

  pcloud_safe_name    = "Production-Safe"
  pcloud_account_name = "db-admin-account"
}
```

**Example 3: Import Existing Secret**
```bash
terraform import cyberarksia_virtual_machine_secret.existing abc-123-def-456
```

**Testing Workflow** (per TESTING-GUIDE.md):
1. Create secret with `terraform apply`
2. Verify in SIA UI (secret exists, name matches)
3. Run `terraform plan` (no changes)
4. Update secret_name, apply, verify drift detection
5. Run `terraform destroy`, verify secret deleted

---

## Phase 2: Implementation Outline

**File**: `internal/provider/virtual_machine_secret_resource.go`

**Structure** (following `secret_resource.go` pattern):

1. **Resource Definition**
   ```go
   type virtualMachineSecretResource struct {
       providerData *ProviderData
   }

   func NewVirtualMachineSecretResource() resource.Resource {
       return &virtualMachineSecretResource{}
   }

   // Implements interfaces
   var (
       _ resource.Resource                   = &virtualMachineSecretResource{}
       _ resource.ResourceWithConfigure      = &virtualMachineSecretResource{}
       _ resource.ResourceWithImportState    = &virtualMachineSecretResource{}
       _ resource.ResourceWithValidateConfig = &virtualMachineSecretResource{}
   )
   ```

2. **Schema** (Terraform Plugin Framework v6)
   ```go
   func (r *virtualMachineSecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
       resp.Schema = schema.Schema{
           Attributes: map[string]schema.Attribute{
               "id": schema.StringAttribute{
                   Computed: true,
                   PlanModifiers: []planmodifier.String{
                       stringplanmodifier.UseStateForUnknown(),  // Stability
                   },
               },
               "secret_id": schema.StringAttribute{
                   Computed: true,
                   PlanModifiers: []planmodifier.String{
                       stringplanmodifier.UseStateForUnknown(),
                   },
               },
               "secret_name": schema.StringAttribute{
                   Required: true,
                   Validators: []validator.String{
                       stringvalidator.LengthBetween(1, 200),  // SIA API constraint
                   },
               },
               "secret_type": schema.StringAttribute{
                   Required: true,
                   PlanModifiers: []planmodifier.String{
                       stringplanmodifier.RequiresReplace(),  // ForceNew
                   },
                   Validators: []validator.String{
                       stringvalidator.OneOf("ProvisionerUser", "PCloudAccount"),
                   },
               },
               // Conditional attributes (validated in ValidateConfig)
               "provisioner_username": schema.StringAttribute{Optional: true},
               "provisioner_password": schema.StringAttribute{
                   Optional: true,
                   Sensitive: true,  // FR-003
               },
               "pcloud_safe_name": schema.StringAttribute{Optional: true},
               "pcloud_account_name": schema.StringAttribute{Optional: true},
           },
       }
   }
   ```

3. **Configure Method** (Type assertion pattern)
   ```go
   func (r *virtualMachineSecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
       if req.ProviderData == nil {
           return
       }

       providerData, ok := req.ProviderData.(*ProviderData)
       if !ok {
           resp.Diagnostics.AddError(
               "Unexpected Resource Configure Type",
               fmt.Sprintf("Expected *ProviderData, got: %T", req.ProviderData),
           )
           return
       }

       r.providerData = providerData
   }
   ```

4. **ValidateConfig Method** (Cross-attribute validation per FR-002/FR-010)
   ```go
   func (r *virtualMachineSecretResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
       var config models.VirtualMachineSecretModel
       resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
       if resp.Diagnostics.HasError() {
           return
       }

       secretType := config.SecretType.ValueString()

       switch secretType {
       case "ProvisionerUser":
           if config.ProvisionerUsername.IsNull() {
               resp.Diagnostics.AddAttributeError(
                   path.Root("provisioner_username"),
                   "Missing Required Field",
                   "provisioner_username is required when secret_type=ProvisionerUser",
               )
           }
           if config.ProvisionerPassword.IsNull() {
               resp.Diagnostics.AddAttributeError(
                   path.Root("provisioner_password"),
                   "Missing Required Field",
                   "provisioner_password is required when secret_type=ProvisionerUser",
               )
           }
           // Check PCloud fields not set
           if !config.PCloudSafeName.IsNull() && !config.PCloudSafeName.IsUnknown() {
               resp.Diagnostics.AddAttributeError(
                   path.Root("pcloud_safe_name"),
                   "Invalid Field Combination",
                   "pcloud_safe_name cannot be set when secret_type=ProvisionerUser",
               )
           }
           if !config.PCloudAccountName.IsNull() && !config.PCloudAccountName.IsUnknown() {
               resp.Diagnostics.AddAttributeError(
                   path.Root("pcloud_account_name"),
                   "Invalid Field Combination",
                   "pcloud_account_name cannot be set when secret_type=ProvisionerUser",
               )
           }

       case "PCloudAccount":
           if config.PCloudSafeName.IsNull() {
               resp.Diagnostics.AddAttributeError(
                   path.Root("pcloud_safe_name"),
                   "Missing Required Field",
                   "pcloud_safe_name is required when secret_type=PCloudAccount",
               )
           }
           if config.PCloudAccountName.IsNull() {
               resp.Diagnostics.AddAttributeError(
                   path.Root("pcloud_account_name"),
                   "Missing Required Field",
                   "pcloud_account_name is required when secret_type=PCloudAccount",
               )
           }
           // Check Provisioner fields not set
           if !config.ProvisionerUsername.IsNull() && !config.ProvisionerUsername.IsUnknown() {
               resp.Diagnostics.AddAttributeError(
                   path.Root("provisioner_username"),
                   "Invalid Field Combination",
                   "provisioner_username cannot be set when secret_type=PCloudAccount",
               )
           }
           if !config.ProvisionerPassword.IsNull() && !config.ProvisionerPassword.IsUnknown() {
               resp.Diagnostics.AddAttributeError(
                   path.Root("provisioner_password"),
                   "Invalid Field Combination",
                   "provisioner_password cannot be set when secret_type=PCloudAccount",
               )
           }
       }
   }
   ```

5. **CRUD Methods**
   - `Create()`: Call `SecretsVM().AddSecret()`, wrap with `RetryWithBackoff`, handle errors with `MapError`
   - `Read()`: Call `SecretsVM().Secret()`, handle 404 as deleted (drift detection), preserve password in state
   - `Update()`: Call `SecretsVM().ChangeSecret()`, only mutable fields
   - `Delete()`: Call `client.DeleteVMSecretDirect()` (workaround), not SDK method
   - `ImportState()`: Extract secret_id from import ID, call Read()

6. **Delete Workaround** (`internal/client/delete_workarounds.go`)
   ```go
   // DeleteVMSecretDirect bypasses SDK's buggy DeleteSecret() method for VM secrets
   func DeleteVMSecretDirect(ctx context.Context, authCtx *ISPAuthContext, secretID string) error {
       tflog.Debug(ctx, "Executing DELETE workaround (ARK SDK bug bypass)", map[string]interface{}{
           "resource_type": "virtual_machine_secret",
           "secret_id":     secretID,
           "workaround":    "empty_map_body",
       })

       // Create ISP client (same pattern as DeleteSecretDirect)
       client, err := isp.FromISPAuth(
           authCtx.ISPAuth,
           "dpa", // Service name (TODO: verify "dpa" vs "sia" during research)
           ".",   // Separator
           "",    // Base path
           nil,   // No refresh callback
       )
       if err != nil {
           return fmt.Errorf("failed to create ISP client for DELETE: %w", err)
       }

       // Construct endpoint (TODO: verify exact path with SDK source)
       endpoint := fmt.Sprintf("/api/v1/sia/secrets/vm/%s", secretID)

       // Execute DELETE with empty body workaround
       response, err := client.Delete(ctx, endpoint, map[string]string{})
       if err != nil {
           return fmt.Errorf("failed to delete VM secret %s: %w", secretID, err)
       }
       defer response.Body.Close()

       // Handle 404 as success (idempotent)
       if response.StatusCode == http.StatusNotFound {
           tflog.Debug(ctx, "VM secret already deleted (404)")
           return nil
       }

       if response.StatusCode != http.StatusNoContent {
           return fmt.Errorf("failed to delete VM secret %s - [%d] - [%s]",
               secretID, response.StatusCode, common.SerializeResponseToJSON(response.Body))
       }

       tflog.Debug(ctx, "DELETE workaround successful")
       return nil
   }
   ```

**Acceptance Tests**: `internal/provider/virtual_machine_secret_resource_test.go`

**Positive (Happy Path) Tests**:
- `TestAccVirtualMachineSecret_ProvisionerUser` (basic CRUD)
- `TestAccVirtualMachineSecret_PCloudAccount` (PAM reference)
- `TestAccVirtualMachineSecret_Update` (name + password changes)
- `TestAccVirtualMachineSecret_Import` (import workflow)
- `TestAccVirtualMachineSecret_ForceNew` (secret_type change triggers recreate)

**Negative (Validation) Tests** (per spec scenarios 1.3/1.4):
- `TestAccVirtualMachineSecret_InvalidSecretType` (rejects invalid secret_type values)
- `TestAccVirtualMachineSecret_MissingProvisionerUsername` (ProvisionerUser without username)
- `TestAccVirtualMachineSecret_MissingProvisionerPassword` (ProvisionerUser without password)
- `TestAccVirtualMachineSecret_MissingPCloudSafeName` (PCloudAccount without safe_name)
- `TestAccVirtualMachineSecret_MissingPCloudAccountName` (PCloudAccount without account_name)
- `TestAccVirtualMachineSecret_InvalidFieldMix` (ProvisionerUser with PCloud fields)
- `TestAccVirtualMachineSecret_SensitiveOutput` (verify passwords not in plan/state output)

---

## Next Steps

1. ✅ **Phase 0 Complete**: Generate `research.md` with SDK investigation
2. ✅ **Phase 1 Complete**: Generate `data-model.md`, `contracts/sdk-methods.md`, `quickstart.md`
3. ✅ **Codex Review Complete**: Plan reviewed and updated (see Revision History below)
4. ⏭️ **Phase 2**: Run `/speckit.tasks` to generate task breakdown
5. **Implementation**: Run `/speckit.implement` after tasks approval

## Revision History

**2025-11-02 - Codex Review Fixes**:
- **Blocker Fix**: Corrected resource struct type from `*client.CyberArkClient` to `*ProviderData`
- **Blocker Fix**: Updated delete workaround signature to match existing `ISPAuthContext` pattern
- **Validation**: Added explicit ValidateConfig implementation plan with cross-attribute validation
- **Testing**: Expanded acceptance tests to include 7 negative/validation test cases
- **Consistency**: Removed incorrect Profile Factory references (not applicable to VM secrets)
- **Clarity**: Enhanced password handling documentation (write-only API, state preservation)

## Constitution Re-Check (Post-Design)

All principles remain compliant after Phase 1 design:

- ✅ **TDD**: Test structure documented in quickstart.md
- ✅ **Peer Review**: Plan ready for Codex review
- ✅ **Pattern Reuse**: Database secret pattern confirmed as template
- ✅ **SDK Constraints**: Delete workaround designed, filtering bug documented
- ✅ **Documentation**: Examples and testing workflows planned
- ✅ **Git Workflow**: Feature branch active, PR workflow planned
- ✅ **Incremental Delivery**: User stories map to implementation phases
- ✅ **Security**: Sensitive attributes explicit, write-only password design

**Final Gate**: ✅ **PASS** - Ready for `/speckit.tasks`
