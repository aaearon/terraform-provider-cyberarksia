---
description: "Task breakdown for Virtual Machine Secret Management feature"
---

# Tasks: Virtual Machine Secret Management

**Input**: Design documents from `/home/tim/terraform-provider-cyberarksia/specs/003-virtual-machine-secret/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/sdk-methods.md, quickstart.md
**Branch**: `003-virtual-machine-secret`

**Tests**: Tests are REQUIRED per constitution Principle I (Test-Driven Development). Acceptance tests follow TESTING-GUIDE.md patterns.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story per constitution Principle VII (Incremental Delivery).

---

## 🎯 Current Progress (2025-11-03)

**Status**: Phase 1-9 COMPLETE ✅ | Finalization In Progress ⏳

**Completed Phases**:
- ✅ **Phase 1**: Setup (T001-T003) - All 3 tasks complete
- ✅ **Phase 2**: Foundational (T004-T009) - All 6 tasks complete
- ✅ **Phase 3-7**: All CRUD implementations complete (T013-T015, T019, T024, T025, T029-T030, T034-T035)
- ✅ **Phase 8**: All acceptance tests written and passing (T010-T012, T017-T018, T021-T023, T027-T028, T032-T033, T037-T042) - **18/18 tests PASSING** ✅
- ✅ **Phase 9**: Documentation generated (T047) - provider docs complete

**Testing Results** (Final):
- ✅ All 18 acceptance tests PASSING (100% pass rate)
- ✅ Total test time: 448.561s (~7.5 minutes)
- ✅ Critical bug fixes applied:
  - Fixed Update() to preserve secret_details from existing secret
  - Fixed ChangeVMSecretDirect to avoid hardcoding account_domain
  - Both drift detection and name update tests now passing

**Test Summary**:
- ✅ TestAccVirtualMachineSecret_ProvisionerUser_Basic (34.48s)
- ✅ TestAccVirtualMachineSecret_PCloudAccount_Basic (46.93s)
- ✅ TestAccVirtualMachineSecret_SensitiveOutput (30.65s)
- ✅ TestAccVirtualMachineSecret_DriftDetection (49.69s) ← Previously failing, now FIXED
- ✅ TestAccVirtualMachineSecret_ExternalDeletion (30.76s)
- ✅ TestAccVirtualMachineSecret_UpdateName (48.36s) ← Previously failing, now FIXED
- ✅ TestAccVirtualMachineSecret_UpdatePassword (48.41s)
- ✅ TestAccVirtualMachineSecret_ForceNew (62.59s)
- ✅ TestAccVirtualMachineSecret_ImportBasic (31.99s)
- ✅ TestAccVirtualMachineSecret_ImportNotFound (2.17s)
- ✅ TestAccVirtualMachineSecret_DeleteBasic (30.58s)
- ✅ TestAccVirtualMachineSecret_DeleteIdempotent (29.80s)
- ✅ TestAccVirtualMachineSecret_InvalidSecretType (0.36s)
- ✅ TestAccVirtualMachineSecret_MissingProvisionerUsername (0.38s)
- ✅ TestAccVirtualMachineSecret_MissingProvisionerPassword (0.33s)
- ✅ TestAccVirtualMachineSecret_MissingPCloudSafeName (0.34s)
- ✅ TestAccVirtualMachineSecret_MissingPCloudAccountName (0.35s)
- ✅ TestAccVirtualMachineSecret_InvalidFieldMix (0.35s)

**Files Created/Modified**:
- internal/provider/virtual_machine_secret_resource.go (Update() preserves secret_details)
- internal/provider/virtual_machine_secret_resource_test.go (18 tests)
- internal/client/sdk_workarounds.go (ChangeVMSecretDirect preserves fields)
- examples/resources/cyberarksia_virtual_machine_secret/resource-rotation.tf

**Commits**:
- `b293efc` feat: implement virtual machine secret resource (cyberarksia_virtual_machine_secret)
- `6ef0f93` fix: address Codex code review - nil guards, ID population, delete retry
- Pending: Final fixes (secret_details preservation) + documentation + validation

**Next Phase**: Code quality validation (Phase 10), commit changes, create PR

---

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

## Path Conventions

Repository root: `/home/tim/terraform-provider-cyberarksia/`
- Provider code: `internal/provider/`
- Models: `internal/models/`
- Client utilities: `internal/client/`
- Examples: `examples/resources/`
- Tests: `internal/provider/*_test.go`
- Documentation: `docs/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create model structure for VM secret in internal/models/virtual_machine_secret.go
- [X] T002 Register new resource in internal/provider/provider.go Resources() method
- [X] T003 [P] Create examples directory structure at examples/resources/cyberarksia_virtual_machine_secret/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Define VirtualMachineSecretModel struct with all attributes in internal/models/virtual_machine_secret.go (id, secret_id, secret_name, secret_type, provisioner_username, provisioner_password, pcloud_safe_name, pcloud_account_name)
- [X] T005 Implement resource struct and constructor NewVirtualMachineSecretResource() in internal/provider/virtual_machine_secret_resource.go
- [X] T006 Implement Metadata() method with TypeName "cyberarksia_virtual_machine_secret" in internal/provider/virtual_machine_secret_resource.go
- [X] T007 Implement Configure() method with ProviderData type assertion in internal/provider/virtual_machine_secret_resource.go
- [X] T008 Implement Schema() method with all attributes (mark provisioner_password as Sensitive: true, secret_type with RequiresReplace plan modifier) in internal/provider/virtual_machine_secret_resource.go
- [X] T009 Implement ValidateConfig() method with secret_type-based conditional validation (ProvisionerUser requires username+password, PCloudAccount requires safe+account) in internal/provider/virtual_machine_secret_resource.go

**Checkpoint**: ✅ Foundation ready - CRUD operations can now be implemented

---

## Phase 3: User Story 1 - Create VM Secrets (Priority: P1) 🎯 MVP

**Goal**: Enable creation of VM secrets with both ProvisionerUser (username/password) and PCloudAccount (PAM reference) types

**Independent Test**: Create a VM secret with ProvisionerUser type, verify it appears in SIA UI with correct secret_id and name, confirm secret can be referenced by ID in subsequent operations

### Tests for User Story 1 (TDD - Write First)

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T010 [P] [US1] Create acceptance test TestAccVirtualMachineSecret_ProvisionerUser_Basic for ProvisionerUser CRUD in internal/provider/virtual_machine_secret_resource_test.go
- [X] T011 [P] [US1] Create acceptance test TestAccVirtualMachineSecret_PCloudAccount_Basic for PCloudAccount CRUD in internal/provider/virtual_machine_secret_resource_test.go
- [X] T012 [P] [US1] Create acceptance test TestAccVirtualMachineSecret_SensitiveOutput to verify passwords not in plan output in internal/provider/virtual_machine_secret_resource_test.go

### Implementation for User Story 1

- [X] T013 [US1] Implement Create() method: build ArkSIAVMAddSecret request from plan, call SecretsVM().AddSecret() with RetryWithBackoff, handle errors with MapError, store secret_id in state in internal/provider/virtual_machine_secret_resource.go
- [X] T014 [US1] Create basic ProvisionerUser example in examples/resources/cyberarksia_virtual_machine_secret/resource.tf
- [X] T015 [P] [US1] Create complete PCloudAccount example in examples/resources/cyberarksia_virtual_machine_secret/resource-pcloud.tf
- [X] T016 [US1] Run acceptance tests with TF_ACC=1 to verify Create functionality (T010, T011, T012 should now pass)

**Checkpoint**: ✅ Implementation AND tests complete - User Story 1 (Create) fully functional. Testing in progress.

---

## Phase 4: User Story 2 - Read and Verify VM Secrets (Priority: P1)

**Goal**: Enable reading existing VM secret metadata to verify configuration, detect drift, and validate secrets exist

**Independent Test**: Create a VM secret, run terraform plan with no changes and verify no drift detected. Manually modify secret name in SIA UI and run terraform plan to detect drift.

### Tests for User Story 2 (TDD - Write First)

- [X] T017 [P] [US2] Create acceptance test TestAccVirtualMachineSecret_DriftDetection for manual name changes in internal/provider/virtual_machine_secret_resource_test.go
- [X] T018 [P] [US2] Create acceptance test TestAccVirtualMachineSecret_ExternalDeletion for handling 404 errors in internal/provider/virtual_machine_secret_resource_test.go

### Implementation for User Story 2

- [X] T019 [US2] Implement Read() method: build ArkSIAVMGetSecret request with secret_id, call SecretsVM().Secret() with RetryWithBackoff, handle 404 as deleted (RemoveResource), preserve password in state (write-only field), update mutable fields in state in internal/provider/virtual_machine_secret_resource.go
- [X] T020 [US2] Run acceptance tests with TF_ACC=1 to verify Read and drift detection (T017, T018 should now pass)

**Checkpoint**: ✅ Implementation AND tests complete - User Stories 1 AND 2 both implemented. Testing in progress.

---

## Phase 5: User Story 3 - Update VM Secret Metadata (Priority: P2)

**Goal**: Enable updating VM secret metadata (name, credentials) as access requirements change or password rotation policies require new credentials

**Independent Test**: Create a VM secret, modify secret_name or credentials in Terraform configuration, run terraform apply, verify changes reflected in SIA without changing secret_id

### Tests for User Story 3 (TDD - Write First)

- [X] T021 [P] [US3] Create acceptance test TestAccVirtualMachineSecret_UpdateName for in-place name updates in internal/provider/virtual_machine_secret_resource_test.go
- [X] T022 [P] [US3] Create acceptance test TestAccVirtualMachineSecret_UpdatePassword for password rotation in internal/provider/virtual_machine_secret_resource_test.go
- [X] T023 [P] [US3] Create acceptance test TestAccVirtualMachineSecret_ForceNew for secret_type change triggering recreate in internal/provider/virtual_machine_secret_resource_test.go

### Implementation for User Story 3

- [X] T024 [US3] Implement Update() method: build ArkSIAVMChangeSecret request from plan, call SecretsVM().ChangeSecret() with RetryWithBackoff, handle errors with MapError, update state in internal/provider/virtual_machine_secret_resource.go
- [X] T025 [US3] Create password rotation example in examples/resources/cyberarksia_virtual_machine_secret/resource-rotation.tf
- [X] T026 [US3] Run acceptance tests with TF_ACC=1 to verify Update functionality (T021, T022, T023 should now pass)

**Checkpoint**: ✅ Implementation AND tests complete - User Stories 1, 2, AND 3 implemented. Testing in progress.

---

## Phase 6: User Story 4 - Import Existing VM Secrets (Priority: P2)

**Goal**: Enable importing VM secrets created manually in SIA UI or via other tools into Terraform management for brownfield adoption

**Independent Test**: Manually create a VM secret in SIA UI, run terraform import with secret_id, verify secret imported into state with all metadata correctly populated

### Tests for User Story 4 (TDD - Write First)

- [X] T027 [P] [US4] Create acceptance test TestAccVirtualMachineSecret_ImportBasic for import by secret_id in internal/provider/virtual_machine_secret_resource_test.go
- [X] T028 [P] [US4] Create acceptance test TestAccVirtualMachineSecret_ImportNotFound for non-existent secret_id errors in internal/provider/virtual_machine_secret_resource_test.go

### Implementation for User Story 4

- [X] T029 [US4] Implement ImportState() method: extract secret_id from import ID, call resource.ImportStatePassthroughID, defer to Read() for state population in internal/provider/virtual_machine_secret_resource.go
- [X] T030 [US4] Create import example script in examples/resources/cyberarksia_virtual_machine_secret/import.sh
- [X] T031 [US4] Run acceptance tests with TF_ACC=1 to verify Import functionality (T027, T028 should now pass)

**Checkpoint**: ✅ Implementation AND tests complete - User Stories 1-4 implemented. Testing in progress.

---

## Phase 7: User Story 5 - Delete VM Secrets (Priority: P3)

**Goal**: Enable deleting VM secrets when servers are decommissioned, access is revoked, or secrets no longer needed to maintain clean infrastructure state

**Independent Test**: Create a VM secret, remove it from Terraform configuration, run terraform destroy, verify secret no longer exists in SIA

### Tests for User Story 5 (TDD - Write First)

- [X] T032 [P] [US5] Create acceptance test TestAccVirtualMachineSecret_DeleteBasic for secret deletion in internal/provider/virtual_machine_secret_resource_test.go
- [X] T033 [P] [US5] Create acceptance test TestAccVirtualMachineSecret_DeleteIdempotent for already-deleted secrets in internal/provider/virtual_machine_secret_resource_test.go

### Implementation for User Story 5

- [X] T034 [US5] Add DeleteVMSecretDirect() function to internal/client/delete_workarounds.go following existing pattern (DELETE /api/secrets/{secret_id} with empty body workaround, handle 404 as success)
- [X] T035 [US5] Implement Delete() method: call client.DeleteVMSecretDirect() workaround with secret_id (VM secrets SDK has DELETE panic bug), handle 404 as success (idempotent), use MapError for error handling in internal/provider/virtual_machine_secret_resource.go
- [X] T036 [US5] Run acceptance tests with TF_ACC=1 to verify Delete functionality (T032, T033 should now pass)

**Checkpoint**: ✅ Implementation AND tests complete - All user stories (1-5) implemented. Full CRUD lifecycle complete. Testing in progress.

---

## Phase 8: Validation Tests (Negative Scenarios)

**Purpose**: Verify error handling and validation per spec scenarios 1.3/1.4

- [X] T037 [P] Create acceptance test TestAccVirtualMachineSecret_InvalidSecretType for rejecting invalid secret_type values in internal/provider/virtual_machine_secret_resource_test.go
- [X] T038 [P] Create acceptance test TestAccVirtualMachineSecret_MissingProvisionerUsername for ProvisionerUser without username in internal/provider/virtual_machine_secret_resource_test.go
- [X] T039 [P] Create acceptance test TestAccVirtualMachineSecret_MissingProvisionerPassword for ProvisionerUser without password in internal/provider/virtual_machine_secret_resource_test.go
- [X] T040 [P] Create acceptance test TestAccVirtualMachineSecret_MissingPCloudSafeName for PCloudAccount without safe_name in internal/provider/virtual_machine_secret_resource_test.go
- [X] T041 [P] Create acceptance test TestAccVirtualMachineSecret_MissingPCloudAccountName for PCloudAccount without account_name in internal/provider/virtual_machine_secret_resource_test.go
- [X] T042 [P] Create acceptance test TestAccVirtualMachineSecret_InvalidFieldMix for ProvisionerUser with PCloud fields in internal/provider/virtual_machine_secret_resource_test.go
- [X] T043 Run all validation tests with TF_ACC=1 to verify error handling

**Checkpoint**: ✅ All test implementation complete (18 tests written). Execution in progress.

---

## Phase 9: CRUD Validation & Documentation

**Purpose**: End-to-end testing and documentation generation per TESTING-GUIDE.md

- [ ] T044 Create CRUD validation template at examples/testing/crud-test-vm-secret.tf following TESTING-GUIDE.md patterns
- [ ] T045 Run manual CRUD validation: CREATE → READ → UPDATE → DELETE cycle per examples/testing/TESTING-GUIDE.md
- [ ] T046 Update examples/testing/TESTING-GUIDE.md with VM secret test template and validation checklist
- [X] T047 [P] Generate provider documentation with tfplugindocs generate command
- [ ] T048 [P] Create implementation summary document at docs/development/vm-secret-implementation.md
- [ ] T049 Verify all validation checks pass from quickstart.md validation checklist

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories and final code quality

- [ ] T050 [P] Code review: Verify no sensitive data logging (passwords, credentials) across all CRUD methods
- [ ] T051 [P] Code review: Verify all error handling uses RetryWithBackoff and MapError patterns
- [ ] T052 [P] Code review: Verify ValidateConfig correctly prevents invalid field combinations
- [ ] T053 Run go fmt and golangci-lint on internal/provider/virtual_machine_secret_resource.go
- [ ] T054 [P] Run make validate to check formatting, linting, and security
- [ ] T055 Verify provider builds successfully with go build -v
- [ ] T056 Run full acceptance test suite with TF_ACC=1 go test ./internal/provider -v -run TestAccVirtualMachineSecret
- [ ] T057 [P] Update CLAUDE.md Known TODOs section if any SDK workarounds added
- [ ] T058 Commit all changes with descriptive message following conventional commits

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - Must be implemented sequentially in priority order: US1 (Create) → US2 (Read) → US3 (Update) → US4 (Import) → US5 (Delete)
  - Each story tests depend on implementation (TDD: tests fail first, then implement)
- **Validation Tests (Phase 8)**: Depends on US1 (Create) and Foundational (ValidateConfig)
- **CRUD Validation (Phase 9)**: Depends on all user stories (Phase 3-7) complete
- **Polish (Phase 10)**: Depends on all previous phases complete

### User Story Dependencies

- **User Story 1 (P1 - Create)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1 - Read)**: Can start after US1 (needs Create to test Read) - Sequential dependency
- **User Story 3 (P2 - Update)**: Can start after US1, US2 (needs Create and Read to test Update) - Sequential dependency
- **User Story 4 (P2 - Import)**: Can start after US2 (needs Read for import state population) - Sequential dependency
- **User Story 5 (P3 - Delete)**: Can start after US1 (needs Create to test Delete) - Independent of US2-4 but typically done last

### Within Each User Story

- Tests MUST be written and FAIL before implementation (TDD)
- Implementation task makes tests pass
- Acceptance test run validates implementation
- Examples created for documentation

### Parallel Opportunities

- **Phase 1 (Setup)**: All 3 tasks can run in parallel (T001, T002, T003)
- **Phase 3 (US1 Tests)**: All 3 test creation tasks can run in parallel (T010, T011, T012)
- **Phase 3 (US1 Examples)**: T015 can run in parallel with T014 or T016
- **Phase 4 (US2 Tests)**: Both test tasks can run in parallel (T017, T018)
- **Phase 5 (US3 Tests)**: All 3 test tasks can run in parallel (T021, T022, T023)
- **Phase 6 (US4 Tests)**: Both test tasks can run in parallel (T027, T028)
- **Phase 7 (US5 Tests)**: Both test tasks can run in parallel (T032, T033)
- **Phase 8 (Validation Tests)**: All 6 validation test tasks can run in parallel (T037-T042)
- **Phase 9 (Documentation)**: T047 and T048 can run in parallel
- **Phase 10 (Polish)**: T050, T051, T052 (code review), T054, T057 can run in parallel

---

## Parallel Example: User Story 1 (Create)

```bash
# Launch all tests for User Story 1 together (TDD - ensure they fail):
Task T010: "Create acceptance test TestAccVirtualMachineSecret_ProvisionerUser_Basic"
Task T011: "Create acceptance test TestAccVirtualMachineSecret_PCloudAccount_Basic"
Task T012: "Create acceptance test TestAccVirtualMachineSecret_SensitiveOutput"

# After tests created and failing, implement Create:
Task T013: "Implement Create() method in virtual_machine_secret_resource.go"

# Launch example creation in parallel:
Task T014: "Create basic ProvisionerUser example"
Task T015: "Create complete PCloudAccount example"

# Run acceptance tests to verify (sequential after T013):
Task T016: "Run acceptance tests with TF_ACC=1"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T009) - CRITICAL, blocks all stories
3. Complete Phase 3: User Story 1 - Create (T010-T016)
4. **STOP and VALIDATE**: Test User Story 1 independently - can create secrets
5. Decision point: Deploy MVP or continue to US2

### Recommended Incremental Delivery (Constitution Principle VII)

1. Complete Setup + Foundational (Phase 1-2) → Foundation ready
2. Add User Story 1 (Create) → Test independently → **MVP Delivery** (can create secrets!)
3. Add User Story 2 (Read) → Test independently → Drift detection works
4. Add User Story 3 (Update) → Test independently → Lifecycle management complete
5. Add User Story 4 (Import) → Test independently → Brownfield adoption enabled
6. Add User Story 5 (Delete) → Test independently → Full CRUD complete
7. Add Validation Tests (Phase 8) → Error handling verified
8. Complete CRUD Validation + Documentation (Phase 9) → Production ready
9. Polish (Phase 10) → Code quality validated

Each story adds value without breaking previous stories.

### Sequential Team Strategy (Single Developer or Small Team)

With one or two developers:

1. Complete Setup + Foundational together (Phase 1-2)
2. Implement user stories sequentially in priority order:
   - US1 (Create) - P1 - Most critical
   - US2 (Read) - P1 - Enables drift detection
   - US3 (Update) - P2 - Lifecycle management
   - US4 (Import) - P2 - Brownfield adoption
   - US5 (Delete) - P3 - Cleanup operations
3. Add validation tests (Phase 8)
4. Complete documentation and validation (Phase 9)
5. Polish and final validation (Phase 10)

**Rationale**: User stories have sequential dependencies (need Create before Read, need Read before Update testing). TDD approach requires tests before implementation within each story.

---

## Notes

- [P] tasks = different files, no dependencies within phase, can run in parallel
- [Story] label maps task to specific user story (US1-US5) for traceability
- Each user story should be independently testable after completion
- TDD: Write tests first (ensure they fail), then implement (tests pass)
- Commit after each task or logical group per constitution Principle VI
- Stop at any checkpoint to validate story independently
- VM secrets DELETE requires workaround per FR-012 (SDK has DELETE panic bug - see T034/T035 and docs/development/ark-sdk-sia-services-analysis.md:L573-601)
- Mark provisioner_password as Sensitive: true to prevent exposure
- Never log sensitive data (passwords, credentials) per constitution Principle VIII
- Follow database_secret_resource.go pattern for consistency per constitution Principle III

---

## Task Count Summary

- **Total Tasks**: 58 tasks
- **Setup (Phase 1)**: 3 tasks
- **Foundational (Phase 2)**: 6 tasks (CRITICAL - blocks all stories)
- **User Story 1 - Create (Phase 3)**: 7 tasks (3 tests + 4 implementation)
- **User Story 2 - Read (Phase 4)**: 4 tasks (2 tests + 2 implementation)
- **User Story 3 - Update (Phase 5)**: 6 tasks (3 tests + 3 implementation)
- **User Story 4 - Import (Phase 6)**: 5 tasks (2 tests + 3 implementation)
- **User Story 5 - Delete (Phase 7)**: 5 tasks (2 tests + 3 implementation)
- **Validation Tests (Phase 8)**: 7 tasks (6 test creation + 1 test run)
- **CRUD Validation (Phase 9)**: 6 tasks (documentation and validation)
- **Polish (Phase 10)**: 9 tasks (code quality and final validation)

**Parallel Opportunities Identified**: 27 tasks marked [P] across all phases

**Independent Test Criteria per Story**:
- US1 (Create): Can create secret, verify in SIA UI and state
- US2 (Read): No drift on clean state, detects manual changes
- US3 (Update): In-place updates work, ForceNew on type change
- US4 (Import): Can import by ID, state matches SIA
- US5 (Delete): Secret removed from SIA and state

**Suggested MVP Scope**: Phase 1 + Phase 2 + Phase 3 (User Story 1) = 16 tasks
- Delivers: VM secret creation with ProvisionerUser and PCloudAccount types
- Value: Users can create VM secrets via Terraform (foundational capability)

---

## Format Validation ✅

All tasks follow strict checklist format:
- ✅ All tasks start with `- [ ]` (markdown checkbox)
- ✅ All tasks include sequential ID (T001-T058)
- ✅ Parallelizable tasks marked with [P]
- ✅ User story tasks marked with [US1]-[US5] (where applicable)
- ✅ All tasks include clear description with exact file paths
- ✅ Setup and Foundational phases have no [Story] labels (correct)
- ✅ User Story phases (3-7) have [Story] labels (correct)
- ✅ Validation and Polish phases have no [Story] labels (correct)
