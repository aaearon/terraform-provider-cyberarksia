# Tasks: VM Access Policy Management

**Feature**: Terraform resources for managing CyberArk SIA virtual machine access policies
**Input**: Design documents from `/specs/001-vm-access-policies/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Acceptance tests are included following TESTING-STRATEGY.md (acceptance tests primary, selective unit tests)

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `- [ ] [ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create basic development infrastructure for VM policy resources

- [x] T001 Create feature branch `001-vm-access-policies` from main
- [x] T002 [P] Create directory structure: `internal/models/`, `internal/validators/vm_validators.go`
- [x] T003 [P] Verify development environment: Go 1.25.0, ARK SDK v1.5.0, TF_ACC env vars

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core data models, validators, and helpers that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Create Terraform state models in `internal/models/vm_policy_models.go` (VMPolicyResourceModel, PrincipalModel, BehaviorModel, all Target models per data-model.md)
- [x] T005 [P] Implement custom enum validators in `internal/validators/vm_validators.go` (LocationType, FQDNOperator, IPOperator, PolicyStatus)
- [x] T006 [P] Extend composite ID helpers in `internal/provider/helpers/composite_ids.go` (ParseVMPolicyPrincipalID, BuildVMPolicyPrincipalID per research.md §5)
- [x] T007 Create VM policy principal assignment model in `internal/models/vm_policy_models.go` (VMPolicyPrincipalAssignmentResourceModel per data-model.md §6)

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Create Basic On-Premises Server Access Policy (Priority: P1) 🎯 MVP

**Goal**: Enable infrastructure-as-code configuration of FQDN/IP-based VM access policies with SSH connection behavior, time-based conditions, and required initial principal assignments

**Independent Test**: Create policy via Terraform with FQDN suffix rule, SSH username, time window, and one principal → Read back from SIA API → Verify all configuration values match including assigned principal

### Implementation for User Story 1

- [x] T008 [P] [US1] Create base resource structure in `internal/provider/vm_policy_resource.go` (type VMPolicyResource, Metadata method)
- [x] T009 [US1] Implement Schema method in `internal/provider/vm_policy_resource.go` with core attributes (name, status, location_type, policy_type, time_zone, tags - all ForceNew/validators per quickstart.md §2.2)
- [x] T010 [US1] Add principals block to schema with validators (Required, SizeAtLeast(1), principal attributes with conditional directory fields)
- [x] T011 [US1] Add conditions attributes to schema (max_session_duration, idle_time with defaults, access_window block, time_frame block)
- [x] T012 [US1] Add behavior block to schema with SSH profile (username required if block present)
- [x] T013 [US1] Add fqdn_ip_targets block to schema (fqdn_rule and ip_rule nested blocks with validators per data-model.md §4.1)
- [x] T014 [US1] Implement ValidateConfig method for location type oneOf enforcement and connection profile validation (exactly one location type, at least one SSH/RDP profile per quickstart.md §2.4)
- [x] T015 [US1] Add conditional validation for principal source_directory fields (required for USER/GROUP, optional for ROLE)
- [x] T016 [US1] Implement Create method with SDK policy building and retry logic (map Terraform plan → SDK models, call AddPolicy with RetryWithBackoff per quickstart.md §2.5 CREATE)
- [x] T017 [US1] Implement Read method with drift detection (call Policy, handle 404 as RemoveResource, map SDK → state per quickstart.md §2.5 READ)
- [x] T018 [US1] Implement Update method with Read-Modify-Write principal preservation (fetch existing, identify inline vs assigned principals, merge, update policy per research.md §4.2)
- [x] T019 [US1] Implement Delete method using SDK directly (NO workaround needed - use vmService.DeletePolicy per research.md §2.5)
- [x] T020 [US1] Implement ImportState method using policy_id
- [x] T021 [US1] Create SDK conversion helpers in `internal/provider/vm_policy_resource.go` (buildSDKPrincipals, buildSDKBehavior, buildSDKTargets, mapSDKPolicyToState per data-model.md §9)

### Testing for User Story 1

- [x] T022 [US1] Create acceptance test file `internal/provider/vm_policy_resource_test.go` with basic FQDN/IP policy test - **DONE** (TestAccVMPolicy_basic)
- [x] T023 [US1] Add acceptance test for policy creation with SSH behavior and time window - **DONE** (TestAccVMPolicy_sshWithTimeWindow)
- [x] T024 [US1] Add acceptance test for drift detection (verify 404 removes from state) - **DONE** (TestAccVMPolicy_driftDetection)
- [x] T025 [US1] Add acceptance test for ForceNew behavior on name change - **DONE** (TestAccVMPolicy_forceNewOnNameChange)
- [x] T026 [US1] Add acceptance test for validation errors (zero principals, missing SSH username) - **DONE** (TestAccVMPolicy_validationErrors)

### Examples for User Story 1

- [x] T027 [P] [US1] Create basic FQDN/IP policy example in `examples/resources/cyberarksia_vm_policy/resource.tf`
- [x] T028 [P] [US1] Create CRUD validation template in `examples/testing/crud-test-vm-policy.tf`

**Checkpoint**: User Story 1 should be fully functional - can create, read, update, delete FQDN/IP policies with SSH via Terraform

---

## Phase 4: User Story 2 - Assign Additional Users and Groups to Access Policies (Priority: P1)

**Goal**: Enable dynamic addition of principals to existing VM policies beyond initial assignments configured at policy creation

**Independent Test**: Create policy with one initial principal → Add second principal via assignment resource → Read policy from SIA → Verify both principals appear in policy's principals list

### Implementation for User Story 2

- [x] T029 [US2] Create assignment resource structure in `internal/provider/vm_policy_principal_assignment_resource.go` (type VMPolicyPrincipalAssignmentResource, Metadata method)
- [x] T030 [US2] Implement Schema method with composite ID attributes (policy_id, principal_id, principal_name, principal_type, source_directory fields all ForceNew per data-model.md §6)
- [x] T031 [US2] Implement Create method with Read-Modify-Write and duplicate detection (read policy, check for duplicate principal ID+Type, append, update per research.md §4.3)
- [x] T032 [US2] Implement Read method to verify assignment exists (parse composite ID, read policy, find principal in array, RemoveResource if not found)
- [x] T033 [US2] Implement Update method as ForceNew (all attributes require replacement)
- [x] T034 [US2] Implement Delete method with Read-Modify-Write (read policy, remove principal from array, update policy)
- [x] T035 [US2] Implement ImportState method with composite ID parsing (use ParseVMPolicyPrincipalID helper per research.md §5.4)

### Testing for User Story 2

- [x] T036 [US2] Create acceptance test file `internal/provider/vm_policy_principal_assignment_resource_test.go` with basic assignment test - **DONE** (TestAccVMPolicyPrincipalAssignment_basic)
- [x] T037 [US2] Add acceptance test for assignment CRUD lifecycle (create, read, delete) - **DONE** (TestAccVMPolicyPrincipalAssignment_crud)
- [x] T038 [US2] Add acceptance test for duplicate principal detection (verify error when assigning same principal twice) - **DONE** (TestAccVMPolicyPrincipalAssignment_duplicateDetection)
- [x] T039 [US2] Add acceptance test for ImportState with composite ID - **DONE** (TestAccVMPolicyPrincipalAssignment_importState, TestAccVMPolicyPrincipalAssignment_compositeID)

### Examples for User Story 2

- [x] T040 [P] [US2] Create principal assignment example in `examples/resources/cyberarksia_vm_policy_principal_assignment/resource.tf`

**Checkpoint**: User Stories 1 AND 2 should both work - can manage policies and dynamically add/remove principals independently

---

## Phase 5: User Story 3 - Manage AWS Cloud VM Access (Priority: P2)

**Goal**: Enable configuration of AWS-specific VM access policies using cloud-native target criteria (regions, VPC IDs, account IDs, resource tags)

**Independent Test**: Create policy with AWS location type, specify region "us-east-1" and tag "Environment=production" → Read back from SIA → Verify all AWS target criteria match expected configuration

### Implementation for User Story 3

- [x] T041 [US3] Add aws_targets block to schema in `internal/provider/vm_policy_resource.go` (regions, tags with key/value structure, vpc_ids, account_ids per data-model.md §4.2)
- [x] T042 [US3] Extend buildSDKTargets helper to handle AWSResource mapping
- [x] T043 [US3] Extend mapSDKPolicyToState helper to handle AWSResource deserialization

### Testing for User Story 3

- [x] T044 [US3] Add acceptance test for AWS policy creation with regions and tags
- [x] T045 [US3] Add acceptance test for AWS policy with VPC IDs and account IDs
- [x] T046 [US3] Add acceptance test for AWS policy update (change regions)

### Examples for User Story 3

- [x] T047 [P] [US3] Create AWS cloud policy example in `examples/resources/cyberarksia_vm_policy/aws_policy.tf`

**Checkpoint**: All P1 and P2 AWS stories functional - can manage both FQDN/IP and AWS policies

---

## Phase 6: User Story 4 - Configure SSH and RDP Connection Behavior (Priority: P2)

**Goal**: Enable configuration of RDP ephemeral user settings with group assignments for VM access policies

**Independent Test**: Create policy with RDP local ephemeral user settings specifying groups ["Administrators"] → Read back from SIA → Verify RDP behavior configuration matches specified groups

### Implementation for User Story 4

**STATUS: BLOCKED - Schema refactor required (see HANDOFF.md Session 6)**

- [x] T048 [US4] Add rdp block to behavior schema in `internal/provider/vm_policy_resource.go` - **DONE** (Session 2)
- [x] T049 [US4] Extend buildSDKBehavior helper to handle RDP profile mapping - **DONE** (Session 2)
- [x] T050 [US4] Extend mapSDKPolicyToState helper to handle RDP profile deserialization - **DONE** (Session 2)
- [x] T051 [US4] Add ValidateConfig logic for RDP ephemeral user oneOf - **DONE** (Session 3)

**NEW: Schema Fix Required (Session 7) - BLOCKER**

- [x] T056 [BLOCKER] Schema fix: ssh.username Required→Optional with validation (vm_policy_resource.go:282)
- [x] T057 [BLOCKER] Schema fix: assign_groups List→Set (API reorders) (vm_policy_resource.go:296, 313, 317)
- [x] T058 [BLOCKER] Schema fix: Added Default(false) for enable_ephemeral_user_reconnect (vm_policy_resource.go:300, 322)
- [x] T059 [BLOCKER] Schema fix: ObjectType definitions use SetType (vm_policy_resource.go:1933-1934, 1944, 1948-1949)
- [x] T060 [BLOCKER] Updated models to use types.Set for AssignGroups/AssignDomainGroups (vm_policy_models.go:67, 73-74)
- [x] T061 [BLOCKER] Verified buildSDKBehavior() works with Optional ssh.username
- [x] T062 [BLOCKER] Verified mapSDKPolicyToState() works with SetValueFrom for assign_groups
- [x] T063 [US4] Fix RDP null object initialization - **DONE** (Session 6, vm_policy_resource.go:1867-1908)
- [x] T064 [BLOCKER] All 13 existing tests pass (User Stories 1-3)
- [x] T065 [BLOCKER] RDP-only policy works (7 new tests pass)
- [x] T066 [BLOCKER] SSH-only policy works (existing SSH tests pass)

**Root Cause**: Terraform Plugin Framework limitation - SingleNestedBlock with Required attributes makes parent block effectively required.
**Reference**: https://github.com/hashicorp/terraform-plugin-framework/issues/740
**API Evidence**: RDP-only policy (ID: `d3f1cb0a-4a3d-4098-8ff1-5ef22be1e602`) shows `ssh_profile` key completely omitted.

### Testing for User Story 4

**STATUS: All 8 RDP tests passing ✅ (Session 7-8 fixes complete)**

- [x] T067 [US4] TestAccVMPolicy_rdpLocalEphemeral (T048) - **PASSING** ✅ (27.27s)
- [x] T068 [US4] TestAccVMPolicy_rdpDomainEphemeral (T049) - **PASSING** ✅ (20.04s)
- [x] T069 [US4] TestAccVMPolicy_sshAndRdp (T050) - **PASSING** ✅ (34.84s)
- [x] T070 [US4] TestAccVMPolicy_rdpUpdate (T051) - **PASSING** ✅ (35.52s)
- [x] T071 [US4] TestAccVMPolicy_rdpWithTimeWindow (T052) - **PASSING** ✅ (18.85s)
- [x] T072 [US4] TestAccVMPolicy_rdpWithAWSTargets (T053) - **PASSING** ✅ (19.33s)
- [x] T073 [US4] TestAccVMPolicy_rdpMultipleGroups (T054) - **PASSING** ✅ (18.62s)
- [x] T074 [US4] TestAccVMPolicy_rdpReconnectSettings (T055) - **PASSING** ✅ (35.62s)

**Test Results**: All 21 tests passing (13 existing + 8 RDP, 566s total)

### Examples for User Story 4

- [x] T055 [P] [US4] Create RDP connection behavior example - **DONE** (Session 2)

**Checkpoint**: RDP tests written, schema fix required before tests can pass

---

## Phase 7: User Story 5 - Manage Azure and GCP Cloud VM Access (Priority: P3)

**Goal**: Enable configuration of Azure and GCP VM access policies using cloud-native target criteria for multi-cloud support

**Independent Test**: Create policy with Azure location type, resource group "production-rg", and tag "Team=Platform" → Read back from SIA → Verify all Azure target criteria match expected configuration

### Implementation for User Story 5

- [x] T056 [P] [US5] Add azure_targets block to schema (regions, tags, resource_groups, vnet_ids, subscriptions per data-model.md §4.4)
- [x] T057 [P] [US5] Add gcp_targets block to schema (regions, labels NOT tags, vpc_ids, projects per data-model.md §4.3)
- [x] T058 [US5] Extend buildSDKTargets helper to handle AzureResource and GCPResource mapping
- [x] T059 [US5] Extend mapSDKPolicyToState helper to handle Azure and GCP deserialization

### Testing for User Story 5

- [x] T060 [P] [US5] Add acceptance test for Azure policy with resource groups and tags - **DONE** (TestAccVMPolicy_azureBasic)
- [x] T061 [P] [US5] Add acceptance test for GCP policy with labels and projects - **DONE** (TestAccVMPolicy_gcpBasic)

**Azure SDK Workaround (Session 11)**: ARK SDK v1.5.0 has bugs preventing Azure VM policies from working:
- SDK uses `"AZURE"` (uppercase) but API expects `"Azure"` (mixed case) for targets key and locationType
- SDK produces `behavior.sshProfile` but API expects `behavior.connectAs.ssh`
- SDK's `Deserialize()` fails on Azure responses

**Solution**: Implemented direct HTTP workarounds in `internal/client/sdk_workarounds.go`:
- `CreateAzureVMPolicyDirect()`, `ReadAzureVMPolicyDirect()`, `UpdateAzureVMPolicyDirect()`
- Resource methods detect Azure via `strings.EqualFold(locationType, "Azure")` and use workarounds
- GitHub Issue: https://github.com/cyberark/ark-sdk-golang/issues/32
- TODO: Remove when ARK SDK v1.6.0+ fixes Azure serialization

### Examples for User Story 5

- [x] T062 [P] [US5] Create Azure policy example in `examples/resources/cyberarksia_vm_policy/azure_policy.tf`
- [x] T063 [P] [US5] Create GCP policy example in `examples/resources/cyberarksia_vm_policy/gcp_policy.tf`

**Checkpoint**: Multi-cloud support complete - AWS, Azure, GCP all supported

---

## Phase 8: User Story 6 - Update Existing Access Policies (Priority: P2)

**Goal**: Enable modification of existing policy configurations while preserving unmodified elements including assigned principals

**Independent Test**: Create policy with 1-hour session duration → Update to 4 hours via Terraform → Read back from SIA → Verify new value persisted while all other settings remain unchanged

**NOTE**: Update functionality already implemented in Phase 3 (T018), this phase adds comprehensive update testing

**STATUS: COMPLETED ✅ (Session 10)**

### Testing for User Story 6

- [x] T064 [US6] Add acceptance test for session duration update preserving other fields - **PASSING** ✅ (TestAccVMPolicy_updateSessionDuration, 55.17s, includes ImportState)
- [x] T065 [US6] Add acceptance test for access window update (change time range) - **PASSING** ✅ (TestAccVMPolicy_updateAccessWindow, 73.65s, includes ImportState)
- [x] T066 [US6] Add acceptance test for target rule updates (add FQDN rule while preserving existing) - **PASSING** ✅ (TestAccVMPolicy_updateTargets, 51.39s, includes ImportState)
- [x] T067 [US6] Add acceptance test for behavior updates (SSH username change, add RDP profile) - **PASSING** ✅ (TestAccVMPolicy_updateBehavior, 55.07s, includes ImportState)
- [x] T068 [US6] Add acceptance test verifying inline principals preserved when updated while assigned principals remain - **PASSING** ✅ (TestAccVMPolicy_updatePreservesPrincipals, 51.86s, includes ImportState)

**Implementation Notes (Session 10)**:
- **Drift Issue Fixed**: Changed `fqdn_rule` from `ListNestedBlock` to `SetNestedBlock` to prevent perpetual drift when API reorders rules
- **Files Modified**:
  - Schema: `internal/provider/vm_policy_resource.go:338` (ListNestedBlock → SetNestedBlock)
  - Model: `internal/models/vm_policy_models.go:80` (types.List → types.Set)
  - Read mapping: `internal/provider/vm_policy_resource.go:2005,2015,2057,2068` (ListValueFrom → SetValueFrom, type declarations)
  - Test assertions: Use `TestCheckTypeSetElemNestedAttrs` for order-independent checks
- **ImportState Verification**: All 5 tests include 3-step validation (Create → Update → ImportState)
- **Test Results**: All 5 tests passing (287.18s total)

**Checkpoint**: ✅ Policy updates fully tested - all configuration changes preserve unmodified elements, ImportState works after updates, no drift issues

---

## Phase 9: User Story 7 - Remove Access and Decommission Policies (Priority: P3)

**Goal**: Support complete policy lifecycle management including principal removal and policy deletion

**Independent Test**: Create policy → Assign principal → Remove assignment via Terraform → Delete policy → Verify policy no longer exists in SIA

**NOTE**: Delete functionality already implemented in Phase 3 (T019) and Phase 4 (T034), this phase adds comprehensive deletion testing

### Testing for User Story 7

**STATUS: All delete operations covered by existing tests ✅**

- [x] T069 [US7] Principal assignment removal - **COVERED** by `TestAccVMPolicyPrincipalAssignment_crud` (implicit Delete via Terraform destroy)
- [x] T070 [US7] Policy deletion - **COVERED** by all VM policy tests (implicit Delete via Terraform destroy, 404 handling verified in Delete() method lines 1155-1162)
- [x] T071 [US7] Drift detection on policy deletion - **COVERED** by Read() method (line 859-862 handles 404, removes from state) and `TestAccVMPolicy_driftDetection` test

**Implementation Evidence**:
- Delete() handles 404 gracefully: `internal/provider/vm_policy_resource.go:1155-1162`
- Read() detects drift (404): `internal/provider/vm_policy_resource.go:858-867`
- Principal assignment CRUD: `internal/provider/vm_policy_principal_assignment_resource_test.go:49-89`

**Checkpoint**: Complete lifecycle management tested - creation, updates, deletion all working

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, provider registration, validation, and final polish

- [x] T072 Register VM policy resource in `internal/provider/provider.go` (add to Resources method)
- [x] T073 Register principal assignment resource in `internal/provider/provider.go`
- [x] T074 [P] Create complete policy example in `examples/resources/cyberarksia_vm_policy/complete.tf` (all features: AWS + RDP + time windows + multiple principals)
- [ ] T075 [P] Update `examples/testing/TESTING-GUIDE.md` with VM policy CRUD validation scenarios
- [x] T076 Run `tfplugindocs generate` to create resource documentation
- [x] T077 [P] Update `CLAUDE.md` resource table with cyberarksia_vm_policy and cyberarksia_vm_policy_principal_assignment
- [x] T078 [P] Create implementation summary in `docs/development/vm-policy-implementation.md`
- [x] T079 Run `make validate` (format, lint, security checks)
- [ ] T080 Run full acceptance test suite: `TF_ACC=1 go test ./internal/provider -v -run TestAccVMPolicy`
- [ ] T081 Verify quickstart.md walkthrough produces working resources

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-9)**: All depend on Foundational phase completion
  - US1 (Phase 3) - P1: Can start after Foundational - No dependencies on other stories
  - US2 (Phase 4) - P1: Can start after Foundational - Integrates with US1 but independently testable
  - US3 (Phase 5) - P2: Can start after Foundational - Builds on US1 schema patterns
  - US4 (Phase 6) - P2: Can start after Foundational - Extends US1 behavior configuration
  - US5 (Phase 7) - P3: Can start after Foundational - Adds to US3 multi-cloud support
  - US6 (Phase 8) - P2: Testing phase only - depends on US1-US5 implementation
  - US7 (Phase 9) - P3: Testing phase only - depends on US1-US2 implementation
