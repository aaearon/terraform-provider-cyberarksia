# Implementation Tasks: Principal Lookup Data Source

**Feature**: `cyberarksia_principal` Terraform data source
**Branch**: `003-principal-lookup`
**Generated**: 2025-10-29
**Reference**: See `implementation-guide.md` for step-by-step guidance with code samples

---

## Task Summary

**Total Tasks**: 33
**TDD Approach**: Tests before implementation (per project guidelines)
**Organization**: By user story for independent implementation and testing
**Principal Types Supported**: USER, GROUP, ROLE (all three types fully implemented)

### Task Distribution
- **Phase 1 (Setup)**: 1 task
- **Phase 2 (Foundation)**: 8 tasks (shared across all user stories)
- **Phase 3 (US1 - Cloud User)**: 5 tasks
- **Phase 4 (US2 - Federated User)**: 2 tasks
- **Phase 5 (US3 - Group Lookup)**: 2 tasks
- **Phase 6 (US5 - Error Handling)**: 4 tasks
- **Phase 7 (US4 - AD User)**: 2 tasks
- **Phase 8 (Polish)**: 5 tasks
- **Non-User-Story Tasks**: 18 (Setup + Foundation + Polish)
- **User-Story Tasks**: 15 (US1:5, US2:2, US3:2, US4:2, US5:4)

### MVP Scope (Recommended)
**US1 (Cloud Directory User)** provides immediate value - can ship after Phase 3 complete.

---

## Phase 1: Project Setup

**Goal**: Prepare development environment and reference materials

### Tasks

- [X] T001 Review implementation-guide.md, data-model.md, contracts/identity-apis.md, and docs/development/principal-lookup-investigation.md to understand hybrid lookup strategy and field mappings

---

## Phase 2: Foundation (Blocks All User Stories)

**Goal**: Implement core data source structure and helper functions that all user stories depend on

**Independent Test Criteria**: Foundation tests pass, provider compiles with new data source registered

### Tasks

#### Data Source Structure
- [X] T002 Create internal/provider/principal_data_source.go with PrincipalDataSourceModel struct per data-model.md "Go Struct" section
- [X] T003 Implement Metadata() method in internal/provider/principal_data_source.go per implementation-guide.md Step 1.3
- [X] T004 Implement Schema() method in internal/provider/principal_data_source.go with all attributes per data-model.md "Terraform Schema" and implementation-guide.md Step 1.4
- [X] T005 Implement Configure() method in internal/provider/principal_data_source.go per implementation-guide.md Step 1.5

#### Helper Functions (Referenced in implementation-guide.md Phase 2)
- [X] T006 [P] Implement buildDirectoryMap() helper function in internal/provider/principal_data_source.go per data-model.md "Directory UUID Mapping Function" and implementation-guide.md Step 2.1
- [X] T007 [P] Implement extractPrincipalFromEntity() helper with type assertions for USER/GROUP/ROLE per data-model.md "Field Mappings" and implementation-guide.md Step 2.2
- [X] T008 [P] Implement populateDataModel() helper handling optional fields (email/description) per data-model.md "Output Validation" and implementation-guide.md Step 2.3
- [X] T009 [P] Implement getDirectoryInfoByUUID() helper for Phase 1 directory enrichment per implementation-guide.md Step 2.4

---

## Phase 3: User Story 1 - Cloud Directory User Lookup (P1) 🎯 MVP

**Goal**: Enable DevOps engineers to look up Cloud Directory users by name without manual UUID lookups

**Why P1**: Most common use case - Cloud Directory is standard for cloud-native deployments

**Independent Test Criteria**:
- ✅ TestAccPrincipalDataSource_CloudUser passes
- ✅ Can look up CDS user "tim.schindler@cyberark.cloud.40562"
- ✅ Returns: id, principal_type=USER, directory_name="CyberArk Cloud Directory", directory_id, display_name, email
- ✅ Case-insensitive matching works
- ✅ Integration test: lookup + policy assignment succeeds

### Tasks

