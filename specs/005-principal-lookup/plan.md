# Implementation Plan: Principal Lookup Data Source

**Branch**: `003-principal-lookup` | **Date**: 2025-10-29 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/003-principal-lookup/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement a Terraform data source (`cyberarksia_principal`) that enables users to look up principals (users, groups, or roles) by name and automatically retrieve all information needed for policy assignments. The data source uses a hybrid lookup strategy leveraging ARK SDK v1.5.0 Identity APIs: UserByName() API for fast user lookups, falling back to ListDirectoriesEntities() with empty search for groups/roles or when users aren't found. This eliminates manual UUID lookups and makes the provider Terraform-idiomatic.

## Technical Context

**Language/Version**: Go 1.25.0
**Primary Dependencies**:
- ARK SDK v1.5.0 (`github.com/cyberark/ark-sdk-golang`)
- Terraform Plugin Framework v1.16.1 (`github.com/hashicorp/terraform-plugin-framework`)
- Terraform Plugin Log v0.9.0 (`github.com/hashicorp/terraform-plugin-log`)

**Storage**: N/A (stateless data source)
**Testing**: Go testing with `terraform-plugin-testing v1.13.3` (acceptance tests against real API)
**Target Platform**: Cross-platform (Linux, macOS, Windows) via Go
**Project Type**: Single Terraform provider codebase (existing project: `internal/provider/`)
**Performance Goals**:
- < 1 second for user lookups (Phase 1: UserByName())
- < 2 seconds for group/role lookups (Phase 2: ListDirectoriesEntities)
- Support tenants with up to 10,000 principals

**Constraints**:
- MUST NOT cache lookup results (Terraform data sources are read-only and stateless)
- MUST NOT log sensitive credentials or tokens
- MUST use existing provider authentication context (no new auth mechanisms)
- SDK PageSize/Limit: 10,000 entities per API call
- Case-insensitive name matching required

**Scale/Scope**:
- Single new data source file (~600-800 lines)
- Helper functions for directory mapping and entity processing (~200 lines)
- Comprehensive acceptance tests for 3 directory types × 3 principal types (9 test scenarios)
- Documentation and examples (~300 lines)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Gate 1: Follow Existing Provider Patterns ✅
**Requirement**: Use established patterns from existing data sources and resources
**Status**: PASS - Will follow pattern from `access_policy_data_source.go` (existing data source in provider)
**Evidence**: Project has established conventions for data sources, error handling (`internal/client.MapError`), retry logic (`RetryWithBackoff`), and structured logging (`terraform-plugin-log/tflog`)

### Gate 2: Test-Driven Development ✅
**Requirement**: Acceptance tests required (TF_ACC=1), test against real API
**Status**: PASS - Will write comprehensive acceptance tests for all directory types and principal types
**Evidence**: Project uses `terraform-plugin-testing` framework, existing tests in `*_test.go` files demonstrate pattern

### Gate 3: Documentation Standards ✅
**Requirement**: Generated documentation following `docs/` structure, examples for all use cases
**Status**: PASS - Will create `docs/data-sources/principal.md` and `examples/data-sources/cyberarksia_principal/`
**Evidence**: Established documentation pattern in project (`docs/resources/`, `examples/resources/`)

### Gate 4: No Sensitive Logging ✅
**Requirement**: NEVER log credentials, tokens, or secrets
**Status**: PASS - Will use structured logging at DEBUG/INFO/ERROR levels with only non-sensitive data (principal names, UUIDs, directory names, durations)
**Evidence**: Existing provider follows this pattern (see `internal/provider/logging.go`)

### Gate 5: Error Handling Standards ✅
**Requirement**: Use `internal/client.MapError()` and `RetryWithBackoff()`, provide actionable error messages
**Status**: PASS - Will leverage existing error handling utilities
**Evidence**: All resources/data sources use these utilities (see `database_policy_workspace_assignment_resource.go`)

### Overall Constitution Compliance: ✅ PASS

All gates pass. No violations. No complexity justification required.

## Project Structure

### Documentation (this feature)

