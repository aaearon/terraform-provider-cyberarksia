# Research Findings: Target Set Resource

**Date**: 2025-11-08
**Feature**: Target Set Resource (`cyberarksia_target_set`)
**Status**: Complete - No unresolved unknowns

## Overview

All technical unknowns were pre-resolved through comprehensive PoC investigation documented in `docs/development/target-sets-poc-investigation.md`. This research phase consolidates findings relevant to implementation planning.

## Key Decisions

### Decision 1: DELETE Operation Workaround

**Decision**: Use direct HTTP DELETE call bypassing ARK SDK

**Rationale**:
- ARK SDK v1.5.0 `DeleteTargetSet()` method has nil body pointer bug causing panic
- Bug confirmed in production CyberArk `ark` CLI (same codebase)
- Panic occurs before API call is made, preventing cleanup
- Workaround pattern already established in `internal/client/delete_workarounds.go` for other resources

**Alternatives Considered**:
- Wait for SDK v1.6.0+ fix: **Rejected** - timeline unknown, blocks feature delivery
- Fork and patch SDK: **Rejected** - maintenance burden, dependency management complexity
- Implement retry logic around panic: **Rejected** - panic cannot be caught in Go, fundamentally broken

**Implementation**:
```go
func DeleteTargetSetDirect(ctx context.Context, auth *ISPAuthContext, name string) error {
    client := isp.FromISPAuth(auth.ISPAuth, "dpa", ".", "", nil)
    response, err := client.Delete(ctx, fmt.Sprintf("/api/targetsets/%s", name), map[string]string{})
    // Check response.StatusCode == 204
}
```

**Evidence**: 45 successful deletions via direct API call in PoC validation (100% success rate)

---

### Decision 2: Prevent Clearing provision_format

**Decision**: Implement custom plan modifier that errors at plan time when user attempts to clear `provision_format`

**Rationale**:
- API uses PATCH semantics - server preserves omitted fields
- Sending empty string or omitting field both result in existing value being preserved
- Attempting to clear creates perpetual drift (plan shows empty, state shows preserved value)
- Audit trail consistency requires stable naming format once established

**Alternatives Considered**:
- Silent preservation: **Rejected** - causes confusing drift behavior, user doesn't understand why config isn't applied
- Computed preservation (UseStateForUnknown): **Rejected** - makes field appear "known after apply" which is misleading
- No special handling: **Rejected** - poor UX, mysterious drift

**Implementation**: Custom plan modifier that:
1. Checks if state has non-empty value
2. Checks if plan has null/empty value
3. Errors with clear message: "Cannot clear provision_format once set due to API limitations. You can update it to a different value, but cannot clear it entirely."

**Evidence**: PoC test confirmed both empty string and field omission preserve existing value

---

### Decision 3: Name Validation Strategy

**Decision**: Add custom validator that warns (not errors) when name contains forward slashes

