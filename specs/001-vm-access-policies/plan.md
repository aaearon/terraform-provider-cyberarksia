# Implementation Plan: VM Access Policy Management

**Branch**: `001-vm-access-policies` | **Date**: 2025-11-16 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-vm-access-policies/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement Terraform resources for managing CyberArk SIA virtual machine access policies, enabling infrastructure-as-code configuration of just-in-time privileged access to servers and VMs. Two primary resources will be created: `cyberarksia_vm_policy` (main policy resource with inline target rules and required initial principals) and `cyberarksia_vm_policy_principal_assignment` (for adding additional principals post-creation). The implementation follows established provider patterns from database policy resources while adapting for VM-specific requirements: inline target rules (no workspace resources), required minimum one principal at creation, and Read-Modify-Write pattern preserving both inline and assigned principals.

## Technical Context

**Language/Version**: Go 1.25.0
**Primary Dependencies**:
- github.com/cyberark/ark-sdk-golang v1.5.0 (ARK SDK for SIA API)
- github.com/hashicorp/terraform-plugin-framework v1.16.1 (Plugin Framework v6)
- github.com/hashicorp/terraform-plugin-log v0.9.0 (Structured logging)

**Storage**: Not applicable (stateless provider, Terraform state managed externally)
**Testing**:
- Acceptance tests: `TF_ACC=1 go test ./... -v` (requires live CyberArk Identity tenant)
- Unit tests: `go test ./internal/... -v` (selective, for complex utilities only per TESTING-STRATEGY.md)

**Target Platform**:
- Linux server (primary development on WSL2)
- Cross-platform compilation support (Linux, macOS, Windows)

**Project Type**: Single project (Terraform provider plugin)
**Performance Goals**:
- API response time < 2s for CRUD operations
- Retry with exponential backoff for transient failures (3 retries, 30s max delay)
- Efficient schema validation to minimize planning time

**Constraints**:
- ARK SDK v1.5.0 DELETE bug workaround NOT needed for VM policies (BaseDeletePolicy works correctly)
- VM policies REQUIRE at least one principal at creation (schema-enforced, not API-enforced)
- Exactly ONE location type per policy (FQDN/IP, AWS, Azure, GCP)
- Read-Modify-Write pattern required for all policy updates (preserve unmanaged fields)
- Token expiration: 15-minute OAuth2 tokens with automatic refresh

**Scale/Scope**:
- Support 60+ database engines (existing), expanding to VM/server management
- Multi-cloud support (AWS, Azure, GCP) + on-premises (FQDN/IP)
- Policy configuration management for enterprises with hundreds of policies

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Status**: ✅ PASS (Constitution template is empty/placeholder - no active principles defined)

**Note**: The project's constitution file (`.specify/memory/constitution.md`) contains only placeholder content with no enforced principles. No violations detected.

**Recommendation**: Consider populating constitution.md with project-specific principles such as:
- Test-Driven Development requirements
- Security validation gates (prevent credential leakage)
- Provider pattern adherence (profile factory, error handling, retry logic)
- Documentation standards (tfplugindocs, examples, CRUD testing)

## Project Structure

### Documentation (this feature)

