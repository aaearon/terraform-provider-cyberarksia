# Implementation Plan: Database Policy Management - Modular Assignment Pattern

**Branch**: `002-sia-policy-lifecycle` | **Date**: 2025-10-28 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-sia-policy-lifecycle/spec.md`

## Summary

Implement full lifecycle management for CyberArk SIA database access policies through three Terraform resources following a modular assignment pattern:
1. **`cyberarksia_database_policy`** - Policy metadata and access conditions (NEW)
2. **`cyberarksia_database_policy_principal_assignment`** - Principal assignments (NEW)
3. **`cyberarksia_database_policy_assignment`** - Database assignments (existing, consistency updates)

This architecture follows the AWS Security Group Rule pattern, enabling distributed team workflows where security teams manage policies and principals while application teams independently manage their database assignments.

## Technical Context

**Language/Version**: Go 1.25.0
**Primary Dependencies**:
- github.com/cyberark/ark-sdk-golang v1.5.0 (UAP policy API)
- github.com/hashicorp/terraform-plugin-framework v1.16.1 (Plugin Framework v6)
- github.com/hashicorp/terraform-plugin-log v0.9.0

**Storage**: CyberArk SIA UAP API (REST, policy objects with embedded principals/targets)
**Testing**: Acceptance tests (TF_ACC=1), selective unit tests for validators
**Target Platform**: Linux/macOS/Windows Terraform CLI (1.0+)
**Project Type**: Terraform Provider (Go plugin)
**Constraints**:
- UAP service must be provisioned on tenant
- Last-write-wins API behavior (no optimistic locking)
- Single location_type per policy (FQDN/IP ONLY for database policies)
- API constraint: UpdatePolicy() accepts only ONE workspace type in Targets map

**Scale/Scope**:
- 3 new/updated resources
- ~3000 lines of Go code (resources + models + validators)
- ~1500 lines of documentation
- 6 HCL examples per resource type

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Checks from .specify/memory/constitution.md

**Note**: Constitution file does not exist yet. Applying standard Terraform provider best practices:

1. ✅ **Reuse Existing Patterns**: Follow established patterns from `policy_workspace_assignment_resource.go`
2. ✅ **Maximize SDK Usage**: Use ARK SDK v1.5.0 UAP methods (AddPolicy, UpdatePolicy, DeletePolicy, Policy, ListPolicies)
3. ✅ **LLM-Friendly Documentation**: Follow FR-012/FR-013 requirements for AI assistant compatibility
4. ✅ **Test Coverage**: CRUD testing per `examples/testing/TESTING-GUIDE.md`

**Gates**:
- ✅ Does NOT introduce new architecture patterns (uses existing read-modify-write)
- ✅ Does NOT require new dependencies beyond ARK SDK v1.5.0
- ✅ Does NOT deviate from Terraform Plugin Framework v6 conventions
- ✅ Does NOT duplicate logic (shares validators, error handling, logging)

## Project Structure

### Documentation (this feature)

```text
specs/002-sia-policy-lifecycle/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output - ARK SDK API mapping
├── data-model.md        # Phase 1 output - State models for 3 resources
├── quickstart.md        # Phase 1 output - Getting started guide
├── contracts/           # Phase 1 output - ARK SDK API contracts
│   ├── policy-crud.md           # AddPolicy/UpdatePolicy/DeletePolicy/Policy
│   ├── policy-list.md           # ListPolicies pagination
│   └── policy-structure.md      # ArkUAPSIADBAccessPolicy schema
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/
├── provider/                                    # Terraform resource implementations
│   ├── database_policy_resource.go             # NEW - Policy metadata + conditions
│   ├── database_policy_principal_assignment_resource.go  # NEW - Principal assignments
│   ├── policy_workspace_assignment_resource.go  # EXISTING - Update for consistency
│   ├── access_policy_data_source.go            # EXISTING - No changes (already supports name/ID lookup)
│   ├── provider.go                             # UPDATE - Register new resources
│   └── logging.go                              # EXISTING - Reuse (no changes)
│
├── models/                                      # State models
│   ├── database_policy.go                      # NEW - Policy state model
│   ├── policy_principal_assignment.go          # NEW - Principal assignment state
│   └── policy_workspace_assignment.go           # EXISTING - No changes needed
│
├── validators/                                  # Custom validators
│   ├── policy_status_validator.go              # NEW - "Active"|"Suspended" only
│   ├── principal_type_validator.go             # NEW - "USER"|"ROLE"|"GROUP"
│   ├── location_type_validator.go              # NEW - "FQDN/IP" only
│   └── profile_validator.go                    # EXISTING - Reuse for auth methods
│
└── client/                                      # API clients
    ├── uap_client.go                            # EXISTING - Reuse (initialized in provider.go)
    └── sia_client.go                            # EXISTING - No changes