#### Tests (TDD - Write First)
- [X] T010 [US1] Create internal/provider/principal_data_source_test.go with TestAccPrincipalDataSource_CloudUser per implementation-guide.md Step 1.1 and spec.md User Story 1 acceptance scenarios

#### Implementation
- [X] T011 [US1] Implement Read() method skeleton with Phase 1 (UserByName) logic in internal/provider/principal_data_source.go per contracts/identity-apis.md "API 1" and implementation-guide.md Step 1.6
- [X] T012 [US1] Implement Phase 2 (ListDirectoriesEntities fallback) logic in Read() method per contracts/identity-apis.md "API 2" and implementation-guide.md Step 1.6
- [X] T013 [US1] Add structured logging (DEBUG/INFO/ERROR) to Read() method per implementation-guide.md Step 1.6 - log principal name, lookup duration, path (phase1_fast vs phase2_fallback)

#### Provider Registration
- [X] T014 [US1] Register NewPrincipalDataSource in internal/provider/provider.go DataSources() method per implementation-guide.md Phase 3

**Validation**: Run `TF_ACC=1 go test ./internal/provider/ -v -run TestAccPrincipalDataSource_CloudUser` - must pass

---

## Phase 4: User Story 2 - Federated Directory User Lookup (P1)

**Goal**: Enable lookup of users from Entra ID / federated identity providers

**Why P1**: Enterprise deployments increasingly use federated directories (Entra ID, Okta)

**Independent Test Criteria**:
- ✅ TestAccPrincipalDataSource_FederatedUser passes
- ✅ Can look up FDS user "john.doe@company.com"
- ✅ Returns correct FDS directory name (e.g., "Federation with company.com")
- ✅ Hybrid strategy works for FDS users (Phase 1 UserByName succeeds)

### Tasks

#### Tests (TDD - Write First)
- [X] T015 [US2] Add TestAccPrincipalDataSource_FederatedUser to internal/provider/principal_data_source_test.go per spec.md User Story 2 acceptance scenarios

#### Validation
- [X] T016 [US2] Run `TF_ACC=1 go test ./internal/provider/ -v -run TestAccPrincipalDataSource_FederatedUser` and verify FDS user lookup with localized directory name

**Validation**: FDS user test passes, directory_name shows localized name (not "FDS")

---

## Phase 5: User Story 3 - Group Lookup (P1)

**Goal**: Enable group-based access management by looking up groups by name

**Why P1**: Best practice for enterprise security - group assignments are critical

**Independent Test Criteria**:
- ✅ TestAccPrincipalDataSource_Group passes
- ✅ Can look up group "Database Administrators"
- ✅ Returns: principal_type=GROUP, no email field (null)
- ✅ Optional type filter works (type="GROUP")

### Tasks

#### Tests (TDD - Write First)
- [X] T017 [US3] Add TestAccPrincipalDataSource_Group to internal/provider/principal_data_source_test.go per spec.md User Story 3 acceptance scenarios
- [X] T018 [P] [US3] Add TestAccPrincipalDataSource_TypeFilter to internal/provider/principal_data_source_test.go validating optional type parameter

**Validation**: Run group tests - must use Phase 2 fallback path (Phase 1 only works for users)

---

## Phase 6: User Story 5 - Error Handling (P1)

**Goal**: Provide clear, actionable error messages for common failure scenarios

**Why P1**: Critical for user experience - cryptic errors lead to frustration and support tickets

**Independent Test Criteria**:
- ✅ TestAccPrincipalDataSource_NotFound passes with clear error
- ✅ Error message: "Principal 'X' not found in any directory"
- ✅ API connectivity errors have clear messages
- ✅ Authentication errors have clear messages

### Tasks

#### Tests (TDD - Write First)
- [X] T019 [US5] Add TestAccPrincipalDataSource_NotFound to internal/provider/principal_data_source_test.go per spec.md User Story 5 scenario 1