```text
specs/001-vm-access-policies/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output - ARK SDK VM models analysis, validation patterns
├── data-model.md        # Phase 1 output - Terraform schema models, SDK mappings
├── quickstart.md        # Phase 1 output - Developer onboarding guide
├── contracts/           # Phase 1 output - SDK API contracts documentation
│   ├── vm_service_api.md         # ARK SDK VM service CRUD operations
│   └── principal_schema.yaml     # Principal assignment schema specification
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
terraform-provider-cyberarksia/
├── internal/
│   ├── provider/                              # Terraform provider implementation
│   │   ├── vm_policy_resource.go              # NEW: Main VM policy resource (CRUD + schema)
│   │   ├── vm_policy_resource_test.go         # NEW: Acceptance tests for VM policy
│   │   ├── vm_policy_principal_assignment_resource.go  # NEW: Principal assignment resource
│   │   ├── vm_policy_principal_assignment_resource_test.go  # NEW: Assignment tests
│   │   ├── helpers/                           # Existing utilities (extend as needed)
│   │   │   ├── composite_ids.go               # EXTEND: Add VM policy principal ID parsing
│   │   │   └── validation.go                  # NEW: VM-specific validators (location type, operators)
│   │   ├── database_policy_resource.go        # REFERENCE: Pattern for policy resources
│   │   ├── database_policy_principal_assignment_resource.go  # REFERENCE: Assignment pattern
│   │   └── provider.go                        # UPDATE: Register new resources
│   ├── client/                                # SDK wrappers and error handling
│   │   ├── retry.go                           # Existing: Reuse RetryWithBackoff
│   │   ├── errors.go                          # Existing: Reuse MapError
│   │   └── delete_workarounds.go              # NOT NEEDED for VM policies
│   ├── models/                                # Data models
│   │   └── vm_policy_models.go                # NEW: Terraform state models for VM policies
│   └── validators/                            # Custom validators
│       └── vm_validators.go                   # NEW: LocationType, FQDNOperator, IPOperator validators
├── examples/
│   ├── resources/
│   │   ├── cyberarksia_vm_policy/             # NEW: VM policy examples
│   │   │   ├── resource.tf                    # Basic FQDN/IP policy with SSH
│   │   │   ├── aws_policy.tf                  # AWS cloud policy example
│   │   │   ├── rdp_policy.tf                  # RDP connection behavior example
│   │   │   └── complete.tf                    # Full-featured policy example
│   │   └── cyberarksia_vm_policy_principal_assignment/  # NEW: Assignment examples
│   │       └── resource.tf                    # Principal assignment example
│   └── testing/
│       ├── crud-test-vm-policy.tf             # NEW: CRUD validation template
│       └── TESTING-GUIDE.md                   # UPDATE: Add VM policy testing scenarios
├── docs/
│   ├── resources/                             # Auto-generated by tfplugindocs
│   │   ├── cyberarksia_vm_policy.md           # NEW: Resource documentation
│   │   └── cyberarksia_vm_policy_principal_assignment.md  # NEW: Assignment docs
│   └── development/
│       ├── vm-policy-implementation.md        # NEW: Implementation summary
│       └── TESTING-STRATEGY.md                # Existing: Reference for test planning
└── CLAUDE.md                                  # UPDATE: Add VM policy resources to table
```

**Structure Decision**: Single project structure (Terraform provider plugin). All implementation follows existing provider patterns in `internal/provider/`, leveraging shared utilities in `internal/client/` and `internal/validators/`. New resources integrate seamlessly with existing database policy resources, reusing proven patterns for error handling, retry logic, and composite ID management.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

*Not applicable - no constitution violations detected.*

---

## Phase 0: Research & Discovery

**Goal**: Resolve all NEEDS CLARIFICATION items and establish implementation patterns.

**Output**: `research.md` with complete SDK mappings, API contracts, validation patterns, and implementation checklist

**Cross-Reference**: All findings documented in `research.md` - see sections below for specific topics

### Research Tasks

1. **ARK SDK VM Policy Models Analysis**
   - **What**: Deep dive into `pkg/services/uap/sia/vm/models/` to understand:
     - `ArkUAPSIAVMAccessPolicy` structure and embedded common policy fields
     - `ArkUAPSIAVMPlatformTargets` (FQDN/IP, AWS, Azure, GCP target types)
     - `ArkUAPSSIAVMBehavior` (SSH/RDP profiles, ephemeral user configurations)
     - Principal array structure (how inline principals are stored)
   - **Why**: Accurate SDK model mapping is critical for schema design and CRUD implementation
   - **Deliverable**: `research.md` §1 documenting all SDK types, field mappings, and serialization patterns

