# Tasks: Database Policy Management - Modular Assignment Pattern

**Input**: Design documents from `/specs/002-sia-policy-lifecycle/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Acceptance tests using TF_ACC=1 environment variable (Terraform provider testing standard)

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and validator infrastructure

- [X] T001 Create policy_status_validator.go in internal/validators/ (validates "Active"|"Suspended" only)
- [X] T002 [P] Create principal_type_validator.go in internal/validators/ (validates "USER"|"GROUP"|"ROLE")
- [X] T003 [P] Create location_type_validator.go in internal/validators/ (validates "FQDN/IP" only)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core data models that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Create database_policy.go model in internal/models/ (DatabasePolicyModel, TimeFrameModel, ConditionsModel, AccessWindowModel, ChangeInfoModel)
- [X] T005 Create policy_principal_assignment.go model in internal/models/ (PolicyPrincipalAssignmentModel with ToSDK/FromSDK methods)
- [X] T006 Add composite ID helpers (buildCompositeID, parseCompositeID) to policy_principal_assignment.go in internal/models/
- [X] T007 Update CLAUDE.md with new resource names, composite ID patterns, read-modify-write constraints

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Create Database Policy with Metadata and Conditions (Priority: P1) 🎯 MVP

**Goal**: Enable infrastructure teams to create database policies that establish the foundational access control framework, including policy metadata (name, description, status) and access conditions (time windows, session limits, idle timeouts).

**Independent Test**: Create a policy with name "Test-Policy", description "Test policy", status "Active", weekday 9-5 access window, 8-hour sessions, 30-minute idle timeout. Verify policy appears in SIA UI with correct settings. Policy functions without principals or targets (they're added via separate assignment resources in later stories).

### Implementation for User Story 1

- [X] T008 [US1] Create database_policy_resource.go in internal/provider/ (resource registration and schema definition)
- [X] T009 [US1] Implement Schema() method for database_policy_resource.go with all policy attributes (name, description, status, delegation_classification, time_frame, policy_tags, time_zone, conditions block)
- [X] T010 [US1] Implement Create() method for database_policy_resource.go (convert state to SDK, wrap AddPolicy with RetryWithBackoff per FR-033, handle errors with MapError including duplicate name detection per FR-009, rely on API validation for name length/time_frame/access_window/tag count per FR-034)
- [X] T011 [US1] Implement Read() method for database_policy_resource.go (call Policy API, update state, handle drift detection)
- [X] T012 [US1] Implement Update() method for database_policy_resource.go (read-modify-write pattern: fetch full policy, modify only metadata/conditions, preserve principals/targets, call UpdatePolicy)
- [X] T013 [US1] Implement Delete() method for database_policy_resource.go (call DeletePolicy, handle cascade delete behavior)
- [X] T014 [US1] Implement ImportState() method for database_policy_resource.go (accepts policy ID, fetches full policy, populates state)
- [X] T015 [US1] Register database_policy resource in provider.go Resources() method
- [X] T016 [US1] Add policy resource logging functions to logging.go (logDatabasePolicyCreate, logDatabasePolicyRead, logDatabasePolicyUpdate, logDatabasePolicyDelete)
- [X] T017 [P] [US1] Create docs/resources/database_policy.md (comprehensive LLM-friendly documentation per FR-012/FR-013: attribute tables with Required/Optional/Computed, type info, constraints, relationships, examples)
- [X] T018 [P] [US1] Create examples/resources/cyberarksia_database_policy/basic.tf (minimal policy with required fields only)
- [X] T019 [P] [US1] Create examples/resources/cyberarksia_database_policy/with-conditions.tf (policy with full conditions: max_session_duration, idle_time, access_window)
- [X] T020 [P] [US1] Create examples/resources/cyberarksia_database_policy/suspended.tf (suspended policy example)
- [X] T021 [P] [US1] Create examples/resources/cyberarksia_database_policy/with-tags.tf (policy with tags and metadata)
- [X] T022 [P] [US1] Create examples/resources/cyberarksia_database_policy/complete.tf (all options: time_frame, time_zone, delegation_classification, full conditions)
- [X] T023 [US1] Update examples/testing/TESTING-GUIDE.md with database_policy CRUD testing section (add to "Resource Testing Workflows")
- [X] T024 [US1] Create examples/testing/crud-test-policy.tf template (CREATE: minimal policy → READ: verify state → UPDATE: change description/status/conditions → DELETE: verify removal)

**Checkpoint**: At this point, User Story 1 should be fully functional - users can create, read, update, delete, and import database policies with metadata and conditions. Policies work independently without principals or targets.

---

## Phase 4: User Story 2 - Assign Principals to Policy (Priority: P1)

**Goal**: Enable platform teams to assign users, groups, and roles from identity directories to policies to control who has access to databases governed by the policy.

**Independent Test**: Create policy from US1, assign USER principal with valid directory info (source_directory_id, source_directory_name), assign GROUP principal, verify both principals appear in SIA UI. Principals are ready for database assignments (US3).

### Implementation for User Story 2

- [X] T025 [US2] Create database_policy_principal_assignment_resource.go in internal/provider/ (resource registration and schema definition)
- [X] T026 [US2] Implement Schema() method for database_policy_principal_assignment_resource.go (policy_id, principal_id, principal_type, principal_name, source_directory_name, source_directory_id with conditional validation)
- [X] T027 [US2] Implement Create() method for database_policy_principal_assignment_resource.go (read-modify-write: fetch policy, locate principals array, append new principal, preserve existing principals, wrap UpdatePolicy with RetryWithBackoff per FR-033, handle errors with MapError)
- [X] T028 [US2] Implement Read() method for database_policy_principal_assignment_resource.go (fetch policy, search principals array by ID+type, populate state, detect removal)
- [X] T029 [US2] Implement Update() method for database_policy_principal_assignment_resource.go (read-modify-write: fetch policy, locate principal in array, update in-place, call UpdatePolicy)
- [X] T030 [US2] Implement Delete() method for database_policy_principal_assignment_resource.go (read-modify-write: fetch policy, remove principal from array, preserve other principals, call UpdatePolicy)
- [X] T031 [US2] Implement ImportState() method for database_policy_principal_assignment_resource.go (parse 3-part composite ID "policy-id:principal-id:principal-type", validate format, fetch policy, locate principal, populate state)
- [X] T032 [US2] Add conditional validation for USER/GROUP principal types (require source_directory_name and source_directory_id, optional for ROLE)
- [X] T033 [US2] Register principal_assignment resource in provider.go Resources() method
- [X] T034 [US2] Add principal assignment logging functions to logging.go (logPrincipalAssignmentCreate, logPrincipalAssignmentRead, logPrincipalAssignmentUpdate, logPrincipalAssignmentDelete)
- [X] T035 [P] [US2] Create docs/resources/database_policy_principal_assignment.md (LLM-friendly docs per FR-023: composite ID format, conditional validation rules, read-modify-write pattern, relationships)
- [X] T036 [P] [US2] Create examples/resources/cyberarksia_database_policy_principal_assignment/user-azuread.tf (USER principal with AzureAD directory)
- [X] T037 [P] [US2] Create examples/resources/cyberarksia_database_policy_principal_assignment/user-ldap.tf (USER principal with LDAP directory)
- [X] T038 [P] [US2] Create examples/resources/cyberarksia_database_policy_principal_assignment/group-azuread.tf (GROUP principal)
- [X] T039 [P] [US2] Create examples/resources/cyberarksia_database_policy_principal_assignment/role.tf (ROLE principal without directory)
- [X] T040 [P] [US2] Create examples/resources/cyberarksia_database_policy_principal_assignment/multiple-principals.tf (multiple assignments to same policy)
- [X] T041 [P] [US2] Create examples/resources/cyberarksia_database_policy_principal_assignment/complete.tf (all principal types together)
- [X] T042 [US2] Update examples/testing/TESTING-GUIDE.md with principal_assignment CRUD testing section (3-part composite ID import validation)
- [X] T043 [US2] Create examples/testing/crud-test-principal-assignment.tf template (CREATE: USER+GROUP+ROLE → READ: verify all types → UPDATE: modify principal_name → DELETE: verify selective removal)

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently - users can create policies (US1) and assign principals to them (US2). Principals are ready for database assignments.

---

## Phase 5: User Story 3 - Assign Databases to Policy (Priority: P1)

**Goal**: Enable application teams to assign database workspaces to policies to control which databases the policy governs access to.

**Independent Test**: Create policy with principals (US1+US2), assign database workspace with db_auth authentication method and roles ["db_reader", "db_writer"], verify database appears in policy targets in SIA UI. Principals can now access the assigned database.

### Implementation for User Story 3

- [X] T044 [US3] Update policy_workspace_assignment_resource.go documentation comments (consistency updates: clarify location_type is always "FQDN/IP", document relationship with new database_policy resource)
- [X] T045 [US3] Verify policy_workspace_assignment_resource.go uses location_type "FQDN/IP" consistently (no code changes expected, validation only per plan.md line 335-336)
- [X] T046 [P] [US3] Update docs/resources/policy_workspace_assignment.md (rename from cyberarksia_policy_workspace_assignment to cyberarksia_database_policy_assignment for consistency, clarify location_type constraint, document relationship with database_policy resource)
- [X] T047 [P] [US3] Update examples/resources/cyberarksia_database_policy_assignment/ examples (update resource name references for consistency, add cross-references to database_policy resource)
- [X] T048 [US3] Verify examples/testing/TESTING-GUIDE.md database_assignment section (ensure 2-part composite ID "policy-id:database-id" is documented, no changes expected per plan.md)

**Checkpoint**: At this point, all three user stories work independently AND together - users can create policies (US1), assign principals (US2), and assign databases (US3). Complete access control chain is functional.

---

## Phase 6: User Story 4 - Update Existing Policy Attributes (Priority: P2)

**Goal**: Enable security teams to modify policy metadata attributes (description, status, conditions) as organizational requirements evolve. Changes to policy metadata should not disrupt existing principal or database assignments.

**Independent Test**: Create policy with principals and databases (US1+US2+US3), modify description from "Test policy" to "Updated test policy", change status from "Active" to "Suspended", update max_session_duration from 4 to 8 hours. Verify changes reflected in SIA UI without affecting assignments.

### Implementation for User Story 4

- [X] T049 [US4] Verify Update() method in database_policy_resource.go preserves principals and targets (read-modify-write validation: fetch policy with ALL principals/targets, modify only metadata/conditions, call UpdatePolicy with full policy)
- [X] T050 [US4] Add test scenario to examples/testing/crud-test-policy.tf (UPDATE section: change description, status, idle_time, verify principals/targets unchanged)
- [X] T051 [US4] Update docs/resources/database_policy.md with update behavior details (in-place vs ForceNew attributes, preservation guarantees for assignments)

**Checkpoint**: Policy metadata updates work without affecting principal or database assignments. All CRUD operations on policies are complete.

---

## Phase 7: User Story 5 - Delete Policies (Priority: P3)

**Goal**: Enable operations teams to remove obsolete policies when they are no longer needed, ensuring clean infrastructure state management.

**Independent Test**: Create policy with principals and databases (US1+US2+US3), run terraform destroy, verify complete removal from SIA UI including cascade deletion of all assignments.

### Implementation for User Story 5

- [X] T052 [US5] Verify Delete() method in database_policy_resource.go handles cascade delete behavior (API automatically removes principals and targets, document in code comments)
- [X] T053 [US5] Add cascade delete documentation to docs/resources/database_policy.md (explain API behavior when deleting policies with active assignments per FR-007)
- [X] T054 [US5] Add delete test scenario to examples/testing/crud-test-policy.tf (DELETE section: verify policy removal, document cascade delete behavior)

**Checkpoint**: Policy deletion works correctly with cascade behavior. Full policy lifecycle management is complete.

---

## Phase 8: User Story 6 - Import Existing Policies and Assignments (Priority: P2)

**Goal**: Enable platform engineers to import existing policies and their assignments (principals and databases) created through the SIA UI into Terraform management, enabling infrastructure-as-code adoption for legacy environments.

**Independent Test**: Create policy manually in SIA UI with principals and databases, import policy using policy ID, import each principal assignment using composite ID "policy-id:principal-id:principal-type", import each database assignment using composite ID "policy-id:database-id". Verify all attributes correctly populated in state, terraform plan shows no changes.

### Implementation for User Story 6

- [X] T055 [US6] Verify ImportState() methods preserve all computed fields (policy: created_by, updated_on; assignments: last_modified)
- [X] T056 [US6] Add import examples to docs/resources/database_policy.md (terraform import commands with policy ID format)
- [X] T057 [US6] Add import examples to docs/resources/database_policy_principal_assignment.md (terraform import with 3-part composite ID format, validation error examples)
- [X] T058 [US6] Add import examples to docs/resources/policy_workspace_assignment.md (terraform import with 2-part composite ID format)
- [X] T059 [US6] Update quickstart.md Step 4 with detailed import workflows (import order: policy first, then principals, then databases)

**Checkpoint**: All resources support import. Users can migrate existing SIA infrastructure to Terraform management.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T060 [P] Run go fmt ./... and gofmt -w . (code formatting)
- [X] T061 [P] Run golangci-lint run (linting)
- [X] T062 [P] Run go test ./internal/validators/... -v (validator unit tests)
- [X] T063 [P] Verify all policy resource attributes have godoc comments (code documentation)
- [X] T064 Update development-history.md in docs/ (add entry for database policy management feature with implementation timeline)
- [X] T065 Validate quickstart.md walkthrough (manual testing: follow all steps, verify functionality)
- [X] T066 [P] Run go build -v (final build validation)
- [X] T067 Review all resource documentation for LLM-friendliness per FR-012/FR-013 (attribute tables complete, constraints documented, examples working)
- [X] T068 [P] Verify MapError pattern usage per FR-031 (grep for MapError in all CRUD methods: database_policy_resource.go, database_policy_principal_assignment_resource.go; confirm all API errors use client.MapError())
- [X] T069 [P] Verify retry logic usage per FR-033 (grep for RetryWithBackoff in all CRUD methods; confirm all API calls wrapped with retry: AddPolicy, UpdatePolicy, DeletePolicy, Policy calls)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-8)**: All depend on Foundational phase completion
  - US1 (Policy CRUD) → Independent, can start after Phase 2
  - US2 (Principal Assignment) → Requires US1 complete (needs policy to assign principals to)
  - US3 (Database Assignment) → Independent of US2, requires US1 complete (existing resource, consistency updates only)
  - US4 (Policy Updates) → Requires US1, US2, US3 complete (validates preservation behavior)
  - US5 (Policy Deletion) → Requires US1, US2, US3 complete (validates cascade delete)
  - US6 (Import) → Requires US1, US2, US3 complete (validates import for all resources)
- **Polish (Phase 9)**: Depends on all user stories being complete

### User Story Dependencies

```
Phase 2 (Foundational)
    ↓