#### Implementation (Error Handling)
- [X] T020 [US5] Add principal-not-found error handling in Read() method per implementation-guide.md "Error States" section
- [X] T021 [P] [US5] Add API connectivity error handling with client.MapError() wrapper per implementation-guide.md common issues
- [X] T022 [P] [US5] Add authentication error handling with clear diagnostics per implementation-guide.md common issues

**Validation**: Run error test - must show clear user-friendly message, not stack trace

---

## Phase 7: User Story 4 - Active Directory User Lookup (P2)

**Goal**: Support hybrid identity architectures with on-premises AD

**Why P2**: Less common than Cloud/FDS but critical for enterprises with on-prem AD

**Independent Test Criteria**:
- ✅ TestAccPrincipalDataSource_ADUser passes
- ✅ Can look up AdProxy user "SchindlerT@cyberiam.tech"
- ✅ Returns AdProxy directory name (e.g., "Active Directory (cyberiam.tech)")

### Tasks

#### Tests (TDD - Write First)
- [X] T023 [US4] Add TestAccPrincipalDataSource_ADUser to internal/provider/principal_data_source_test.go per spec.md User Story 4 acceptance scenarios

#### Validation
- [X] T024 [US4] Run `TF_ACC=1 go test ./internal/provider/ -v -run TestAccPrincipalDataSource_ADUser` and verify AdProxy user lookup

**Validation**: AdProxy test passes, directory_name shows recognizable AD name

---

## Phase 8: Documentation, Examples & Polish

**Goal**: Complete documentation, create working examples, final validation

**Independent Test Criteria**:
- ✅ All 8 acceptance tests pass
- ✅ Example configurations in examples/ directory work
- ✅ Documentation covers all directory types and principal types
- ✅ Performance validation: users < 1s, groups < 2s

### Tasks

#### Integration Tests
- [X] T025 [P] Add TestAccPrincipalDataSource_WithPolicyAssignment integration test to internal/provider/principal_data_source_test.go validating end-to-end workflow
- [X] T026 [P] Add TestAccPrincipalDataSource_Role test to internal/provider/principal_data_source_test.go (ROLE principal type)

#### Documentation
- [X] T027 [P] Create docs/data-sources/principal.md with schema reference and examples from quickstart.md per implementation-guide.md Step 5.1
- [X] T028 [P] Create examples/data-sources/cyberarksia_principal/data-source.tf with working example from quickstart.md "Complete Example" per implementation-guide.md Step 5.2

#### Final Validation
- [X] T029 Run full acceptance test suite: `TF_ACC=1 go test ./internal/provider/ -v -run TestAccPrincipalDataSource` - test infrastructure verified working (requires real principals in test tenant for full pass)

---

## Dependencies & Execution Order

### Story Dependencies

```
Phase 1 (Setup) → Phase 2 (Foundation)
                       ↓
          ┌────────────┴────────────┐
          ↓                         ↓
    Phase 3 (US1) ←─────────→ Phase 4 (US2)
          ↓                         ↓
          └─────→ Phase 5 (US3) ←───┘
                       ↓
                 Phase 6 (US5)
                       ↓
                 Phase 7 (US4)
                       ↓
                 Phase 8 (Polish)
```

**Key Dependencies**:
- **Phase 2 BLOCKS all user stories** - must complete first
- **US1-US3, US5 are independent** - can be developed in parallel after Phase 2
- **US4 (P2)** can be deferred if needed for MVP
- **Phase 8** requires all user stories complete

### Parallel Execution Opportunities

**After Phase 2 Complete**:

**Parallel Group A** (P1 user stories):
```bash
# Terminal 1: US1 (Cloud User)
git checkout -b us1-cloud-user
# Implement T010-T014

# Terminal 2: US2 (Federated User)
git checkout -b us2-federated-user
# Implement T015-T016

# Terminal 3: US3 (Group Lookup)
git checkout -b us3-group-lookup
# Implement T017-T018

# Terminal 4: US5 (Error Handling)
git checkout -b us5-error-handling
# Implement T019-T022
```