docs/
├── resources/
│   ├── database_policy.md                       # NEW - Policy resource docs
│   ├── database_policy_principal_assignment.md  # NEW - Principal assignment docs
│   └── policy_workspace_assignment.md            # EXISTING - Update for consistency
│
└── data-sources/
    └── access_policy.md                         # EXISTING - No changes needed

examples/
├── resources/
│   ├── cyberarksia_database_policy/             # NEW - 5 examples
│   │   ├── basic.tf                             # Minimal policy
│   │   ├── with-conditions.tf                   # Policy with full conditions
│   │   ├── suspended.tf                         # Suspended policy
│   │   ├── with-tags.tf                         # Policy with tags
│   │   └── complete.tf                          # All options
│   │
│   ├── cyberarksia_database_policy_principal_assignment/  # NEW - 6 examples
│   │   ├── user-azuread.tf                      # USER principal with directory
│   │   ├── user-ldap.tf                         # USER principal LDAP
│   │   ├── group-azuread.tf                     # GROUP principal
│   │   ├── role.tf                              # ROLE principal (no directory)
│   │   ├── multiple-principals.tf               # Multiple assignments
│   │   └── complete.tf                          # All options
│   │
│   └── cyberarksia_database_policy_assignment/  # EXISTING - Update naming for consistency
│       └── [existing examples remain]
│
└── testing/
    ├── TESTING-GUIDE.md                         # UPDATE - Add policy resource testing
    ├── crud-test-policy.tf                      # NEW - Policy CRUD test template
    └── crud-test-principal-assignment.tf        # NEW - Principal assignment CRUD template