US1 (Policy CRUD) ← MVP - can deliver independently
    ↓
    ├──→ US2 (Principal Assignment) ← Requires US1
    └──→ US3 (Database Assignment) ← Requires US1, independent of US2
         ↓
         US4 (Policy Updates) ← Requires US1+US2+US3
         ↓
         US5 (Policy Deletion) ← Requires US1+US2+US3
         ↓
         US6 (Import) ← Requires US1+US2+US3
```

### Within Each User Story

**User Story 1** (Policy CRUD):
1. T008-T016: Core resource implementation (sequential: schema → CRUD → import)
2. T017-T022: Documentation and examples (parallel, can run together)
3. T023-T024: Testing templates (sequential after implementation)

**User Story 2** (Principal Assignment):
1. T025-T034: Core resource implementation (sequential: schema → CRUD → import → validation)
2. T035-T041: Documentation and examples (parallel, can run together)
3. T042-T043: Testing templates (sequential after implementation)

**User Story 3** (Database Assignment):
- T044-T048: Documentation updates only (all parallel, no code changes)

**User Story 4** (Policy Updates):
- T049-T051: Validation and documentation (sequential: verify → test → document)

**User Story 5** (Policy Deletion):
- T052-T054: Validation and documentation (sequential: verify → document → test)

**User Story 6** (Import):
- T055-T059: Documentation enhancements (parallel, can run together)

### Parallel Opportunities

**Phase 1 (Setup)**: T001, T002, T003 (all validators, different files)

**Phase 2 (Foundational)**: T004-T006 (models can be created in parallel, T006 depends on T005)

**Within US1**: T017-T022 (all documentation/examples, different files)

**Within US2**: T035-T041 (all documentation/examples, different files)

**Within US3**: T044-T048 (all documentation updates, different files)

**Within US6**: T055-T059 (all import documentation, different files)

**Phase 9 (Polish)**: T060, T061, T062, T063, T066 (independent quality checks)

---

## Parallel Example: User Story 1 (Policy CRUD)

```bash
# After T008-T016 complete (core implementation), launch all documentation tasks together:
Task T017: "Create docs/resources/database_policy.md"
Task T018: "Create examples/resources/cyberarksia_database_policy/basic.tf"
Task T019: "Create examples/resources/cyberarksia_database_policy/with-conditions.tf"
Task T020: "Create examples/resources/cyberarksia_database_policy/suspended.tf"
Task T021: "Create examples/resources/cyberarksia_database_policy/with-tags.tf"
Task T022: "Create examples/resources/cyberarksia_database_policy/complete.tf"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003) → Validators ready
2. Complete Phase 2: Foundational (T004-T007) → Models ready (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (T008-T024) → Policy CRUD complete
4. **STOP and VALIDATE**: Test User Story 1 independently using examples/testing/crud-test-policy.tf
5. Deploy/demo if ready - users can create, read, update, delete policies

### Incremental Delivery

1. **Foundation** (Phase 1-2): Validators + Models → Ready for resources
2. **MVP** (Phase 3): User Story 1 → Test independently → Deploy (policy metadata + conditions management)
3. **Principal Management** (Phase 4): User Story 2 → Test with US1 → Deploy (add principal assignment capability)
4. **Database Management** (Phase 5): User Story 3 → Test with US1 → Deploy (complete access control chain)
5. **Lifecycle Management** (Phase 6-7): User Stories 4-5 → Test update/delete workflows → Deploy (full CRUD lifecycle)
6. **Import Support** (Phase 8): User Story 6 → Test import workflows → Deploy (enable brownfield adoption)
7. **Polish** (Phase 9): Quality and documentation improvements

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together (Phase 1-2)
2. Once Foundational is done (T007 complete):
   - **Developer A**: User Story 1 (T008-T024) - Policy CRUD
   - **Developer B**: Prepares for User Story 2 (review research.md, plan principal assignment implementation)
3. After US1 complete:
   - **Developer A**: User Story 2 (T025-T043) - Principal Assignment
   - **Developer B**: User Story 3 (T044-T048) - Database Assignment (can start in parallel, independent of US2)
4. After US1+US2+US3 complete:
   - **Developer A**: User Story 4-5 (T049-T054) - Updates and deletion
   - **Developer B**: User Story 6 (T055-T059) - Import support
5. Team completes Polish together (Phase 9)

---

## Task Summary

**Total Tasks**: 69 tasks

**Tasks per Phase**:
- Phase 1 (Setup): 3 tasks
- Phase 2 (Foundational): 4 tasks
- Phase 3 (US1 - Policy CRUD): 17 tasks
- Phase 4 (US2 - Principal Assignment): 19 tasks
- Phase 5 (US3 - Database Assignment): 5 tasks
- Phase 6 (US4 - Policy Updates): 3 tasks
- Phase 7 (US5 - Policy Deletion): 3 tasks
- Phase 8 (US6 - Import): 5 tasks
- Phase 9 (Polish): 10 tasks

**Tasks per User Story**:
- US1: 17 tasks (core policy management)
- US2: 19 tasks (principal assignment with conditional validation)
- US3: 5 tasks (database assignment consistency updates)
- US4: 3 tasks (policy update validation)
- US5: 3 tasks (policy deletion validation)
- US6: 5 tasks (import support documentation)

**Parallel Opportunities**: 29 tasks marked [P] across all phases

**Independent Test Criteria**:
- US1: Create policy, verify in SIA UI without principals/targets
- US2: Assign principals to policy, verify in SIA UI
- US3: Assign databases to policy, verify principals can access
- US4: Update policy metadata, verify assignments preserved
- US5: Delete policy, verify cascade removal
- US6: Import existing resources, verify state matches remote

**Estimated Timeline**: 2-3 weeks for full implementation + testing + documentation

---

## Notes

- [P] tasks = different files, no dependencies within phase
- [Story] label (US1-US6) maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Tests use TF_ACC=1 environment variable (Terraform acceptance testing standard)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- MVP delivery possible after Phase 3 (User Story 1) - policy metadata + conditions management
- All validators created first (Phase 1) to support all resources
- All models created in Phase 2 to unblock parallel resource implementation
- Documentation follows Terraform provider best practices (attribute tables, constraints, examples)
- LLM-friendly documentation per FR-012/FR-013 enables AI-assisted Terraform configuration generation