```text
specs/003-principal-lookup/
├── spec.md                    # Feature specification (COMPLETE)
├── plan.md                    # This file (/speckit.plan command output - COMPLETE)
├── implementation-guide.md    # Step-by-step implementation guide with code samples (COMPLETE)
├── data-model.md              # Phase 1 output - Entity definitions and field mappings (COMPLETE)
├── quickstart.md              # Phase 1 output - Quick start guide with examples (COMPLETE)
├── contracts/                 # Phase 1 output - ARK SDK API contracts (COMPLETE)
│   └── identity-apis.md       # Consolidated API contracts (UserByName, ListDirectoriesEntities, ListDirectories)
└── tasks.md                   # Phase 2 output (/speckit.tasks command - NOT created yet)
```

**Note**: `research.md` was completed via `docs/development/principal-lookup-investigation.md` (already exists with PoC validation)

### Source Code (repository root)

```text
terraform-provider-cyberarksia/
├── internal/
│   ├── provider/
│   │   ├── principal_data_source.go       # NEW - Main data source implementation (~700 lines)
│   │   ├── principal_data_source_test.go  # NEW - Acceptance tests (~400 lines)
│   │   ├── provider.go                     # MODIFY - Register new data source (1 line)
│   │   └── helpers/                        # EXISTING - Shared utilities
│   │       └── directory_mapping.go        # NEW - Directory UUID mapping helpers (~100 lines)
│   └── client/                             # EXISTING - Error handling, retry logic
│       ├── error_mapping.go                # EXISTING - MapError() utility
│       └── retry.go                        # EXISTING - RetryWithBackoff() utility
├── docs/
│   └── data-sources/
│       └── principal.md                    # NEW - Generated documentation (~200 lines)
├── examples/
│   └── data-sources/
│       └── cyberarksia_principal/
│           ├── data-source.tf              # NEW - Usage examples (~100 lines)
│           └── README.md                   # NEW - Example documentation
└── tests/                                   # EXISTING - Placeholder for future tests
```

**Structure Decision**: Single existing Terraform provider project (Option 1). Adding one new data source file plus tests, documentation, and examples following established provider patterns. No new packages or modules required - all code integrates into existing `internal/provider/` structure.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

**N/A** - No constitution violations. No complexity justification required.

---

## Phase 0: Research & Investigation

### Research Status: ✅ COMPLETE

**Reference**: `docs/development/principal-lookup-investigation.md` (ALREADY COMPLETED)

The investigation has been completed via PoC testing in `/tmp/principal-lookup-poc/`. Key findings:

#### 1. ARK SDK API Behavior (VALIDATED)

**UserByName() API**:
- ✅ Searches by `Username` (SystemName) field
- ✅ Case-insensitive matching
- ✅ Works for CDS and FDS users
- ❌ Does NOT return directory information
- ❌ Only returns users (not groups or roles)

**ListDirectoriesEntities() API**:
- ✅ Searches by `DisplayName` field (NOT SystemName)
- ✅ Empty search parameter returns ALL entities
- ✅ Returns complete directory information (`DirectoryServiceType`, `ServiceInstanceLocalized`)
- ✅ Works for USER, GROUP, and ROLE
- ⚠️ Requires client-side filtering for exact SystemName match

**ListDirectories() API**:
- ✅ Returns directory type → UUID mapping
- ✅ Required for mapping `DirectoryServiceType` (CDS/FDS/AdProxy) to `DirectoryServiceUUID`

#### 2. Optimal Lookup Strategy (VALIDATED VIA POC)

**Hybrid Approach** (implemented in `/tmp/principal-lookup-poc/optimal-hybrid-lookup.go`):

```
Phase 1: Try UserByName() for fast user lookup
  ↓ If user found → Get directory info by matching UUID
  ↓ If user NOT found OR type filter is GROUP/ROLE → Phase 2

Phase 2: ListDirectoriesEntities(search="") + client-side filter
  ↓ Scan all entities
  ↓ Match by exact SystemName (case-insensitive)
  ↓ Return complete principal + directory data
```

**Performance Results**:
- CDS User: < 1 second (Phase 1 path)
- FDS User: < 1 second (Phase 1 path)
- Groups: < 2 seconds (Phase 2 path, ~200 entities scanned)
- Roles: < 2 seconds (Phase 2 path)

#### 3. Directory Name Mapping (CLARIFIED)

**CRITICAL**: Use `ServiceInstanceLocalized` for `source_directory_name` field:
- CDS: "CyberArk Cloud Directory"
- FDS: "Federation with company.com"
- AdProxy: "Active Directory (domain.com)"