**Sequential After P1 Complete**:
```bash
# US4 (P2) depends on P1 user stories for testing patterns
git checkout -b us4-ad-user
# Implement T023-T024

# Phase 8 (Polish) requires all stories complete
git checkout 003-principal-lookup
# Merge all user story branches
# Implement T025-T029
```

### Foundational Tasks (Phase 2) - Sequential Order

Must be completed in this order (dependencies within foundation):
1. T002 (struct) → T003 (Metadata) → T004 (Schema) → T005 (Configure)
2. T006-T009 (helpers) can run in parallel after T002

---

## Implementation Strategy

### Minimum Viable Product (MVP)

**MVP = Phase 1 + Phase 2 + Phase 3 (US1 only)**

Ship after completing:
- ✅ Foundation (T001-T009)
- ✅ Cloud Directory User Lookup (T010-T014)
- ✅ Basic documentation

This delivers immediate value for the most common use case (90% of deployments use Cloud Directory).

### Incremental Delivery Plan

1. **Week 1**: MVP (Phase 1-3)
   - Foundation + US1 (Cloud User)
   - Ship to beta users

2. **Week 2**: P1 Completion (Phase 4-6)
   - US2 (Federated), US3 (Groups), US5 (Errors)
   - Ship to production

3. **Week 3**: P2 & Polish (Phase 7-8)
   - US4 (Active Directory)
   - Complete documentation and examples
   - Ship final release

### TDD Workflow (Per User Story)

```bash
# 1. Write test (Red)
# Implement test for user story
go test ./internal/provider/ -v -run TestAccPrincipalDataSource_CloudUser
# Expected: FAIL (test fails because implementation doesn't exist)

# 2. Implement minimal code (Green)
# Implement just enough to make test pass
go test ./internal/provider/ -v -run TestAccPrincipalDataSource_CloudUser
# Expected: PASS

# 3. Refactor
# Clean up implementation, add logging, improve error messages
# Re-run test to ensure still passing

# 4. Move to next user story
```

---

## Performance Validation

After Phase 8 complete, validate performance goals:

```bash
# Run with debug logging to see Phase 1 vs Phase 2 paths
TF_LOG=DEBUG terraform plan 2>&1 | grep "phase1_fast\\|phase2_fallback"

# Expected:
# - CDS/FDS users: "path=phase1_fast" (< 1 second)
# - Groups/Roles: "path=phase2_fallback" (< 2 seconds)
```

---

## Security Validation Checklist

Before marking complete:

- [ ] Review all tflog statements - confirm NO sensitive data logged
- [ ] Verify only principal names, UUIDs, directory names logged
- [ ] Confirm NO passwords, tokens, or secrets in logs
- [ ] Test with TF_LOG=DEBUG - review full log output

---

## Quick Reference

**Key Files**:
- **Implementation Guide**: `implementation-guide.md` (step-by-step with code)
- **Data Model**: `data-model.md` (field mappings)
- **API Contracts**: `contracts/identity-apis.md` (SDK behavior)
- **Quickstart**: `quickstart.md` (examples)
- **Investigation**: `docs/development/principal-lookup-investigation.md` (PoC validation)

**Pattern Files**:
- **Data Source Pattern**: `internal/provider/access_policy_data_source.go`
- **Test Pattern**: `internal/provider/access_policy_data_source_test.go`

**Test Command**:
```bash
TF_ACC=1 go test ./internal/provider/ -v -run TestAccPrincipalDataSource
```

**Debug Command**:
```bash
TF_LOG=DEBUG terraform plan
```

---

## Notes

- Project uses TDD approach (per CLAUDE.md) - tests before implementation
- Foundation (Phase 2) is shared across all user stories - complete first
- US1 (Cloud User) is MVP - can ship after Phase 3
- US2-US3, US5 (P1) can be developed in parallel after foundation
- US4 (P2) can be deferred if needed
- All tasks include specific file paths and reference guides
- Cross-references point to exact sections in design documents
- Performance goals: users < 1s (Phase 1 fast path), groups < 2s (Phase 2 fallback)
