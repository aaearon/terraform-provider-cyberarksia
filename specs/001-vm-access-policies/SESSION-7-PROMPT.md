# Session 7: Fix RDP Schema + Complete Remaining User Stories

## Context

You are continuing the implementation of VM Access Policy resources for the CyberArk SIA Terraform provider. Read the complete handoff document for context:

**Handoff Location**: `/home/tim/terraform-provider-cyberarksia/specs/001-vm-access-policies/HANDOFF.md`

**Current Status** (from handoff):
- Branch: `001-vm-access-policies` (commit: 7bf24fb)
- Progress: User Stories 1-3 complete (13/13 tests passing)
- **BLOCKED**: User Story 4 has schema issue preventing RDP-only policies
- User Stories 5-7: Not started

## Your Mission

Fix the Terraform Plugin Framework schema limitation and complete all remaining User Stories.

---

## Phase 1: Schema Refactor (BLOCKER - Do This First)

### Problem Statement
RDP-only and SSH-only policies fail because `SingleNestedBlock` with `Required` attributes makes the parent block effectively required. See HANDOFF.md Session 6 for full details.

### API Evidence
RDP-only policy structure (from `/home/tim/terraform-provider-cyberarksia/specs/001-vm-access-policies/rdp-only-api-structure.json`):
```json
"behavior": {
  "rdp_profile": {
    "local_ephemeral_user": {
      "assign_groups": ["Power Users"],
      "enable_ephemeral_user_reconnect": false
    }
  }
}
```
Note: `ssh_profile` key is **completely omitted** (not null).

### Tasks (T056-T066 from tasks.md)

**Step 1: Refactor Schema (T056-T059)**
1. Change `behavior.ssh` from `SingleNestedBlock` → `SingleNestedAttribute` (line 277)
2. Change `behavior.rdp` from `SingleNestedBlock` → `SingleNestedAttribute` (line 289)
3. Change `rdp.local_ephemeral_user` from `SingleNestedBlock` → `SingleNestedAttribute` (line 292)
4. Change `rdp.domain_ephemeral_user` from `SingleNestedBlock` → `SingleNestedAttribute` (line 306)

**File**: `internal/provider/vm_policy_resource.go`

**Reference Documentation**:
- HashiCorp Docs: https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes/single-nested
- GitHub Issue: https://github.com/hashicorp/terraform-plugin-framework/issues/740

**Step 2: Update Models if Needed (T060)**
Check if `internal/models/vm_policy_models.go` BehaviorModel needs updates after schema change.

**Step 3: Verify Mapping Logic (T061-T062)**
Ensure these functions still work correctly:
- `buildSDKBehavior()` - Terraform → SDK conversion
- `mapSDKPolicyToState()` - SDK → Terraform conversion

**Step 4: Run Tests (T064-T066)**
```bash
export CYBERARK_USERNAME="timtest@cyberark.cloud.40562"
export CYBERARK_PASSWORD='nvk*phv*hfd3ATR2rfc'
export TF_ACC=1

# Verify existing tests still pass
go test ./internal/provider -v -run "TestAccVMPolicy" -timeout 30m

# Expected: All 21 tests pass (13 existing + 8 RDP)
```

### Success Criteria
- [ ] RDP-only policies work (no SSH block required)
- [ ] SSH-only policies work (no RDP block required)  
- [ ] SSH+RDP combined policies still work
- [ ] All 21 tests pass
- [ ] Provider compiles without errors

---

## Phase 2: Complete Remaining User Stories (Optional - If Time Permits)

### User Story 5: Azure/GCP Cloud Policies (T060-T063)

**Reference**: `specs/001-vm-access-policies/spec.md` §2.3

Add acceptance tests for Azure and GCP targets:
- Azure: resource_groups, subscriptions, vnets, tags
- GCP: projects, labels, regions

**File**: `internal/provider/vm_policy_resource_test.go`

**Test Commands**:
```bash
go test ./internal/provider -v -run "TestAccVMPolicy_azure" -timeout 15m
go test ./internal/provider -v -run "TestAccVMPolicy_gcp" -timeout 15m
```

### User Story 6: Update Operations (T064-T068)

Test update scenarios:
- Session duration updates
- Access window changes
- Target rule modifications
- Verify no ForceNew on updatable fields

### User Story 7: Delete Operations (T069-T071)

Test deletion scenarios:
- Principal removal
- Policy deletion
- Drift detection

---

## Environment Setup

