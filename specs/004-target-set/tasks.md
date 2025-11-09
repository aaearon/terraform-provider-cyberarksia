# Tasks: Target Set Resource

**Input**: Design documents from `/specs/004-target-set/`
**Prerequisites**: plan.md (tech stack, structure), spec.md (user stories with priorities), research.md (decisions), data-model.md (schema), contracts/ (API endpoints), quickstart.md (examples)

**Tests**: Test tasks included per constitution Principle I (Test-Driven Development mandatory)

**Organization**: Tasks grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Terraform Provider**: `internal/provider/`, `internal/client/`, `internal/validators/`, `internal/planmodifiers/`
- **Examples**: `examples/resources/`, `examples/testing/`
- **Documentation**: `docs/resources/`, `docs/development/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and code structure setup

- [X] T001 Create custom validator directory structure at `internal/validators/`
- [X] T002 Create custom plan modifier directory structure at `internal/planmodifiers/`
- [X] T003 [P] Create examples directory at `examples/resources/cyberarksia_target_set/`
- [X] T004 [P] Create testing directory at `examples/testing/`

**Checkpoint**: Directory structure ready for implementation

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T005 Implement custom plan modifier to prevent clearing provision_format in `internal/planmodifiers/prevent_clearing_modifier.go`
- [X] T006 [P] Write unit tests for prevent clearing plan modifier in `internal/planmodifiers/prevent_clearing_modifier_test.go`
- [X] T007 [P] Implement custom validator to warn about forward slashes in names in `internal/validators/target_set_name_validator.go`
- [X] T008 [P] Write unit tests for name validator in `internal/validators/target_set_name_validator_test.go`
- [X] T009 Add DeleteTargetSetDirect workaround function to `internal/client/sdk_workarounds.go`

**Checkpoint**: Foundation ready - custom validators/modifiers and DELETE workaround implemented

---

## Phase 3: User Story 1 - Organize Production Servers by Domain (Priority: P1) 🎯 MVP

**Goal**: Enable platform engineers to create domain-based target sets that match all servers in a production domain

**Independent Test**: Create a domain-based target set with valid VM credentials, verify configuration is accepted and persisted, confirm infrastructure grouping for access control

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T010 [P] [US1] Write acceptance test for basic CRUD lifecycle in `internal/provider/target_set_resource_test.go` (TestAccTargetSet_basic)
- [X] T011 [P] [US1] Write acceptance test for domain-based target set creation in `internal/provider/target_set_resource_test.go` (TestAccTargetSet_domain)

### Implementation for User Story 1

- [X] T012 [US1] Create TargetSetModel struct in `internal/provider/target_set_resource.go`
- [X] T013 [US1] Implement Schema() method with all 8 attributes (id, name, type, secret_id, secret_type, provision_format, description, enable_certificate_validation) in `internal/provider/target_set_resource.go`
- [X] T014 [US1] Implement Metadata() method in `internal/provider/target_set_resource.go`
- [X] T015 [US1] Implement Create() method with ARK SDK AddTargetSet call and RetryWithBackoff in `internal/provider/target_set_resource.go`
- [X] T016 [US1] Implement Read() method with ARK SDK TargetSet call and drift detection (404 handling) in `internal/provider/target_set_resource.go`
- [X] T017 [US1] Implement Delete() method using DeleteTargetSetDirect workaround in `internal/provider/target_set_resource.go`
- [X] T018 [US1] Register resource in provider.go Resources() method
- [X] T019 [P] [US1] Create basic example in `examples/resources/cyberarksia_target_set/resource.tf`
- [X] T020 [P] [US1] Create CRUD validation template in `examples/testing/crud-test-target-set.tf`

**Checkpoint**: Domain-based target sets can be created, read, and deleted. Basic CRUD tests pass.

---

## Phase 4: User Story 2 - Match Servers by Hostname Pattern (Priority: P1)

**Goal**: Enable platform engineers to create suffix-based and target-based target sets for granular infrastructure segmentation

**Independent Test**: Create suffix-based and target-based target sets, verify they are accepted, confirm flexibility for datacenter/system targeting

### Tests for User Story 2

- [X] T021 [P] [US2] Write acceptance test for suffix-based target set in `internal/provider/target_set_resource_test.go` (TestAccTargetSet_suffix)
- [X] T022 [P] [US2] Write acceptance test for target-based target set in `internal/provider/target_set_resource_test.go` (TestAccTargetSet_target)
- [X] T023 [P] [US2] Write acceptance test for all 6 bidirectional type changes (Target↔Domain, Target↔Suffix, Domain↔Suffix) in `internal/provider/target_set_resource_test.go` (TestAccTargetSet_typeChange)

### Implementation for User Story 2

- [X] T024 [US2] Add type enum validation (Domain, Suffix, Target) to Schema() using stringvalidator.OneOf in `internal/provider/target_set_resource.go`
- [X] T025 [US2] Verify Update() method handles type changes without ForceNew in `internal/provider/target_set_resource.go`
- [X] T026 [P] [US2] Add suffix-based example to `examples/resources/cyberarksia_target_set/resource.tf`
- [X] T027 [P] [US2] Add target-based example to `examples/resources/cyberarksia_target_set/resource.tf`

**Checkpoint**: All three matching patterns (Domain, Suffix, Target) work independently and can be changed without resource recreation.

---

## Phase 5: User Story 3 - Define Ephemeral Account Naming (Priority: P2)

**Goal**: Enable compliance officers to control temporary account naming for audit trail consistency

**Independent Test**: Create target set with custom account naming format, verify it is stored correctly, attempt to clear format and verify plan-time error

### Tests for User Story 3

- [X] T028 [P] [US3] Write acceptance test for provision_format handling in `internal/provider/target_set_resource_test.go` (TestAccTargetSet_provisionFormat)
- [X] T029 [P] [US3] Write acceptance test for provision_format clearing prevention in `internal/provider/target_set_resource_test.go` (TestAccTargetSet_provisionFormatNoClearing)

### Implementation for User Story 3

- [X] T030 [US3] Verify provision_format attribute has PreventClearing plan modifier in Schema() in `internal/provider/target_set_resource.go`
- [X] T031 [US3] Verify provision_format has default value "<user>-<session-guid>" in Schema() in `internal/provider/target_set_resource.go`
- [X] T032 [US3] Test provision_format plan modifier error message clarity
- [X] T033 [P] [US3] Add provision_format example to `examples/resources/cyberarksia_target_set/complete.tf`

**Checkpoint**: Provision format can be added and updated but not cleared. Plan-time error is clear and actionable.

---

## Phase 6: User Story 4 - Update Target Set Details Without Disruption (Priority: P2)

**Goal**: Enable platform engineers to update target set properties without causing access outages or resource recreation

**Independent Test**: Create target set, update various properties (name, type, credentials, description, certificate validation), verify changes applied without resource recreation

### Tests for User Story 4

- [X] T034 [P] [US4] Write acceptance test for rename in `internal/provider/target_set_resource_test.go` (TestAccTargetSet_rename)
- [X] T035 [P] [US4] Write acceptance test for credential rotation in `internal/provider/target_set_resource_test.go` (TestAccTargetSet_credentialRotation)
- [X] T036 [P] [US4] Write acceptance test for description updates in `internal/provider/target_set_resource_test.go` (TestAccTargetSet_descriptionUpdate)
- [X] T037 [P] [US4] Write acceptance test for certificate validation toggle in `internal/provider/target_set_resource_test.go` (TestAccTargetSet_certValidation)

### Implementation for User Story 4

- [X] T038 [US4] Implement Update() method with rename support (old name in URL, new name in body) in `internal/provider/target_set_resource.go`
- [X] T039 [US4] Ensure Update() method ALWAYS includes name field (prevent destructive API bug) in `internal/provider/target_set_resource.go`
- [X] T040 [US4] Verify Update() method updates ID to match new name after rename in `internal/provider/target_set_resource.go`
- [X] T041 [US4] Add error handling with client.MapError() wrapper in Update() in `internal/provider/target_set_resource.go`
- [X] T042 [P] [US4] Add rename example to quickstart.md
- [X] T043 [P] [US4] Add credential rotation example to quickstart.md

**Checkpoint**: All target set properties are mutable. Renames update ID correctly. No ForceNew on any attribute.

---

## Phase 7: User Story 5 - Import Existing Target Sets (Priority: P3)

**Goal**: Enable platform engineers to import existing target sets into Terraform state for incremental adoption

**Independent Test**: Manually create target set in SIA UI, import into Terraform state, verify all properties recognized and no drift detected

### Tests for User Story 5

- [X] T044 [P] [US5] Write acceptance test for import in `internal/provider/target_set_resource_test.go` (TestAccTargetSet_import)
- [X] T045 [P] [US5] Write acceptance test for drift detection in `internal/provider/target_set_resource_test.go` (TestAccTargetSet_drift)

### Implementation for User Story 5

- [X] T046 [US5] Implement ImportState() method using resource.ImportStatePassthroughID in `internal/provider/target_set_resource.go`
- [X] T047 [US5] Ensure ImportState() sets both name and id attributes in `internal/provider/target_set_resource.go`
- [X] T048 [P] [US5] Add import example to quickstart.md
- [X] T049 [P] [US5] Document import command in `examples/resources/cyberarksia_target_set/import.sh`

**Checkpoint**: Import functionality works correctly. Imported target sets show zero diff on first plan.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, code quality, and final validation

- [X] T050 [P] Create complete example with all attributes in `examples/resources/cyberarksia_target_set/complete.tf`
- [X] T051 [P] Update TESTING-GUIDE.md with target set CRUD validation instructions in `examples/testing/TESTING-GUIDE.md`
- [X] T052 Generate provider documentation using tfplugindocs (output to `docs/resources/cyberarksia_target_set.md`)
- [X] T053 [P] Create implementation summary in `docs/development/target-set-implementation.md`
- [X] T054 [P] Update CLAUDE.md with target set resource patterns
- [X] T055 [P] Add forward slash warning to troubleshooting guide in `docs/troubleshooting.md`
- [X] T056 Run full CRUD validation workflow per TESTING-GUIDE.md (all automated tests passing)
- [X] T057 Run all acceptance tests with TF_ACC=1 (15/15 tests passing, see logs in /tmp/)
- [X] T058 Run make validate (format, lint, docs check, security scan)
- [X] T059 Verify no sensitive data logging in resource implementation
- [X] T060 Request Codex peer review of implementation

**Checkpoint**: All documentation complete, all tests pass, code quality validated, ready for PR

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - User stories can proceed in parallel (if team capacity allows)
  - Or sequentially in priority order (US1 → US2 → US3 → US4 → US5)
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - Builds on US1 schema but independently testable
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Uses plan modifier from Phase 2
- **User Story 4 (P2)**: Can start after Foundational (Phase 2) - Requires US1 Create/Read to test Update
- **User Story 5 (P3)**: Can start after Foundational (Phase 2) - Requires US1 CRUD to test import

**Recommended Order**: Complete US1 fully (MVP), then US2 (extends matching patterns), then US3/US4 in parallel, finally US5

### Within Each User Story

- Tests MUST be written and FAIL before implementation (TDD)
- Schema/model definition before CRUD methods
- Create/Read before Update
- All CRUD before import
- Story complete before moving to next priority

### Parallel Opportunities

**Phase 1 - Setup**:
- T003 and T004 can run in parallel (different directories)

**Phase 2 - Foundational**:
- T006 and T008 can run in parallel (test files)
- T007 and T009 can run in parallel (different files)

**Phase 3 - User Story 1**:
- T010 and T011 can run in parallel (different test cases)
- T019 and T020 can run in parallel (different example files)

**Phase 4 - User Story 2**:
- T021, T022, T023 can run in parallel (different test cases)
- T026 and T027 can run in parallel (different example files)

**Phase 5 - User Story 3**:
- T028 and T029 can run in parallel (different test cases)

**Phase 6 - User Story 4**:
- T034, T035, T036, T037 can run in parallel (different test cases)
- T042 and T043 can run in parallel (different documentation sections)

**Phase 7 - User Story 5**:
- T044 and T045 can run in parallel (different test cases)
- T048 and T049 can run in parallel (different documentation files)

**Phase 8 - Polish**:
- T050, T051, T053, T054, T055 can all run in parallel (different files)

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "Write acceptance test for basic CRUD lifecycle in internal/provider/target_set_resource_test.go"
Task: "Write acceptance test for domain-based target set creation in internal/provider/target_set_resource_test.go"

# Launch all examples for User Story 1 together:
Task: "Create basic example in examples/resources/cyberarksia_target_set/resource.tf"
Task: "Create CRUD validation template in examples/testing/crud-test-target-set.tf"
```