**Rationale**:
- API accepts names with forward slashes during CREATE (200 OK)
- DELETE fails with 403 Forbidden due to URL path interpretation issues
- DELETE workaround also fails (URL encoding doesn't resolve issue)
- User can create resource but cannot destroy it (Terraform state cleanup impossible)

**Alternatives Considered**:
- Hard error on forward slashes: **Rejected** - API technically accepts them, creates provider/API mismatch
- No validation: **Rejected** - poor UX, user discovers issue only during destroy
- URL encoding in DELETE: **Rejected** - tested in PoC, still returns 403

**Implementation**: Validator that shows warning:
```
Warning: Name contains forward slashes which will cause deletion failures (403 errors).
While the API accepts this during creation, you will not be able to destroy this resource.
Consider using hyphens or underscores instead.
```

**Evidence**: PoC test created `enhanced/with/slashes/1762585058` successfully but DELETE returned 403

---

### Decision 4: No Pre-flight Secret Validation

**Decision**: Do not validate `secret_id` exists before creating target set

**Rationale**:
- API explicitly allows creation with non-existent `secret_id` (field has `omitempty` tag, no referential integrity)
- Terraform's dependency graph handles ordering naturally via resource references
- Pre-flight validation adds latency and potential race conditions
- Target set is functionally usable - validation happens at JIT access time, not config time
- Supports valid edge cases: import before secret exists, secret rotation workflows

**Alternatives Considered**:
- Pre-flight GET request to verify secret: **Rejected** - extra API call, race conditions, doesn't prevent deletion after creation
- Soft validation (warning): **Rejected** - creates noise for valid forward-reference scenarios
- Make field required in schema: **Accepted** - provider enforces as required (target set non-functional without it)

**Implementation**:
- Schema: Mark `secret_id` as Required (target set non-functional without it)
- No pre-flight API validation
- Examples show reference pattern: `secret_id = cyberarksia_virtual_machine_secret.admin.id`
- Documentation includes: "Reference cyberarksia_virtual_machine_secret.example.id for proper dependency ordering"

**Evidence**: PoC created `test-no-secrets-1762536105` without secret_id, API accepted (200 OK)

---

### Decision 5: Handle All Field Mutability

**Decision**: Allow in-place updates for ALL fields (no ForceNew on any attribute)

**Rationale**:
- PoC validated all 6 type change combinations (Target↔Domain↔Suffix bidirectional)
- API supports credential rotation (`secret_id` and `secret_type` updates)
- Rename supported via UPDATE with new name in body + old name in URL
- All fields mutable despite UI blocking some changes

**Alternatives Considered**:
- ForceNew on type changes: **Rejected** - API supports in-place updates, PoC validated all combinations
- ForceNew on credential changes: **Rejected** - prevents credential rotation use case
- ForceNew on renames: **Rejected** - API handles renames, ID follows name seamlessly

**Implementation**:
- No `RequiresReplace` plan modifiers on any field
- UPDATE method handles rename by using `state.Name` (old) in URL, `plan.Name` (new) in body
- UPDATE method always includes `name` field to avoid destructive API bug

**Evidence**:
- PoC test `test-type-change-1762536105` changed Target → Domain successfully
- PoC validated all 6 type combinations
- Chained renames tested (A→B→C→A, old names return 404)

---

## Best Practices Applied

### Terraform Plugin Framework Patterns

**Pattern**: Schema Definition
- Required vs Optional vs Computed attributes clearly distinguished
- Default values via `stringdefault.StaticString()` and `booldefault.StaticBool()`
- Validators via `stringvalidator.OneOf()` for enums
- Custom validators implement `validator.String` interface

**Pattern**: CRUD Methods
- All methods use `ctx context.Context` for cancellation support
- Error handling via `resp.Diagnostics.AddError()` with user-friendly messages
- State management via `req.State.Get()` and `resp.State.Set()`
- Retry logic via `client.RetryWithBackoff()` wrapper

**Pattern**: Import Support
- `ImportState` uses `resource.ImportStatePassthroughID()` for name-based ID
- ID computed to match name in same transaction

**Source**: Existing resources (`cyberarksia_virtual_machine_secret`, `cyberarksia_database_workspace`)

### Error Handling

**Pattern**: Drift Detection
- Read() method handles 404 as deleted resource (removes from state)
- Non-404 errors propagate to user with actionable messages
- All API errors wrapped with context via `client.MapError()`

**Pattern**: User-Facing Errors
- Include what failed: "Failed to create target set"
- Include why it failed: API error message
- Include guidance when possible: "Check that VM secret exists"

**Source**: `internal/client/errors.go` and `internal/client/retry.go`

### Testing Strategy

**Pattern**: Acceptance Tests
- Test against real SIA API with `TF_ACC=1`
- Basic CRUD lifecycle test (create, read, update, delete)
- Rename test (verify ID follows name change)
- Type change test (verify no ForceNew)
- Import test (verify ImportState accuracy)
- Drift detection test (manual deletion detected)

**Pattern**: CRUD Validation
- Follow `examples/testing/TESTING-GUIDE.md` workflow
- Copy template to `/tmp`, customize with test data
- Run CREATE → READ → UPDATE → DELETE cycle
- Verify validation_summary outputs

**Source**: Constitution Principle I (Test-Driven Development)

---

## Technology Stack Validation

### Go 1.25.0

**Validation**: Current project already using Go 1.25.0, no migration needed

**Relevant Features**:
- Generics support (Go 1.18+) for type-safe validators
- Error wrapping with `%w` verb (Go 1.13+)
- Context cancellation (standard library)

### Terraform Plugin Framework v6

**Validation**: Project currently uses v1.16.1

**Relevant Features**:
- `schema.Schema` for resource definition
- `resource.Resource` interface with CRUD methods
- Plan modifiers via `planmodifier.String` interface
- Validators via `validator.String` interface
- State management via typed models with `tfsdk` tags

**Documentation**: https://developer.hashicorp.com/terraform/plugin/framework

### ARK SDK v1.5.0

**Validation**: Project currently uses v1.5.0

**Known Bugs**:
- DELETE panic (nil body pointer) - workaround required
- UPDATE without name field causes deletion - field inclusion required

**Relevant Services**:
- `siaAPI.WorkspacesTargetSets()` - target set management
- `TargetSet(getTargetSet *models.ArkSIAGetTargetSet)` - read
- `AddTargetSet(addTargetSet *models.ArkSIAAddTargetSet)` - create
- `UpdateTargetSet(updateTargetSet *models.ArkSIAUpdateTargetSet)` - update
- `DeleteTargetSet()` - **DO NOT USE** - panic bug
- `ListTargetSets()` - list all target sets

**Source**: `docs/development/target-sets-poc-investigation.md` lines 50-52, 295-333

---

## Integration Points

### Upstream Dependency: VM Secrets

**Resource**: `cyberarksia_virtual_machine_secret`
**Relationship**: Target sets reference VM secrets via `secret_id` (UUID)

**Integration Pattern**:
```hcl
resource "cyberarksia_virtual_machine_secret" "admin" {
  name        = "admin-credentials"
  secret_type = "ProvisionerUser"
  username    = "admin"
  password    = var.admin_password
}

resource "cyberarksia_target_set" "production" {
  name        = "prod.example.com"
  type        = "Domain"
  secret_id   = cyberarksia_virtual_machine_secret.admin.id  # Dependency
  secret_type = cyberarksia_virtual_machine_secret.admin.secret_type
}
```

**Dependency Handling**:
- Terraform automatically orders creation (secret before target set)
- No pre-flight validation needed (Terraform handles this)
- Examples demonstrate proper reference pattern

### Downstream Dependency: VM Access Policies

**Resource**: `cyberarksia_vm_policy` (future implementation)
**Relationship**: Access policies reference target sets by name

**Integration Pattern**:
```hcl
resource "cyberarksia_vm_policy" "production_access" {
  name = "Production Server Access"
  # References target set by name
  target_sets = [cyberarksia_target_set.production.name]
}
```

**Consideration**: Target set deletion may invalidate policies (API doesn't enforce referential integrity)

---

## Open Questions

**None** - All technical unknowns resolved through comprehensive PoC investigation.

---

## References

- **Primary Source**: `docs/development/target-sets-poc-investigation.md`
  - 50 automated tests (98% pass rate)
  - API behavior documentation (lines 421-467)
  - Field mutability matrix (lines 366-390)
  - SDK bugs and workarounds (lines 492-541)

- **Pattern Sources**:
  - `internal/provider/virtual_machine_secret_resource.go` (similar resource structure)
  - `internal/provider/database_workspace_resource.go` (CRUD patterns)
  - `internal/client/delete_workarounds.go` (DELETE workaround pattern)

- **Constitution**: `.specify/memory/constitution.md` (Principles I-VIII)

- **Development Guide**: `CLAUDE.md` (Quick Start, Project Structure, Common Workflows)

---

**Research Status**: ✅ COMPLETE
**Unresolved Items**: 0
**Next Phase**: Design & Contracts (data-model.md, contracts/, quickstart.md)