```bash
# Navigate to project
cd /home/tim/terraform-provider-cyberarksia

# Verify branch
git checkout 001-vm-access-policies
git log --oneline -2  # Should show: 7bf24fb, f6087b6

# Load credentials
export CYBERARK_USERNAME="timtest@cyberark.cloud.40562"
export CYBERARK_PASSWORD='nvk*phv*hfd3ATR2rfc'
export TF_ACC=1

# Verify current test baseline (before schema fix)
go test ./internal/provider -v -run "TestAccVMPolicy" -timeout 30m
# Expected: 13/21 tests pass (User Stories 1-3)
```

---

## Key References

**Must Read First**:
1. `/home/tim/terraform-provider-cyberarksia/specs/001-vm-access-policies/HANDOFF.md` (Session 6 section)
2. `/home/tim/terraform-provider-cyberarksia/specs/001-vm-access-policies/tasks.md` (T056-T066)
3. `/home/tim/terraform-provider-cyberarksia/specs/001-vm-access-policies/rdp-only-api-structure.json`

**Schema Files**:
- `internal/provider/vm_policy_resource.go` (lines 274-330)
- `internal/models/vm_policy_models.go`

**Test File**:
- `internal/provider/vm_policy_resource_test.go`

**Existing RDP Tests** (ready to run after schema fix):
- TestAccVMPolicy_rdpLocalEphemeral (T048)
- TestAccVMPolicy_rdpDomainEphemeral (T049)
- TestAccVMPolicy_sshAndRdp (T050) - Already passing ✅
- TestAccVMPolicy_rdpUpdate (T051)
- TestAccVMPolicy_rdpWithTimeWindow (T052)
- TestAccVMPolicy_rdpWithAWSTargets (T053)
- TestAccVMPolicy_rdpMultipleGroups (T054)
- TestAccVMPolicy_rdpReconnectSettings (T055)

---

## Validation Checklist

After schema refactor:

**Build & Compile**:
- [ ] `go build` succeeds without errors
- [ ] `golangci-lint run` passes
- [ ] `go vet ./...` passes

**Test Suite**:
- [ ] All 13 existing tests still pass (User Stories 1-3)
- [ ] 7 RDP-only tests now pass (previously blocked)
- [ ] 1 SSH+RDP test still passes (was already working)
- [ ] **Total: 21/21 tests passing**

**Manual Verification** (using ark CLI or SIA UI):
- [ ] Create RDP-only policy via Terraform (no SSH block)
- [ ] Create SSH-only policy via Terraform (no RDP block)
- [ ] Verify policies in SIA UI match Terraform config

---

## Deliverables

### Minimum (Phase 1 Only)
1. Schema refactored to SingleNestedAttribute
2. All 21 tests passing (13 + 8 RDP)
3. Update tasks.md (mark T056-T066 complete)
4. Update HANDOFF.md with Session 7 progress
5. Git commit with message:
   ```
   fix: refactor behavior schema to support RDP-only and SSH-only policies

   - Changed behavior.ssh/rdp from SingleNestedBlock to SingleNestedAttribute
   - Changed nested RDP blocks to SingleNestedAttribute
   - Fixed Terraform Plugin Framework limitation (#740)
   - All 21 tests now passing (13 existing + 8 RDP)

   Fixes T056-T066
   User Story 4: Complete ✅
   ```

### Stretch Goal (Phase 2)
1. User Story 5 tests implemented (Azure/GCP)
2. User Story 6 tests implemented (Updates)
3. User Story 7 tests implemented (Deletes)
4. All 30+ tests passing

---

## Important Notes

- **DO NOT** create workarounds - fix the schema properly
- **ALWAYS** run existing tests after schema changes to catch regressions
- **USE** the ark CLI (`/home/tim/go/bin/ark exec uap vm`) to verify API behavior if needed
- **FOLLOW** TDD principles from CLAUDE.md
- **UPDATE** documentation continuously as you work

---

## Success Measurement

**Phase 1 Success**: All RDP tests pass
```bash
go test ./internal/provider -v -run "TestAccVMPolicy_rdp" -timeout 20m
# Expected: 8/8 tests pass
```

**Full Success**: All VM policy tests pass
```bash
go test ./internal/provider -v -run "TestAccVMPolicy" -timeout 30m
# Expected: 30+ tests pass (depending on how many User Stories completed)
```

---

## Getting Help

If you encounter issues:
1. Check HANDOFF.md Session 6 for detailed root cause analysis
2. Review rdp-only-api-structure.json for API expectations
3. Compare with existing database_policy schema patterns
4. Search Terraform Plugin Framework docs for SingleNestedAttribute examples

Good luck! 🚀