**NOT** `DirectoryServiceType` (which is SDK internal: "CDS", "FDS", "AdProxy")

#### 4. Edge Cases (VALIDATED)

| Scenario | Behavior | Solution |
|----------|----------|----------|
| Principal not found | Empty result set | Return clear error: "Principal 'X' not found" |
| Multiple principals with same DisplayName | Possible (tested: 2x "Tim Schindler") | SystemName is unique → no issue |
| Incomplete metadata (no email) | Fields may be empty | Use types.StringNull() for missing fields |
| API connectivity failure | SDK returns error | Wrap with MapError() for Terraform diagnostics |

#### 5. No Unknowns Remaining

All NEEDS CLARIFICATION items resolved:
- ✅ ARK SDK API behavior validated via PoC
- ✅ Hybrid lookup strategy proven via testing
- ✅ Performance characteristics measured
- ✅ Edge cases identified and handled
- ✅ Directory name mapping clarified

**Research Artifact**: `docs/development/principal-lookup-investigation.md` contains complete findings with code samples and test results.

---

## Phase 1: Design & Contracts

### Phase 1 Artifacts: ✅ COMPLETE

#### 1. Data Model (`data-model.md`)
- Principal entity definition
- Directory entity definition
- Field mappings from ARK SDK to Terraform schema
- Validation rules
- State transitions (stateless data source pattern)
- Error states

#### 2. API Contracts (`contracts/identity-apis.md`)
- UserByName() API contract (Phase 1 fast path)
- ListDirectoriesEntities() API contract (Phase 2 fallback)
- ListDirectories() API contract (directory UUID mapping)
- Hybrid lookup strategy documentation

#### 3. Quickstart Guide (`quickstart.md`)
- Basic usage examples
- All directory types (CDS, FDS, AdProxy)
- All principal types (USER, GROUP, ROLE)
- Complete working example with policy assignment
- Error handling examples

---

## Phase 2: Implementation Tasks

**Note**: The `/speckit.tasks` command will generate the detailed `tasks.md` file with dependency-ordered implementation tasks.

**CRITICAL**: Before implementing, read `implementation-guide.md` which provides:
- Step-by-step implementation with code samples
- Cross-references to data-model.md, contracts/identity-apis.md, and investigation report
- TDD approach with test-first development
- Helper function templates
- Common issues and solutions

### High-Level Task Breakdown

#### Task Group 1: Data Source Implementation (~700 lines)
**Primary File**: `internal/provider/principal_data_source.go`
**Reference Guide**: `implementation-guide.md` "Phase 1: Core Data Source"
**Pattern Reference**: `internal/provider/access_policy_data_source.go` (existing data source)

Subtasks:
1. **Create test file first** (TDD) - `implementation-guide.md` Step 1.1
   - Reference: `spec.md` "User Scenarios & Testing" for test matrix
   - Start with TestAccPrincipalDataSource_CloudUser
2. **Create data source structure** - `implementation-guide.md` Step 1.2
   - Reference: `data-model.md` "Go Struct" for model definition
3. **Implement Metadata()** - `implementation-guide.md` Step 1.3
4. **Implement Schema()** - `implementation-guide.md` Step 1.4
   - Reference: `data-model.md` "Terraform Schema" for field definitions
   - Reference: `data-model.md` "Validation Rules" for validators
5. **Implement Configure()** - `implementation-guide.md` Step 1.5
6. **Implement Read()** - `implementation-guide.md` Step 1.6 (MOST COMPLEX)
   - Reference: `contracts/identity-apis.md` for API behavior
   - Reference: `docs/development/principal-lookup-investigation.md` for validated approach
   - Hybrid lookup: Phase 1 (UserByName) → Phase 2 (ListDirectoriesEntities)
   - Error handling with proper diagnostics
   - Structured logging (DEBUG/INFO/ERROR)

#### Task Group 2: Helper Functions (~100 lines)
**Reference Guide**: `implementation-guide.md` "Phase 2: Helper Functions"
**Decision**: Keep helpers in main data source file (simpler) OR create separate helpers package

Subtasks:
1. **buildDirectoryMap()** - `implementation-guide.md` Step 2.1
   - Reference: `data-model.md` "Directory UUID Mapping Function"
   - Maps DirectoryServiceType (CDS/FDS/AdProxy) → DirectoryServiceUUID