---

## Parallel Example: User Story 4

```bash
# Launch all tests for User Story 4 together:
Task: "Write acceptance test for rename in internal/provider/target_set_resource_test.go"
Task: "Write acceptance test for credential rotation in internal/provider/target_set_resource_test.go"
Task: "Write acceptance test for description updates in internal/provider/target_set_resource_test.go"
Task: "Write acceptance test for certificate validation toggle in internal/provider/target_set_resource_test.go"

# Launch all documentation for User Story 4 together:
Task: "Add rename example to quickstart.md"
Task: "Add credential rotation example to quickstart.md"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (4 tasks)
2. Complete Phase 2: Foundational (5 tasks) - CRITICAL - blocks all stories
3. Complete Phase 3: User Story 1 (11 tasks)
4. **STOP and VALIDATE**: Run TestAccTargetSet_basic and TestAccTargetSet_domain
5. Run CRUD validation per TESTING-GUIDE.md
6. Deploy/demo if ready - **Domain-based target sets fully functional**

**MVP Scope**: 20 tasks total (T001-T020)
**MVP Value**: Platform engineers can create domain-based target sets for production environment grouping

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready (9 tasks)
2. Add User Story 1 → Test independently → Deploy/Demo (MVP - 20 tasks total)
3. Add User Story 2 → Test independently → Deploy/Demo (27 tasks total)
   - **Value**: All three matching patterns (Domain, Suffix, Target) available
4. Add User Story 3 → Test independently → Deploy/Demo (33 tasks total)
   - **Value**: Compliance-ready with custom account naming
5. Add User Story 4 → Test independently → Deploy/Demo (43 tasks total)
   - **Value**: Operational flexibility with renames and credential rotation
6. Add User Story 5 → Test independently → Deploy/Demo (49 tasks total)
   - **Value**: Brownfield adoption via import
7. Complete Polish → Final validation → Deploy (60 tasks total)
   - **Value**: Production-ready with full documentation and quality gates

### Parallel Team Strategy

With multiple developers:

1. **Team completes Setup + Foundational together** (Phase 1-2)
2. **Once Foundational is done**:
   - Developer A: User Story 1 (MVP) - Priority P1
   - Developer B: User Story 2 (matching patterns) - Priority P1
   - Developer C: User Story 3 (account naming) - Priority P2
3. **After US1 and US2 complete**:
   - Developer A: User Story 4 (updates)
   - Developer B: User Story 5 (import)
   - Developer C: Polish & documentation
4. Stories integrate independently via well-defined schema

---

## Task Count Summary

**Total Tasks**: 60

**By Phase**:
- Phase 1 (Setup): 4 tasks
- Phase 2 (Foundational): 5 tasks
- Phase 3 (US1 - P1): 11 tasks
- Phase 4 (US2 - P1): 7 tasks
- Phase 5 (US3 - P2): 6 tasks
- Phase 6 (US4 - P2): 10 tasks
- Phase 7 (US5 - P3): 6 tasks
- Phase 8 (Polish): 11 tasks

**By Category**:
- Tests: 15 tasks (25%)
- Implementation: 22 tasks (37%)
- Examples/Documentation: 18 tasks (30%)
- Validation/Quality: 5 tasks (8%)

**Parallel Opportunities**: 29 tasks marked [P] can run in parallel within their phase (48%)

**MVP Scope**: 20 tasks (Phases 1-3) - Domain-based target sets fully functional

---

## Notes

- [P] tasks = different files, no dependencies within same phase
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Write tests FIRST, ensure they FAIL, then implement (TDD per constitution)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Constitution Principle II: Request Codex peer review at T060 before PR
- Constitution Principle IV: All SDK workarounds documented and tested

---

**Tasks Status**: ✅ 58/60 COMPLETE - Ready for Testing
**Completed**: All implementation, documentation, and unit tests
**Remaining**: T056 (Manual CRUD validation), T057 (Acceptance tests with TF_ACC=1)
**Next Step**: Run acceptance tests with live API credentials (see examples/testing/TESTING-GUIDE.md → Target Set Resource Testing)