- **Polish (Phase 10)**: Depends on all desired user stories being complete

### User Story Dependencies

All user stories extend the base VM policy resource (US1):
- **US1 (P1)**: Foundation - FQDN/IP policies with SSH
- **US2 (P1)**: Assignment resource - depends on US1 for policy resource
- **US3 (P2)**: AWS targets - extends US1 schema
- **US4 (P2)**: RDP behavior - extends US1 behavior
- **US5 (P3)**: Azure/GCP targets - extends US1 schema (parallel with US3)
- **US6 (P2)**: Update testing - requires US1-US5 for comprehensive coverage
- **US7 (P3)**: Delete testing - requires US1-US2

### Within Each User Story

- Schema definition before CRUD methods
- Validators before ValidateConfig implementation
- Create/Read before Update/Delete
- SDK helpers before CRUD methods that use them
- Core implementation before tests
- Acceptance tests before examples

### Parallel Opportunities

- **Phase 1 (Setup)**: All tasks can run in parallel
- **Phase 2 (Foundational)**: T005, T006, T007 can run in parallel after T004
- **After Foundational**: US1 (P1), US3 (P2), US4 (P2), US5 (P3) schemas can be developed in parallel
- **Within US1**: T008-T009 sequential, then T010-T013 parallel, T014-T015 sequential, T016-T021 parallel, T022-T026 parallel, T027-T028 parallel
- **Within US2**: T029-T030 sequential, T031-T035 parallel if using separate functions
- **Within US5**: T056-T057 parallel, T060-T061 parallel, T062-T063 parallel
- **Phase 10 (Polish)**: T072-T073 sequential, then T074-T078 parallel, T079-T081 sequential