2. **extractPrincipalFromEntity()** - `implementation-guide.md` Step 2.2
   - Reference: `data-model.md` "Field Mappings" for SDK → Terraform mapping
   - Handle USER, GROUP, ROLE entity types
   - Type assertions with comma-ok pattern
3. **populateDataModel()** - `implementation-guide.md` Step 2.3
   - Reference: `data-model.md` "Output Validation" for optional field handling
   - Use types.StringNull() for missing email/description
4. **getDirectoryInfoByUUID()** - `implementation-guide.md` Step 2.4
   - Reference: `docs/development/principal-lookup-investigation.md` PoC 4
   - Used after Phase 1 to enrich user data with directory info

#### Task Group 3: Provider Registration (~1 line)
**Reference Guide**: `implementation-guide.md` "Phase 3: Provider Registration"
**File**: `internal/provider/provider.go`

Subtask:
1. **Add NewPrincipalDataSource to DataSources()**
   - Add single line: `NewPrincipalDataSource,` to return array
   - Pattern: Follow existing `NewAccessPolicyDataSource` registration

#### Task Group 4: Acceptance Tests (~400 lines)
**Reference Guide**: `implementation-guide.md` "Phase 4: Complete Test Suite"
**Primary File**: `internal/provider/principal_data_source_test.go`
**Reference**: `spec.md` "User Scenarios & Testing" for complete test matrix

Subtasks (one test per scenario):
1. **TestAccPrincipalDataSource_CloudUser** - `implementation-guide.md` Step 1.1 (already created)
   - Reference: `spec.md` User Story 1
   - Validates CDS user lookup with all required fields
2. **TestAccPrincipalDataSource_FederatedUser**
   - Reference: `spec.md` User Story 2
   - Validates FDS (Entra ID) user lookup
3. **TestAccPrincipalDataSource_ADUser**
   - Reference: `spec.md` User Story 4
   - Validates AdProxy user lookup
4. **TestAccPrincipalDataSource_Group**
   - Reference: `spec.md` User Story 3
   - Validates GROUP lookup with optional type filter
5. **TestAccPrincipalDataSource_Role**
   - Validates ROLE lookup
6. **TestAccPrincipalDataSource_NotFound**
   - Reference: `spec.md` User Story 5
   - Validates error handling for non-existent principal
7. **TestAccPrincipalDataSource_TypeFilter**
   - Validates optional type parameter (USER/GROUP/ROLE)
8. **TestAccPrincipalDataSource_WithPolicyAssignment**
   - Integration test: lookup + use in policy assignment
   - Validates end-to-end workflow

#### Task Group 5: Documentation (~300 lines)
**Reference Guide**: `implementation-guide.md` "Phase 5: Documentation"
**Reference**: `quickstart.md` for all examples and patterns

Subtasks:
1. **Create `docs/data-sources/principal.md`** - Step 5.1
   - Schema reference (all attributes)
   - Usage examples for all directory types (CDS/FDS/AdProxy)
   - Usage examples for all principal types (USER/GROUP/ROLE)
   - Integration example with policy assignment
   - Reference: Copy examples from `quickstart.md`
2. **Create `examples/data-sources/cyberarksia_principal/data-source.tf`** - Step 5.2
   - Reference: `quickstart.md` "Complete Example"
   - Working Terraform configuration with outputs
3. **Create `examples/data-sources/cyberarksia_principal/README.md`**
   - Quick start instructions
   - Prerequisites (service account, test principals)
   - Running the example
4. **Update provider documentation**
   - Add principal data source to main README
   - Update TESTING-GUIDE.md with principal lookup examples

#### Task Group 6: Testing & Validation
**Reference Guide**: `implementation-guide.md` "Validation Checklist"

Subtasks:
1. **Run acceptance tests**
   - Command: `TF_ACC=1 go test ./internal/provider/ -v -run TestAccPrincipalDataSource`
   - All 8 tests must pass
2. **Validate all directory types**
   - CDS user lookup (Cloud Directory)
   - FDS user lookup (Federated Directory / Entra ID)
   - AdProxy user lookup (Active Directory)
3. **Validate all principal types**
   - USER lookups
   - GROUP lookups
   - ROLE lookups
4. **Performance validation**
   - User lookup < 1 second (Phase 1 fast path)
   - Group/role lookup < 2 seconds (Phase 2 fallback)
   - Log analysis: Verify "phase1_fast" vs "phase2_fallback" paths