2. **VM Service CRUD Operations Verification**
   - **What**: Analyze `pkg/services/uap/sia/vm/ark_uap_sia_vm_service.go` for:
     - `AddPolicy()` - verify principal requirement behavior
     - `Policy()` - read operation return structure
     - `UpdatePolicy()` - confirm Read-Modify-Write requirement
     - `DeletePolicy()` - confirm no nil body panic (unlike database resources)
   - **Why**: Validate planning context claims about SDK behavior (no DELETE workaround needed, etc.)
   - **Deliverable**: `research.md` §2 documenting actual SDK method signatures and behavior

3. **Validation Pattern Research**
   - **What**: Research Terraform Plugin Framework validators for:
     - ExactlyOneOf pattern for location type blocks (FQDN/IP, AWS, Azure, GCP)
     - ListValidator.SizeAtLeast(1) for required principals array
     - Conditional validators for nested attributes (RDP LocalEphemeralUser vs DomainEphemeralUser)
     - Custom enum validators (FQDNOperator, IPOperator, LocationType)
   - **Why**: Correct validation is essential for catching configuration errors at plan time
   - **Deliverable**: `research.md` §3 with validator patterns and code examples

4. **Read-Modify-Write Pattern for Inline vs Assigned Principals**
   - **What**: Design algorithm for preserving principals during updates:
     - Identify inline principals from resource config (plan.Principals)
     - Identify assigned principals from existing policy (not in inline set)
     - Merge both sets for UpdatePolicy() call
     - Handle duplicate detection across both sets
   - **Why**: Critical pattern unique to VM policies (database policies don't have inline principals)
   - **Deliverable**: `research.md` §4 with pseudocode and example scenarios

5. **Composite ID Format for Principal Assignments**
   - **What**: Extend `internal/provider/helpers/composite_ids.go` pattern:
     - Format: `policy-id:principal-id:principal-type` (3-part, same as database policy)
     - Parsing function: `ParseVMPolicyPrincipalID(id string) (policyID, principalID, principalType string, err error)`
     - Building function: `BuildVMPolicyPrincipalID(policyID, principalID, principalType string) string`
   - **Why**: Reuse proven composite ID pattern for import and state management
   - **Deliverable**: `research.md` §5 documenting ID format with examples

### Research Output Structure

File: `specs/001-vm-access-policies/research.md`

Sections:
1. **ARK SDK Model Mappings** (`research.md` §1) - All SDK types with field-by-field documentation
2. **VM Service API Contract** (`research.md` §2) - CRUD operation signatures and behavior notes
3. **Validation Patterns** (`research.md` §3) - Terraform validators with code examples
4. **Principal Preservation Algorithm** (`research.md` §4) - Read-Modify-Write pattern with pseudocode
5. **Composite ID Specification** (`research.md` §5) - Format, parsing, building functions
6. **Design Decisions Log** (`research.md` §6) - Key choices made during research (with rationale)
7. **Implementation Checklist** (`research.md` §7) - Phase-by-phase implementation tasks

---

## Phase 1: Design & Contracts

**Prerequisites**: `research.md` complete with all SDK models documented

**Output**: `data-model.md`, `contracts/`, `quickstart.md`, updated agent context

### Design Artifacts

#### 1. Data Model (`data-model.md`)

**Purpose**: Complete Terraform state models with SDK field mappings

**Cross-Reference**: See `data-model.md` for complete entity definitions

**Entity Summary** (detailed in `data-model.md` §1-6):

1. **VMPolicyResourceModel** - See `data-model.md` §1
2. **VMPolicyPrincipalModel** - See `data-model.md` §2
3. **VMPolicyConditionsModel** - See `data-model.md` §5
4. **VMPolicyBehaviorModel** - See `data-model.md` §3
5. **VMPolicyTargetsModel** - See `data-model.md` §4 (4.1-4.4 for each location type)
6. **VMPolicyPrincipalAssignmentResourceModel** - See `data-model.md` §6

**Key Mappings**:
- Terraform ↔ SDK field mappings: `data-model.md` §1, Table "Field Mappings"
- Validation constraints: `data-model.md` §7
- State transitions: `data-model.md` §8
- SDK conversion helpers: `data-model.md` §9

#### 2. API Contracts (`contracts/`)

**Purpose**: Complete SDK service documentation with error handling

**Cross-Reference**: See `contracts/vm_service_api.md` for complete API specification

**Contract Summary**:
- Service initialization: `contracts/vm_service_api.md` "Service Initialization"
- CREATE operation: `contracts/vm_service_api.md` "CREATE: AddPolicy"
- READ operation: `contracts/vm_service_api.md` "READ: Policy"
- UPDATE operation: `contracts/vm_service_api.md` "UPDATE: UpdatePolicy"
- DELETE operation: `contracts/vm_service_api.md` "DELETE: DeletePolicy"
- Error handling: `contracts/vm_service_api.md` "Error Handling Patterns"
- Principal schema: `contracts/principal_schema.yaml`

File: `contracts/vm_service_api.md` - Excerpt:

```markdown
# ARK SDK VM Service API Contract

## Service: `github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/vm.ArkUAPSIAVMService`

### CREATE: AddPolicy

**Signature**: `AddPolicy(policy *models.ArkUAPSIAVMAccessPolicy) (*models.ArkUAPSIAVMAccessPolicy, error)`

**Request**:
- Policy object with Metadata, Principals (at least 1), Targets, Behavior, Conditions
- Principal array must have ≥1 element (schema-enforced, not API-enforced)

**Response**:
- Created policy with server-generated PolicyID
- Includes all provided fields plus computed fields (DelegationClassification, timestamps)

**Error Scenarios**:
- 400 Bad Request: Invalid field values, conflicting location types
- 401 Unauthorized: Invalid/expired token
- 409 Conflict: Duplicate policy name

### READ: Policy

**Signature**: `Policy(req *actions.ArkUAPGetPolicyRequest) (*models.ArkUAPSIAVMAccessPolicy, error)`

**Request**: `{PolicyID: "uuid"}`

**Response**:
- Full policy object including ALL principals (inline + assigned)
- All metadata, targets, behavior, conditions

**Error Scenarios**:
- 404 Not Found: Policy deleted (drift detection)
- 401 Unauthorized: Invalid/expired token

### UPDATE: UpdatePolicy

**Signature**: `UpdatePolicy(policy *models.ArkUAPSIAVMAccessPolicy) (*models.ArkUAPSIAVMAccessPolicy, error)`

**Request**:
- Full policy object (Read-Modify-Write required)
- Must preserve unmanaged fields

**Response**:
- Updated policy with all current values

**Error Scenarios**:
- 404 Not Found: Policy deleted
- 400 Bad Request: Validation errors
- 409 Conflict: Name conflict if name changed

### DELETE: DeletePolicy

**Signature**: `DeletePolicy(req *actions.ArkUAPDeletePolicyRequest) error`

**Request**: `{PolicyID: "uuid"}`

**Response**: `nil` on success

**Error Scenarios**:
- 404 Not Found: Already deleted (treat as success)
- 401 Unauthorized: Invalid/expired token

**Note**: Unlike database workspace/secret DELETE methods, `DeletePolicy()` for VM policies uses `BaseDeletePolicy()` which correctly handles nil body. NO workaround needed.
```

File: `contracts/principal_schema.yaml`

```yaml
# Principal Schema Specification
# Used in both inline principals (vm_policy resource) and assignments (vm_policy_principal_assignment resource)

Principal:
  type: object
  required:
    - principal_id
    - principal_name
    - principal_type
  properties:
    principal_id:
      type: string
      format: uuid
      maxLength: 40
      description: "Unique principal identifier (UUID format)"

    principal_name:
      type: string
      minLength: 1
      maxLength: 512
      pattern: '[\w.\-+]+'
      description: "Principal name (email for USER, display name for GROUP/ROLE)"

    principal_type:
      type: string
      enum: [USER, GROUP, ROLE]
      description: "Principal type"

    source_directory_name:
      type: string
      maxLength: 50
      pattern: '\w+'
      description: "Source directory name (REQUIRED for USER/GROUP)"
      conditionalRequired:
        - when: principal_type IN [USER, GROUP]

    source_directory_id:
      type: string
      description: "Source directory ID (REQUIRED for USER/GROUP)"
      conditionalRequired:
        - when: principal_type IN [USER, GROUP]

# Composite ID Format for Principal Assignments
CompositeID:
  format: "policy-id:principal-id:principal-type"
  examples:
    - "a1b2c3d4-5678-90ab-cdef-1234567890ab:e5f6a7b8-9012-34cd-ef56-7890abcdef12:USER"
    - "policy123:principal456:GROUP"
  parsing:
    parts: 3
    separator: ":"
    fields: [policy_id, principal_id, principal_type]
```

#### 3. Quickstart Guide (`quickstart.md`)

**Purpose**: Step-by-step developer onboarding with complete code examples

**Cross-Reference**: See `quickstart.md` for implementation walkthrough

**Content**:
1. **Development Setup** (`quickstart.md` §1): Environment, dependencies, credentials
2. **VM Policy Resource** (`quickstart.md` §2):
   - Create data models (`quickstart.md` §2.1)
   - Define schema with validators (`quickstart.md` §2.2)
   - Implement custom validators (`quickstart.md` §2.3)
   - Implement ValidateConfig (`quickstart.md` §2.4) - **Location type oneOf, principal requirements**
   - Implement CRUD methods (`quickstart.md` §2.5):
     - CREATE - Basic pattern
     - READ - Drift detection
     - UPDATE - **Read-Modify-Write with principal preservation (critical pattern)**
     - DELETE - No workaround needed
3. **Principal Assignment Resource** (`quickstart.md` §3):
   - Extend composite ID helpers (`quickstart.md` §3.1)
   - Implement assignment resource (`quickstart.md` §3.2)
4. **Testing** (`quickstart.md` §4): Acceptance tests, CRUD validation
5. **Common Pitfalls** (`quickstart.md` §5): Principal preservation, DELETE workarounds, validation

#### 4. Agent Context Update

Run: `.specify/scripts/bash/update-agent-context.sh claude`

**New Technology to Add**:
- ARK SDK VM Service (`github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/vm`)
- VM Policy Models (`uapsiavmmodels` package)
- Terraform Plugin Framework validators (ExactlyOneOf, conditional validators)

### Phase 1 Output

Files created:
- ✅ `specs/001-vm-access-policies/data-model.md` - Complete entity documentation
- ✅ `specs/001-vm-access-policies/contracts/vm_service_api.md` - SDK API contract
- ✅ `specs/001-vm-access-policies/contracts/principal_schema.yaml` - Schema specification
- ✅ `specs/001-vm-access-policies/quickstart.md` - Developer onboarding guide
- ✅ Agent context file updated with VM-specific technologies

---

## Phase 2: Tasks Generation

**Status**: NOT STARTED (Phase 2 is handled by `/speckit.tasks` command, NOT `/speckit.plan`)

**Next Steps After Planning**:
1. Review this plan with stakeholders
2. Run `/speckit.tasks` to generate actionable task breakdown in `tasks.md`
3. Implement tasks following generated sequence
4. Use CRUD validation templates from `examples/testing/` for verification

---

## Re-evaluation Gates

### Post-Design Constitution Check

*Re-run after Phase 1 completes*

**Evaluation Criteria**:
- ✅ No new dependencies beyond ARK SDK v1.5.0 and Terraform Plugin Framework
- ✅ No new architectural patterns (reuses existing provider patterns)
- ✅ No security violations (no credential logging, uses existing retry/error handling)
- ✅ Test strategy follows TESTING-STRATEGY.md (acceptance tests primary, selective unit tests)

**Status**: PASS (assuming constitution remains unpopulated; if populated, re-evaluate against actual principles)

---

## Notes for Implementation

### Critical Reminders

**IMPORTANT**: Each reminder includes cross-references to detailed implementation guidance

1. **Principal Requirement**: VM policies REQUIRE ≥1 principal at creation (schema-enforced). Add `Required: true` + `listvalidator.SizeAtLeast(1)` to principals attribute.
   - **Schema**: See `quickstart.md` §2.2 for principals block definition
   - **Validation**: See `research.md` §3.3 for ListValidator.SizeAtLeast(1) pattern

2. **No DELETE Workaround**: Unlike database resources, VM policy `DeletePolicy()` works correctly. Do NOT use `internal/client/delete_workarounds.go`.
   - **Implementation**: See `quickstart.md` §2.5 Delete() method
   - **Explanation**: See `research.md` §2.5, `contracts/vm_service_api.md` "DELETE: DeletePolicy" section

3. **Read-Modify-Write for Principals**: When updating policy, ALWAYS preserve both inline and assigned principals.
   - **Algorithm**: See `research.md` §4.2 for complete pseudocode with step-by-step logic
   - **Implementation**: See `quickstart.md` §2.5 Update() method for complete working example
   - **Pattern**: See `contracts/vm_service_api.md` "UPDATE: UpdatePolicy" section

4. **Location Type OneOf**: Exactly ONE location type per policy (FQDN/IP, AWS, Azure, GCP).
   - **Validation**: See `quickstart.md` §2.4 ValidateConfig() for oneOf enforcement logic
   - **Pattern**: See `research.md` §3.2 for ExactlyOneOf validator examples

5. **ForceNew Attributes**: Name and location_type changes require resource replacement.
   - **Implementation**: See `quickstart.md` §2.2 schema for RequiresReplace() usage

6. **Error Handling**: Always use retry logic and proper error classification.
   - **Patterns**: See `contracts/vm_service_api.md` "Error Handling Patterns" section
   - **Examples**: See `quickstart.md` §2.5 for CRUD error handling in each method

7. **Testing**: Follow established testing guide for CRUD validation.
   - **Structure**: See `quickstart.md` §4 for acceptance test patterns

### Reference Implementations

**Use these existing files as patterns** (detailed in `quickstart.md`):

- **Main Policy Resource**: `internal/provider/database_policy_resource.go`
  - Schema structure → Apply pattern to VM policy (`quickstart.md` §2.2)
  - CRUD implementation → Apply pattern to VM policy (`quickstart.md` §2.5)
  - ValidateConfig → Apply pattern to VM policy (`quickstart.md` §2.4)

- **Assignment Resource**: `internal/provider/database_policy_principal_assignment_resource.go`
  - Read-Modify-Write pattern → Apply to VM principal assignment (`quickstart.md` §3.2)
  - Composite ID usage → Apply to VM assignment (`research.md` §5, `quickstart.md` §3.1)
  - ImportState → Apply to VM assignment

- **Composite IDs**: `internal/provider/helpers/composite_ids.go`
  - Parsing/building functions → Extend for VM policies (`research.md` §5.2-5.3 for function signatures)

- **Error Handling**: `internal/client/errors.go`, `internal/client/retry.go`
  - RetryWithBackoff → Use in all CRUD methods (`contracts/vm_service_api.md` "Error Handling")
  - MapError → Use for error classification (`quickstart.md` §2.5 examples in each CRUD method)

- **Validators**: `internal/validators/`
  - Custom enum pattern → Apply to LocationType, FQDNOperator, IPOperator (`quickstart.md` §2.3)

### Design Decisions to Capture in `research.md`

1. **Why inline principals + assignment resource pattern?**
   - Enables minimum 1 principal at creation (schema requirement)
   - Supports flexible principal management (initial + additional)
   - Preserves separation of concerns (policy vs principal assignment)

2. **Why no workspace assignment resource for VM policies?**
   - VM policies embed target rules inline (unlike database policies)
   - Target criteria (FQDN/IP/cloud filters) are policy configuration, not references

3. **Why Read-Modify-Write instead of partial updates?**
   - ARK SDK `UpdatePolicy()` accepts full policy object
   - Must preserve unmanaged fields (assigned principals, computed metadata)

4. **Why oneOf for location types instead of optional blocks?**
   - API constraint: Exactly ONE location type per policy
   - Terraform best practice: Use validators to enforce API constraints at plan time

---

**Plan Complete**: Ready for Phase 0 research execution. Use this document as the canonical reference for implementation decisions.