---

## Parallel Example: Foundational Phase

```bash
# After T004 completes, launch these together:
Task T005: "Implement custom enum validators in internal/validators/vm_validators.go"
Task T006: "Extend composite ID helpers in internal/provider/helpers/composite_ids.go"
Task T007: "Create VM policy principal assignment model in internal/models/vm_policy_models.go"
```

## Parallel Example: User Story 1 Schema Blocks

```bash
# After T009 completes, launch these together:
Task T010: "Add principals block to schema"
Task T011: "Add conditions attributes to schema"
Task T012: "Add behavior block to schema"
Task T013: "Add fqdn_ip_targets block to schema"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Basic FQDN/IP policies)
4. Complete Phase 4: User Story 2 (Principal assignments)
5. Complete Phase 10: Polish (register, document, validate)
6. **STOP and VALIDATE**: Test US1+US2 independently
7. Deploy/demo if ready

**MVP Delivers**: Complete FQDN/IP on-premises server access policy management with principal assignment capabilities

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 → Test independently → **MVP: Basic policies**
3. Add US2 → Test independently → **MVP+ Principal management**
4. Add US3 → Test independently → **V1.1: AWS cloud support**
5. Add US4 → Test independently → **V1.2: RDP support**
6. Add US5 → Test independently → **V1.3: Multi-cloud (Azure/GCP)**
7. Add US6-US7 testing → **V1.4: Production-ready**

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together (Phases 1-2)
2. Once Foundational is done:
   - Developer A: US1 (Phase 3) - Core policy resource
   - Developer B: US2 (Phase 4) - Assignment resource (after US1 schema done)
   - Developer C: US3 (Phase 5) - AWS targets (parallel with US4/US5)
   - Developer D: US4 (Phase 6) - RDP behavior (parallel with US3/US5)
3. Integration: US6-US7 testing phases
4. Final: Phase 10 polish together

---

## Summary

**Total Tasks**: 81 (78 completed ✅, 3 remaining)
**By User Story**:
- Setup: 3/3 tasks ✅
- Foundational: 4/4 tasks ✅ (BLOCKS all user stories)
- US1 (P1 - Basic policies): 26/26 tasks ✅ (implementation + tests)
- US2 (P1 - Principal assignment): 12/12 tasks ✅ (implementation + tests)
- US3 (P2 - AWS cloud): 7/7 tasks ✅
- US4 (P2 - RDP behavior): 20/20 tasks ✅ (includes schema fixes)
- US5 (P3 - Multi-cloud): 8/8 tasks ✅ (implementation + tests)
- US6 (P2 - Update testing): 5/5 tasks ✅ (Session 10 - includes drift fix)
- US7 (P3 - Delete testing): 3/3 tasks ✅
- Polish: 7/10 tasks (3 tasks pending: T075, T080, T081)

**Parallel Opportunities**: 28 tasks marked [P] can run in parallel with other tasks
**Independent Stories**: Each user story (US1-US7) can be tested independently
**MVP Scope**: US1 + US2 (34 tasks) delivers core on-premises VM access policy management

**Latest Update (Session 10)**: User Story 6 complete - all update tests passing with ImportState verification, drift issue resolved

**Format Validation**: ✅ All tasks follow checklist format with checkbox, ID, [P] marker (where applicable), [Story] label (for user story tasks), and file paths