```

**Structure Decision**: Follows existing single-project Terraform provider structure. All Go code in `internal/`, documentation in `docs/`, examples in `examples/`. New resources follow established naming: `{type}_{name}_resource.go`.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

*No violations - all gates pass.*

---

## Phase 0: Research & Design Decisions

**Objective**: Map ARK SDK v1.5.0 UAP API to Terraform resources, resolve design decisions, document API contracts.

### Research Tasks

1. **ARK SDK UAP API Structure** (research.md § API Mapping)
   - Document `ArkUAPSIADBAccessPolicy` full hierarchy
   - Map SDK fields to Terraform attributes
   - Identify computed vs configurable fields
   - Document UpdatePolicy() constraint (single workspace type only)

2. **Read-Modify-Write Pattern** (research.md § Implementation Patterns)
   - Document how to preserve UI-managed principals/targets
   - Explain API constraint workaround for UpdatePolicy()
   - Provide code examples from existing `policy_workspace_assignment_resource.go`

   **Algorithm**:
   1. **Fetch**: GET policy by ID (returns complete policy with all principals/targets)
   2. **Locate**: Find our managed element in the fetched policy structure
   3. **Modify**: Update/add/remove only our managed element, preserve all others
   4. **Validate**: Ensure changes don't conflict with API constraints (e.g., single workspace type limit)
   5. **Write**: PUT policy back with modified element using UpdatePolicy()
   6. **Handle conflicts**: Retry on transient failures (409/412), fail with clear message on persistent errors

   **Critical API Constraint**: UpdatePolicy() requires sending full policy metadata (name, status, conditions, principals) but accepts only ONE workspace type in Targets map per call. This means when updating database assignments, send only the modified workspace type (e.g., "FQDN/IP"), not all workspace types.

3. **Composite ID Strategy** (research.md § Import Format)
   - Principal assignment: `policy-id:principal-id:principal-type` (3-part)
   - Database assignment: `policy-id:database-id` (2-part, existing)
   - Justification: Principal IDs can be duplicate across types

4. **Policy Status Management** (research.md § Status Lifecycle)
   - Supported statuses: "Active", "Suspended" (both user-controllable)
   - Default: "Active"
   - Validation: Custom validator (policy_status_validator.go)

5. **ForceNew Attributes** (research.md § Update Behavior)
   - Policy resource: None (policy ID is the unique identifier; all user-configurable attributes support in-place updates; location_type is fixed and not user-modifiable)
   - Principal assignment: `policy_id`, `principal_id`, `principal_type` (changing these means assigning a different principal, which is a new assignment)
   - Database assignment: `policy_id`, `database_workspace_id` (existing - changing these means assigning a different database, which is a new assignment)

6. **Pagination Pattern** (research.md § List Operations)
   - ARK SDK returns `<-chan *ArkUAPDBPolicyPage`
   - Data source already implements (access_policy_data_source.go:99-220)
   - Pattern: `for page := range policyPages { for _, policy := range page.Items {...} }`

### Design Decisions

**Decision 1: Policy Resource Scope**
**Choice**: Metadata + Conditions only (principals/targets via separate assignment resources)
**Rationale**:
- Follows modular assignment pattern (spec requirement)
- Enables distributed team workflows
- Consistent with existing `policy_workspace_assignment` approach

**Decision 2: PolicyEntitlement Attributes**
**Choices**:
- `target_category`: Fixed to "DB" (provider only supports database policies)
- `location_type`: Fixed to "FQDN/IP" (ONLY valid value for database access policies - all database workspaces use this regardless of cloud provider)
- `policy_type`: Fixed to "Recurring" (ONLY valid value for database access policies - user cannot specify this)

**Decision 3: Conditions Schema**
**Choice**: Full conditions support from Phase 1 - all condition attributes available
**Includes**:
- `max_session_duration` (1-24 hours, required)
- `idle_time` (1-120 minutes, optional, default 10)
- `access_window` block (optional):
  - `days_of_the_week` (0=Sunday through 6=Saturday)
  - `from_hour` (HH:MM format)
  - `to_hour` (HH:MM format)

**Decision 4: Principal Composite ID Format**
**Choice**: 3-part ID `policy-id:principal-id:principal-type`
**Rationale**: Principals can have duplicate IDs across types (e.g., user "admin", role "admin")

**Decision 5: Status Validation**
**Choice**: Custom validator allowing only "Active"|"Suspended"
**Rationale**: These are the only two valid status values for database access policies per spec FR-004

---

## Phase 1: Data Models & Contracts

**Objective**: Define Terraform state models, ARK SDK mapping, and API contracts.

### Deliverables

1. **data-model.md** - State models for all 3 resources
   - `DatabasePolicyModel` (PolicyID, Name, Description, Status, LocationType, PolicyType, DelegationClassification, PolicyTags, TimeZone, Conditions)
   - `PolicyPrincipalAssignmentModel` (ID, PolicyID, PrincipalID, PrincipalType, PrincipalName, SourceDirectoryName, SourceDirectoryID)
   - Update `PolicyDatabaseAssignmentModel` (no schema changes, consistency updates only)

2. **contracts/policy-crud.md** - ARK SDK CRUD operations
   - `AddPolicy(*ArkUAPSIADBAccessPolicy) (*ArkUAPSIADBAccessPolicy, error)`
   - `UpdatePolicy(*ArkUAPSIADBAccessPolicy) (*ArkUAPSIADBAccessPolicy, error)`
   - `DeletePolicy(*ArkUAPDeletePolicyRequest) error`
   - `Policy(*ArkUAPGetPolicyRequest) (*ArkUAPSIADBAccessPolicy, error)`

3. **contracts/policy-list.md** - Pagination pattern
   - `ListPolicies() (<-chan *ArkUAPDBPolicyPage, error)`
   - `ListPoliciesBy(*ArkUAPSIADBFilters) (<-chan *ArkUAPDBPolicyPage, error)`
   - Channel iteration pattern with page.Items

4. **contracts/policy-structure.md** - Full ARK SDK schema
   - `ArkUAPSIADBAccessPolicy` hierarchy
   - Field mappings (Terraform attribute → SDK field)
   - Computed fields (PolicyID, CreatedBy, UpdatedOn)
   - HTML encoding methods (EncodeName, EncodeDescription)

5. **quickstart.md** - Getting started guide
   - Prerequisites (UAP service provisioned, credentials)
   - Create policy with conditions
   - Assign principals (USER, GROUP, ROLE)
   - Assign databases
   - Import existing policies
   - Multi-team workflow example

### Agent Context Update

After Phase 1 completion, run:
```bash
.specify/scripts/bash/update-agent-context.sh claude
```

This will update CLAUDE.md with:
- New resource names and purposes
- Key implementation patterns (composite IDs, read-modify-write)
- Validation rules for new validators
- Known limitations (concurrent modification race conditions)

---

## Non-Functional Requirements

### Performance

**Latency**: No explicit requirements - depends on ARK SDK and CyberArk API performance
- CRUD operations: Typical REST API latency (100ms-2s)
- List operations: Pagination handled by SDK transparently
- No provider-level caching (state is source of truth)

**Throughput**: Standard Terraform provider patterns
- Sequential resource operations (Terraform's execution model)
- Parallel operations via count/for_each meta-arguments
- Rate limiting handled by SDK retry logic (FR-033)

**Scalability**:
- Supports large policy counts (pagination via SDK)
- Supports many principals/databases per policy (read-modify-write pattern)
- No artificial limits imposed by provider

### Security

**Credentials**: Follow existing provider authentication patterns
- ISP credentials (username + client_secret) configured at provider level
- No credential logging (standard practice)
- Bearer tokens managed by ARK SDK (15-minute expiration, automatic refresh)

**Sensitive Data**: Terraform Plugin Framework sensitive attributes
- Authentication profiles: Mark as sensitive in schema
- No logging of passwords, tokens, client_secret
- Use `terraform-plugin-log/tflog` with structured logging (excludes sensitive fields)

**Permissions**: Required UAP service permissions (CyberArk-managed)
- `uap:policy:read` - Read policies
- `uap:policy:create` - Create policies
- `uap:policy:update` - Update policies (including principal/database assignments)
- `uap:policy:delete` - Delete policies
- **Note**: Exact permission names inferred - consult CyberArk documentation for official names

**Audit**: Leverage CyberArk SIA audit logs
- Provider does not implement separate audit logging
- All operations tracked by SIA platform (created_by, updated_on computed fields)

### Reliability

**Error Handling**: Comprehensive error mapping (research.md § API Error Handling)
- API-only validation for business rules (time_frame, access_window, name length, tag count per FR-034)
- Client-side validation only for provider constructs (composite IDs, enum validators per FR-036)
- Retry transient failures with exponential backoff (FR-033)
- Clear error messages with actionable guidance (FR-031, FR-032)

**Retry Logic**: Per FR-033
- Max retries: 3
- Base delay: 500ms
- Max delay: 30s
- Exponential backoff for transient errors (429, 500, 502/503, network errors)

**Idempotency**: Standard Terraform patterns
- Read operations: No side effects (FR-010)
- Create operations: Idempotent via unique policy names (FR-009)
- Update operations: Read-modify-write ensures consistency (FR-018, FR-025)
- Delete operations: No-op if already deleted

**Drift Detection**: Standard Terraform Read() pattern (FR-011)
- Refresh during plan phase detects out-of-band changes
- State reconciliation via terraform refresh

### Maintainability

**Code Organization**: Follow existing provider structure
- Resources: `internal/provider/*_resource.go`
- Models: `internal/models/*.go`
- Validators: `internal/validators/*.go`
- Clients: `internal/client/*.go`

**Documentation**: LLM-friendly per FR-012/FR-013
- 100% attribute coverage in docs
- ≥3 working examples per resource
- All constraints explicitly stated
- Terraform registry standards compliance

**Testing**: Per TESTING-GUIDE.md
- Acceptance tests with TF_ACC=1 (primary)
- Unit tests for validators and helpers (selective)
- CRUD testing framework templates

**Logging**: Structured logging with terraform-plugin-log
- Use tflog.Info/Debug/Warn/Error with context
- Include operation type, resource ID in metadata
- Exclude sensitive data (passwords, tokens)
- Example: `tflog.Info(ctx, "Creating policy", map[string]interface{}{"policy_name": name})`

### Compatibility

**Terraform Version**: 1.0+ with Plugin Framework v6 (Assumption 7)
- Tested versions: To be determined during implementation
- Expected compatibility: Terraform 1.0 through latest

**ARK SDK Version**: v1.5.0+ (Assumption 3, plan.md line 19)
- Minimum: v1.5.0 (UAP policy management methods available)
- Provider pinned to v1.5.0 in go.mod
- Updates require provider version bump and testing

**Go Version**: 1.25.0 (plan.md line 17)
- Consistent with existing provider codebase

**Operating Systems**: Cross-platform (plan.md line 26)
- Linux, macOS, Windows supported (standard Go compilation)

---

## Phase 2: Implementation Plan (Tasks Generation)

**Note**: Phase 2 is executed by `/speckit.tasks` command, NOT by `/speckit.plan`.

This plan document ends after Phase 1. The `/speckit.tasks` command will:
1. Load this plan.md and data-model.md
2. Generate dependency-ordered tasks in tasks.md
3. Create actionable implementation steps with acceptance criteria

**Expected Task Groups** (for reference):
1. **Validators** (4 tasks) - policy_status, principal_type, location_type validators
2. **State Models** (3 tasks) - DatabasePolicyModel, PolicyPrincipalAssignmentModel, update existing
3. **Policy Resource** (5 tasks) - Schema, Create, Read, Update, Delete, Import
4. **Principal Assignment Resource** (5 tasks) - Schema, Create, Read, Update, Delete, Import
5. **Database Assignment Updates** (2 tasks) - Consistency updates, location_type validation
6. **Documentation** (6 tasks) - 3 resource docs + 3 example sets
7. **Testing** (4 tasks) - CRUD templates, acceptance tests, integration tests

**Total Estimated**: ~30 tasks, ~40-60 hours implementation time

---

## Constitution Re-Check (Post-Design)

*GATE: Must pass before proceeding to implementation.*

### Re-evaluation Against Constitution

1. ✅ **Pattern Reuse**: Successfully reuses read-modify-write, composite ID, ForceNew patterns
2. ✅ **SDK Maximization**: Uses ARK SDK v1.5.0 for all UAP operations
3. ✅ **Documentation Quality**: Follows LLM-friendly patterns from existing docs
4. ✅ **Testability**: CRUD testing framework applicable to all new resources

**No new violations introduced.** Ready to proceed to tasks generation.

---

## Risks & Mitigation

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| UpdatePolicy() API constraint (single workspace type) causes issues | High | Low | Already handled in existing `policy_workspace_assignment_resource.go` (line 1110+) |
| Concurrent modification race conditions | Medium | Medium | Document as known limitation (same as aws_security_group_rule); users coordinate workspaces |
| Principal directory validation complexity | Low | Low | Use API-only validation (same pattern as database workspace validation) |
| Status management state conflicts | Medium | Low | Custom validator limits user input to "Active"\|"Suspended" only |
| Location type compatibility breaks | High | Low | ForceNew on location_type + documentation warnings |

---

## Next Steps

1. **Review this plan** - Confirm design decisions and architecture
2. **Run `/speckit.tasks`** - Generate actionable task list with dependencies
3. **Execute implementation** - Follow generated tasks.md
4. **CRUD testing** - Use `examples/testing/TESTING-GUIDE.md` for validation
5. **Documentation review** - Ensure LLM-friendly per FR-012/FR-013

**Estimated Timeline**: 2-3 weeks for full implementation + testing + documentation.