5. **Security validation**
   - Review tflog output: Confirm NO sensitive data logged
   - Verify only principal names, UUIDs, directory names logged
6. **Manual integration testing**
   - Reference: `examples/testing/TESTING-GUIDE.md`
   - Test principal lookup + policy assignment workflow
   - Verify Terraform plan/apply with data source

---

## Implementation Sequence

```
1. Setup Phase 0 (Research) → ✅ COMPLETE
2. Setup Phase 1 (Design) → ✅ COMPLETE
3. Generate tasks.md (/speckit.tasks) → NEXT STEP
4. Implement data source (TDD: tests first)
5. Implement helper functions
6. Register data source in provider
7. Run acceptance tests
8. Generate documentation
9. Manual testing using TESTING-GUIDE.md
10. Code review
11. Merge to main
```

---

## Dependencies

### External Dependencies
- ARK SDK v1.5.0 (`github.com/cyberark/ark-sdk-golang`)
- Terraform Plugin Framework v1.16.1
- Terraform Plugin Testing v1.13.3
- Existing provider authentication context (ProviderData.AuthContext)

### Internal Dependencies
- `internal/client/error_mapping.go` (MapError utility)
- `internal/client/retry.go` (RetryWithBackoff utility)
- `internal/provider/provider.go` (ProviderData struct)
- Existing data source pattern from `access_policy_data_source.go`

### Test Environment Dependencies
- CyberArk SIA tenant with test principals
- Service account credentials (TF_ACC_USERNAME, TF_ACC_CLIENT_SECRET)
- At least one principal of each type (USER, GROUP, ROLE)
- At least one principal in each directory type (CDS, FDS, AdProxy)

---

## Risks & Mitigations

### Risk 1: Performance Degradation for Large Tenants
**Description**: ListDirectoriesEntities() scans all entities (up to 10,000)
**Mitigation**: Phase 1 fast path handles 90% of use cases (users). Only groups/roles use Phase 2.
**Fallback**: If performance becomes an issue, can add optional caching or request SDK enhancement.

### Risk 2: SystemName Uniqueness Assumption
**Description**: Implementation assumes SystemName is unique across directories
**Mitigation**: Validated via PoC testing (200+ entities). SystemName is the primary identifier.
**Fallback**: If duplicates found, return error listing all matches with directory names.

### Risk 3: SDK API Changes
**Description**: ARK SDK v1.5.0 APIs might change in future versions
**Mitigation**: Pin SDK version in go.mod. Document SDK version compatibility.
**Fallback**: Update implementation if breaking changes occur in SDK updates.

### Risk 4: Missing Directory Information
**Description**: UserByName() doesn't return directory info, requiring Phase 2 lookup
**Mitigation**: Hybrid approach handles this automatically (get UUID from Phase 1, directory info from Phase 2).
**Fallback**: Already implemented in hybrid strategy.

---

## Success Metrics

### Functional Metrics
- ✅ 100% lookup accuracy (no false positives)
- ✅ All 3 directory types supported (CDS, FDS, AdProxy)
- ✅ All 3 principal types supported (USER, GROUP, ROLE)
- ✅ Zero manual UUID lookups required
- ✅ All acceptance tests passing

### Performance Metrics
- ✅ User lookups < 1 second (Phase 1 fast path)
- ✅ Group/role lookups < 2 seconds (Phase 2 fallback)
- ✅ Supports tenants with 10,000+ principals

### Quality Metrics
- ✅ Clear, actionable error messages
- ✅ Structured logging (DEBUG/INFO/ERROR levels)
- ✅ No sensitive data logged
- ✅ Follows existing provider patterns
- ✅ Comprehensive documentation with examples

---

## Conclusion

The principal lookup data source implementation plan is complete and ready for task generation.

**Constitution Compliance**: ✅ PASS - All gates satisfied
**Research**: ✅ COMPLETE - All unknowns resolved via PoC
**Design**: ✅ COMPLETE - Data model, contracts, and quickstart defined
**Next Step**: Run `/speckit.tasks` to generate dependency-ordered implementation tasks

**Branch**: `003-principal-lookup`
**Spec Location**: `specs/003-principal-lookup/spec.md`
**Plan Location**: `specs/003-principal-lookup/plan.md`
**Investigation**: `docs/development/principal-lookup-investigation.md`
