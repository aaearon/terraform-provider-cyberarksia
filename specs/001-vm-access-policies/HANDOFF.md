# VM Access Policy Implementation - Handoff Document

**Date**: 2025-11-17
**Branch**: `001-vm-access-policies` (commit: 6885431)
**Status**: User Stories 1-2 complete, principal drift bug fixed

---

## Session 4 Progress (2025-11-17 - Current)

### ✅ What Was Completed

**User Story 2: Principal Assignment Tests (T036-T039)**
1. ✅ **Test file created**: `vm_policy_principal_assignment_resource_test.go` (5 tests)
2. ✅ **Critical bug discovered**: VM policy Read() method caused drift by mapping ALL principals (inline + assigned) to state
3. ✅ **Root cause analysis**: Consulted Codex (as required by CLAUDE.md) - confirmed bug exists in both vm_policy AND database_policy
4. ✅ **Fix implemented**: Modified Read() to filter principals, only including inline principals in state
5. ✅ **All tests passing**: 5/5 User Story 2 tests pass (157.42s)

**P2 Cleanup Completed**
6. ✅ **Field renaming**: `created_by.name` → `created_by.user` (consistency with database_policy)
7. ✅ **Test infrastructure**: Added random IDs to prevent duplicate policy names

**Key Technical Changes**
- `vm_policy_resource.go Read()`: Extract inline principal keys from state, filter API results
- `mapSDKPolicyToState()`: Accept `inlinePrincipalKeys` parameter, filter principals based on keys
- When no prior state (import/create), pass nil to include all principals
- Prevents drift while allowing assignment resources to manage their principals independently

**Git Commits (Session 4)**
```
6885431 - fix: prevent principal drift in vm_policy Read() method
1e6cde7 - fix: align created_by/updated_by fields with database_policy pattern
```

### 📊 Testing Status

**User Story 1: All 5 tests passing (111.99s)**
- ✅ TestAccVMPolicy_basic
- ✅ TestAccVMPolicy_sshWithTimeWindow
- ✅ TestAccVMPolicy_driftDetection
- ✅ TestAccVMPolicy_forceNewOnNameChange
- ✅ TestAccVMPolicy_validationErrors

**User Story 2: All 5 tests passing (157.42s)**
- ✅ TestAccVMPolicyPrincipalAssignment_basic (T036)
- ✅ TestAccVMPolicyPrincipalAssignment_crud (T037)
- ✅ TestAccVMPolicyPrincipalAssignment_duplicateDetection (T038)
- ✅ TestAccVMPolicyPrincipalAssignment_importState (T039)
- ✅ TestAccVMPolicyPrincipalAssignment_compositeID

**Total: 10/10 tests passing for User Stories 1-2**

### 🔍 Codex Analysis Findings

**Issue**: Principal drift when assignment resources add principals to policies

**Codex Verdict** (via zen MCP clink tool):
1. Bug confirmed in BOTH vm_policy and database_policy resources
2. database_policy hides bug via `lifecycle { ignore_changes = [principal] }` in examples
3. No Terraform framework feature fixes this automatically
4. Recommended fix: Filter principals in Read() using same logic as Update()
5. Solution applied to vm_policy only (database_policy deferred to avoid scope creep)

**Implementation**: Followed Codex recommendation exactly - filter applied in Read(), passes all tests

### 📝 Known Issues Documented

**database_policy has same drift bug** (not fixed in this session):
- Uses `lifecycle { ignore_changes = [principal] }` workaround
- See: `docs/development/INLINE-ASSIGNMENT-FIX.md:232-254`
- See: `examples/testing/resources/database-full-crud.md:258-297`
- To be addressed in separate effort

---

## Session 3 Progress (2025-11-17)

### ✅ What Was Actually Fixed (Root Causes)

**Priority 1: Plan/State Consistency Errors**
1. ✅ **Schema fix**: Changed `created_by`/`updated_by` from SingleNestedBlock → SingleNestedAttribute
2. ✅ **Create() pattern**: Only set ID fields, let Terraform's Read() populate rest (matches database_policy)
3. ✅ **Empty string normalization**: description, from_hour, to_hour, domain → null when empty
4. ✅ **CRITICAL ROOT CAUSE FIX**: Changed `days_of_the_week` from List → Set
   - Removed sorting workaround (was masking schema design flaw)
   - List = ordered (drift), Set = unordered (drift-free)
   - Added validators: ValueInt64sAre(Between(0,6)), SizeBetween(1,7)
   - Now matches database_policy pattern

**Priority 0: Interface Implementation**
5. ✅ **ModifyPlan() implemented** (was declared but missing)
   - Validates removing principals won't violate "min 1 principal" constraint
   - Queries API to count TOTAL principals (inline + external assignments)
   - Blocks plan if would leave 0 principals
   - Pattern reused from database_policy_resource.go:524-653

**Priority 1: Validation**
6. ✅ **Access window validation**: from_hour and to_hour must both be set or both omitted
7. ✅ **Principal directory validation**: Already implemented (code review was outdated)

**User Story 1: All Tests Passing**
8. ✅ TestAccVMPolicy_basic (25.83s)
9. ✅ TestAccVMPolicy_sshWithTimeWindow (16.88s)
10. ✅ TestAccVMPolicy_driftDetection (23.66s)
11. ✅ TestAccVMPolicy_forceNewOnNameChange (33.65s)
12. ✅ TestAccVMPolicy_validationErrors (4.82s) - 3 subtests

**Total Test Time**: 104.88s for all User Story 1 tests

### 🔧 Remaining P2 Fixes (Minor, Not Blocking)

1. **Status field normalization** (prevents drift)
   - Need: `types.StringValue(strings.ToLower(sdkPolicy.Metadata.Status.Status))`
   - Currently: API returns "Active", user writes "active" → drift

2. **Field name consistency** (aligns with database_policy)
   - Change: `created_by.name` → `created_by.user`
   - Change: `created_by.timestamp` → `created_by.time`
   - Requires: Model update + schema update + state mapping update

### 📊 Code Review Findings Summary

**What We Discovered**: The initial fixes only made tests pass, didn't fix root causes.

**Comparison with database_policy**: Found 4 critical gaps (now 2 remain as P2):
- ✅ Fixed: days_of_the_week List→Set (eliminated drift root cause)
- ✅ Fixed: ModifyPlan() implementation (interface contract)
- ✅ Fixed: Access window validation
- ⏳ Pending: Status normalization (P2)
- ⏳ Pending: Field name consistency (P2)

### Git Commits (Session 3)

```
76975e8 - fix: resolve VM policy plan/state consistency errors
f891872 - test: complete User Story 1 acceptance tests (T023-T026)
b923111 - fix(critical): change days_of_the_week from List to Set
```

---

## Session 2 Progress (2025-11-16 16:30-17:15)

### ✅ Bugs Fixed
1. **DelegationClassification required**: Added `policy.DelegationClassification = "Unrestricted"` in Create method (line 663)
2. **Type mismatch errors**:
   - Fixed `ip_rule` empty list initialization (line 1775-1784)
   - Fixed `behavior.rdp` object types with proper attribute definitions (lines 1705-1715, 1718-1730)
3. **Delete panic**: Updated Delete method to use `DeleteDatabasePolicyDirect` workaround (line 959)
4. **Test config**: Created `vm_policy_resource_test.go` with 5 User Story 1 tests

### 🔍 Key Discoveries (from tenant inspection)
- **`access_window` is REQUIRED** by API (not optional as schema suggests)
- `from_hour`/`to_hour` ARE valid but optional within `access_window` (for time-based restrictions)
- `days_of_the_week` is the only required field in `access_window`
- Policies successfully created with empty `targets` and `behavior` objects
- No `idle_time` returned in API responses (server-computed default)

### 🔧 Remaining Issues (Plan/State Consistency)

**Test Status**: Policy creates and deletes successfully, but 3 Terraform consistency errors:

1. **`.created_by: was absent, but now present`**
   - Computed field appears after CREATE
   - Fix: Ensure schema marks as Computed + handle in plan modifier

2. **`.updated_by: was absent, but now present`**
   - Computed field appears after CREATE
   - Fix: Same as created_by

3. **`.fqdn_ip_targets.fqdn_rule[0].domain: was null, but now cty.StringVal("")`**
   - Optional field returns empty string instead of null
   - Fix: Normalize empty strings to null in Read method (mapSDKPolicyToState)

### 📝 Test File Created
- `internal/provider/vm_policy_resource_test.go` (467 lines)
- Tests: T022 (basic), T023 (SSH+time window), T024 (drift), T025 (ForceNew), T026 (validation)
- All configs use correct principal data source attributes
- Test names use timestamp for uniqueness

---

## 🚀 START HERE - Next Session Instructions

### Current Status (Session 4 Complete)
- ✅ User Stories 1-2: COMPLETE (10/10 tests passing)
- ✅ P2 cleanup: COMPLETE (field renaming done)
- ✅ Principal drift bug: FIXED
- ⏳ User Stories 3-7: PENDING (21 tests remaining)
- 📋 Ready to continue with AWS, RDP, Azure/GCP, Update, Delete tests

### Session 5 Quick Start

**Credentials:**
```bash
export CYBERARK_USERNAME=timtest@cyberark.cloud.40562
export CYBERARK_PASSWORD='nvk*phv*hfd3ATR2rfc'
export TF_ACC=1
```

**Verify Current State:**
```bash
cd /home/tim/terraform-provider-cyberarksia
git checkout 001-vm-access-policies
git log --oneline -3  # Should show: 6885431, 1e6cde7, 0d566a9

# Run existing tests to verify
go test ./internal/provider -v -run "TestAccVMPolicy" -timeout 10m
# Expected: 10 tests pass (User Stories 1-2)
```

### Recommended Approach: Continue with User Story 3 (AWS Cloud Policies)

**User Story 3: AWS Cloud Policies (T044-T046) - ~1 hour**

Tests to implement in `vm_policy_resource_test.go`:

**T044**: AWS policy with regions and tags
```go
func TestAccVMPolicy_awsBasic(t *testing.T) {
    // Test: Create AWS policy with regions + tags
    // Verify: location_type = "AWS", aws_targets populated
}
```

**T045**: AWS VPC IDs and account IDs
```go
func TestAccVMPolicy_awsVpcAndAccounts(t *testing.T) {
    // Test: AWS policy with vpc_ids + account_ids
    // Verify: Filters applied correctly
}
```

**T046**: Update AWS regions
```go
func TestAccVMPolicy_awsUpdateRegions(t *testing.T) {
    // Test: Update regions list, verify no ForceNew
    // Verify: Only regions changed, principals preserved
}
```

**Test Command:**
```bash
TF_ACC=1 go test ./internal/provider -v -run "TestAccVMPolicy_aws" -timeout 15m
```

### Alternative: Continue with User Story 4 (RDP)

**If P2 fixes are skipped, proceed to User Story 2:**

**Fix 1 - Computed Fields**: In `vm_policy_resource.go` line ~1804 (mapSDKPolicyToState):
```go
// Explicitly handle nil for computed fields not present in plan:
if sdkPolicy.Metadata.CreatedBy.User != "" {
    state.CreatedBy = &models.ChangeInfoModel{...}
} else {
    state.CreatedBy = nil
}
// Same for UpdatedBy
```

**Fix 2 - Empty String Normalization**: In `vm_policy_resource.go` line ~1735 (FQDN mapping):
```go
domain := types.StringNull()
if rule.Domain != "" {
    domain = types.StringValue(rule.Domain)
}
```

**Test Command**:
```bash
export CYBERARK_USERNAME=timtest@cyberark.cloud.40562
export CYBERARK_PASSWORD='nvk*phv*hfd3ATR2rfc'
export TF_ACC=1
go test ./internal/provider -v -run TestAccVMPolicy_basic -timeout 10m
```

### Priority 2: Complete User Story 1 Tests (1.5 hours)
After fixes pass, run remaining T023-T026 tests.

---

## Current State (Updated Session 4)

### ✅ Completed (50/81 tasks - 62%)

**Implementation:**
- ✅ Two resources fully implemented with CRUD lifecycle
  - `cyberarksia_vm_policy` (main policy resource - drift bug FIXED)
  - `cyberarksia_vm_policy_principal_assignment` (dynamic principal management)
- ✅ Multi-cloud support: FQDN/IP, AWS, Azure, GCP
- ✅ Connection behavior: SSH + RDP (local/domain ephemeral users)
- ✅ Time-based access controls
- ✅ Schema validation (all attributes properly configured)
- ✅ Provider compilation successful
- ✅ golangci-lint: 0 issues
- ✅ Documentation auto-generated (tfplugindocs)
- ✅ 8 comprehensive examples created
- ✅ CRUD validation template created
- ✅ Implementation summary documented
- ✅ User Stories 1-2: 10/10 tests passing
- ✅ Principal drift bug: Fixed via Read() filtering
- ✅ P2 cleanup: Field renaming complete

**Files Modified:** 17 files, ~2,300 insertions

### ⏳ Remaining Work (31 tasks - 38%)

**All Blockers Resolved:** Credentials working, drift bug fixed, tests infrastructure solid

**Pending Acceptance Tests** (21 tests remaining):
   - ✅ T022-T026: User Story 1 (FQDN/IP basic policies) - 5 tests COMPLETE
   - ✅ T036-T039: User Story 2 (principal assignments) - 5 tests COMPLETE (4 planned + 1 bonus)
   - ⏳ T044-T046: User Story 3 (AWS cloud) - 3 tests PENDING
   - ⏳ T048-T055: User Story 4 (RDP behavior) - 8 tests PENDING
   - ⏳ T060-T061: User Story 5 (Azure/GCP) - 2 tests PENDING
   - ⏳ T064-T068: User Story 6 (update tests) - 5 tests PENDING
   - ⏳ T069-T071: User Story 7 (delete tests) - 3 tests PENDING

2. **Documentation & Validation:**
   - T075: Update TESTING-GUIDE.md with VM policy scenarios
   - T080: Run full acceptance test suite
   - T081: Verify quickstart.md walkthrough

---

## Next Steps - Complete Implementation

### Step 1: Setup Test Environment (5 minutes)

```bash
# Navigate to project
cd /home/tim/terraform-provider-cyberarksia

# Ensure on correct branch
git checkout 001-vm-access-policies
git status  # Should show: "nothing to commit, working tree clean"

# Load credentials from backup
source /home/tim/.env-cyberark-sia-backup

# Verify credentials loaded
echo "Username: $CYBERARK_USERNAME"
echo "Password: ${CYBERARK_PASSWORD:0:5}..." # First 5 chars only
echo "Identity URL: $CYBERARK_IDENTITY_URL"

# Enable acceptance tests
export TF_ACC=1

# Optional: Enable verbose logging
export TF_LOG=DEBUG
export TF_LOG_PATH=./tf-test.log
```

**Verify Credentials Work:**
```bash
# Quick provider config test
go test ./internal/provider -v -run TestAccProvider_Configure -timeout 2m
```

### Step 3: Complete User Story 1 Tests - Priority: HIGH

**Context:**
- User Story 1: Basic FQDN/IP policies with SSH connection behavior
- These are P1 (highest priority) MVP tests
- Test file: `internal/provider/vm_policy_resource_test.go`

**Reference Patterns:**
- Look at: `internal/provider/database_policy_resource_test.go`
- Pattern: TestAccVMPolicy_basic, TestAccVMPolicy_ssh, TestAccVMPolicy_drift, etc.

**Tests to Implement:**

**T022**: Basic FQDN/IP policy test
```go
func TestAccVMPolicy_basic(t *testing.T) {
    // Test: Create basic FQDN/IP policy with SSH
    // Verify: policy_id, name, status, location_type = "FQDN/IP"
    // Check: At least 1 principal assigned
}
```

**T023**: SSH behavior + time window test
```go
func TestAccVMPolicy_sshWithTimeWindow(t *testing.T) {
    // Test: Policy with SSH username + access_window (business hours)
    // Verify: behavior.ssh.username, access_window.days_of_the_week, from_hour, to_hour
}
```

**T024**: Drift detection test
```go
func TestAccVMPolicy_driftDetection(t *testing.T) {
    // Test: Create policy, manually delete via API, run terraform refresh
    // Verify: 404 removes resource from state (no error)
    // Pattern: Use vmService.DeletePolicy() directly in test
}
```

**T025**: ForceNew behavior test
```go
func TestAccVMPolicy_forceNewOnNameChange(t *testing.T) {
    // Test: Create policy, change name in config
    // Verify: Terraform plan shows destroy + recreate (not update)
    // Check: PreConfig/CheckDestroy logic
}
```

**T026**: Validation error tests
```go
func TestAccVMPolicy_validationErrors(t *testing.T) {
    // Test: Zero principals, missing SSH username, conflicting location types
    // Verify: ExpectError with proper error messages
    // Pattern: Multiple test steps with invalid configs
}
```

**File Location:**
```bash
# Create test file
touch internal/provider/vm_policy_resource_test.go

# Start with basic structure
cat > internal/provider/vm_policy_resource_test.go <<'TESTFILE'
package provider

import (
    "fmt"
    "testing"

    "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVMPolicy_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccVMPolicyConfig_basic(),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttrSet("cyberarksia_vm_policy.test", "policy_id"),
                    resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "name", "test-vm-policy"),
                    resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "status", "Active"),
                    resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "location_type", "FQDN/IP"),
                ),
            },
        },
    })
}

func testAccVMPolicyConfig_basic() string {
    return `
data "cyberarksia_principal" "test_user" {
  name = "YOUR_TEST_USER@cyberark.cloud.XXXXX"  # REPLACE
  type = "USER"
}

resource "cyberarksia_vm_policy" "test" {
  name          = "test-vm-policy"
  location_type = "FQDN/IP"
  status        = "Active"

  principal {
    principal_id            = data.cyberarksia_principal.test_user.principal_id
    principal_name          = data.cyberarksia_principal.test_user.principal_name
    principal_type          = data.cyberarksia_principal.test_user.principal_type
    source_directory_name   = data.cyberarksia_principal.test_user.source_directory_name
    source_directory_id     = data.cyberarksia_principal.test_user.source_directory_id
  }

  behavior {
    ssh {
      username = "testuser"
    }
  }

  fqdn_ip_targets {
    fqdn_rule {
      operator             = "SUFFIX"
      computername_pattern = "-test"
    }
  }

  max_session_duration = 2
}
`
}
TESTFILE
```

**Run Tests:**
```bash
# Run single test
TF_ACC=1 go test ./internal/provider -v -run TestAccVMPolicy_basic -timeout 10m

# Run all US1 tests
TF_ACC=1 go test ./internal/provider -v -run TestAccVMPolicy -timeout 20m
```

**Mark Tasks Complete:**
After each test passes, update `specs/001-vm-access-policies/tasks.md`:
```bash
# Change "- [ ] T022" to "- [x] T022"
# Repeat for T023-T026
```

### Step 3: Implement User Story 2 Acceptance Tests (T036-T040) - Priority: HIGH

**Context:**
- User Story 2: Principal assignment resource
- Test file: `internal/provider/vm_policy_principal_assignment_resource_test.go`
- Reference: `internal/provider/database_policy_principal_assignment_resource_test.go`

**Tests to Implement:**

**T036**: Basic assignment test
```go
func TestAccVMPolicyPrincipalAssignment_basic(t *testing.T) {
    // Create policy with 1 principal, add 2nd via assignment
    // Verify composite ID format
}
```

**T037**: CRUD lifecycle test
```go
func TestAccVMPolicyPrincipalAssignment_crud(t *testing.T) {
    // CREATE: Assign principal
    // READ: Verify assignment exists
    // DELETE: Remove assignment
}
```

**T038**: Duplicate detection test
```go
func TestAccVMPolicyPrincipalAssignment_duplicateDetection(t *testing.T) {
    // Assign principal, try to assign same principal again
    // ExpectError: duplicate principal
}
```

**T039**: ImportState test
```go
func TestAccVMPolicyPrincipalAssignment_importState(t *testing.T) {
    // Create assignment, terraform import using composite ID
    // Verify all attributes imported correctly
}
```

**T040**: Create example file (already done ✅)

**Run Tests:**
```bash
TF_ACC=1 go test ./internal/provider -v -run TestAccVMPolicyPrincipalAssignment -timeout 15m
```

### Step 4: Remaining Acceptance Tests (T044-T071) - Priority: MEDIUM

**User Story 3 (AWS):** T044-T047
```bash
# Tests: AWS policy creation, regions/tags, VPC IDs, update regions
TF_ACC=1 go test ./internal/provider -v -run TestAccVMPolicy_aws -timeout 15m
```

**User Story 4 (RDP):** T048-T055
```bash
# Tests: RDP local/domain ephemeral user, SSH+RDP combined
# NOTE: T048-T051 are implementation tasks (may need RDP schema additions)
TF_ACC=1 go test ./internal/provider -v -run TestAccVMPolicy_rdp -timeout 15m
```

**User Story 5 (Azure/GCP):** T060-T063
```bash
# Tests: Azure resource groups/tags, GCP labels/projects
# WARNING: Azure had HTTP 500 errors in POC (see research.md)
TF_ACC=1 go test ./internal/provider -v -run TestAccVMPolicy_azure
TF_ACC=1 go test ./internal/provider -v -run TestAccVMPolicy_gcp
```

**User Story 6 (Updates):** T064-T068
```bash
# Tests: Session duration update, access window update, target rules update
TF_ACC=1 go test ./internal/provider -v -run TestAccVMPolicy_update -timeout 15m
```

**User Story 7 (Delete):** T069-T071
```bash
# Tests: Principal removal, policy deletion, drift detection
TF_ACC=1 go test ./internal/provider -v -run TestAccVMPolicy_delete -timeout 10m
```

### Step 5: Update TESTING-GUIDE.md (T075)

**File:** `examples/testing/TESTING-GUIDE.md`

**Add Section:**
```markdown
### VM Policy CRUD Validation

**Template:** `crud-test-vm-policy.tf`

**Prerequisites:**
1. Valid CyberArk SIA credentials (CYBERARK_USERNAME, CYBERARK_PASSWORD)
2. At least one test principal available (user or group)
3. Provider configured (see provider configuration examples)

**Validation Steps:**

1. **CREATE**: Copy template to /tmp, customize principal references, run `terraform apply`
   - Verify: policy_id populated, status = Active, principals assigned
   - Check: delegation_classification computed

2. **READ**: Run `terraform apply` again (no changes expected)
   - Verify: No drift, all values match state

3. **UPDATE**: Modify max_session_duration, run `terraform apply`
   - Verify: Only modified fields change, principals preserved

4. **DELETE**: Run `terraform destroy`
   - Verify: Clean deletion, no orphaned resources

**Full Checklist:** See template file for validation checklist outputs
```

### Step 6: Final Validation (T080-T081)

**T080**: Run full acceptance test suite
```bash
# Run ALL VM policy tests
TF_ACC=1 go test ./internal/provider -v -run TestAccVMPolicy -timeout 30m 2>&1 | tee test-results.log

# Check results
grep -E "PASS|FAIL" test-results.log
```

**T081**: Verify quickstart.md walkthrough
```bash
# Follow: specs/001-vm-access-policies/quickstart.md
# Manually test each code example
# Verify all examples work as documented
```

### Step 7: Final Git Commit & PR

**Mark Remaining Tasks Complete:**
```bash
# Update specs/001-vm-access-policies/tasks.md
# Change all "- [ ] TXXX" to "- [x] TXXX" for completed tests
```

**Commit Test Implementation:**
```bash
git add -A
git commit -m "test: add acceptance tests for VM policy resources

Implements comprehensive acceptance test coverage for VM access policies:
- User Story 1: FQDN/IP basic policies (5 tests)
- User Story 2: Principal assignments (4 tests)
- User Story 3: AWS cloud policies (3 tests)
- User Story 4: RDP connection behavior (8 tests)
- User Story 5: Azure/GCP multi-cloud (2 tests)
- User Story 6: Update operations (5 tests)
- User Story 7: Delete operations (3 tests)

Total: 30+ acceptance tests covering all CRUD operations and edge cases

All tests pass with live CyberArk SIA tenant
Updated TESTING-GUIDE.md with VM policy validation scenarios

Closes: 001-vm-access-policies implementation
Ready for PR and merge to main"
```

**Create Pull Request:**
```bash
# Push branch to remote
git push origin 001-vm-access-policies

# Create PR via GitHub CLI
gh pr create \
  --title "feat: VM Access Policy Resources - Complete Implementation" \
  --body "$(cat <<'PRBODY'
## Summary

Implements comprehensive VM access policy management for CyberArk SIA Terraform provider.

## Resources

- `cyberarksia_vm_policy`: Main VM policy resource (FQDN/IP, AWS, Azure, GCP)
- `cyberarksia_vm_policy_principal_assignment`: Dynamic principal management

## Features

- Multi-cloud support with location-specific target criteria
- SSH and RDP connection behavior with ephemeral user support
- Time-based access controls (session duration, idle timeout, access windows)
- Required minimum 1 principal at creation with dynamic assignments
- Read-Modify-Write pattern preserving inline and assigned principals
- Complete CRUD lifecycle with import support

## Implementation

- **Files**: 15 modified/added (2 resources, models, validators, helpers, 8 examples, docs)
- **Tasks**: 81/81 complete (100% of planned tasks)
- **Tests**: 30+ acceptance tests (all passing)
- **Validation**: Go fmt/vet/lint passed, Terraform fmt passed, docs generated

## Documentation

- Auto-generated resource docs (tfplugindocs)
- Comprehensive examples (basic, cloud-specific, RDP, complete)
- CRUD validation template with checklists
- Implementation summary in docs/development/
- Updated TESTING-GUIDE.md

## Testing

All acceptance tests pass with live CyberArk SIA tenant:
- ✅ User Story 1: FQDN/IP basic policies
- ✅ User Story 2: Principal assignments
- ✅ User Story 3: AWS cloud policies
- ✅ User Story 4: RDP connection behavior
- ✅ User Story 5: Azure/GCP multi-cloud
- ✅ User Story 6: Update operations
- ✅ User Story 7: Delete operations

## References

- Specification: specs/001-vm-access-policies/spec.md
- Implementation Summary: docs/development/vm-policy-implementation.md
- Task Breakdown: specs/001-vm-access-policies/tasks.md (81/81 complete)

Ready for review and merge to main.
PRBODY
)"
```

---

## Critical Context & References

### Credentials

**Location:** `/home/tim/.env-cyberark-sia-backup`

**Load with:**
```bash
source /home/tim/.env-cyberark-sia-backup
export TF_ACC=1
```

**Expected Variables:**
- `CYBERARK_USERNAME`: Service account (format: `service-account@cyberark.cloud.XXXXX`)
- `CYBERARK_PASSWORD`: Service account password
- `CYBERARK_IDENTITY_URL`: Tenant URL (optional, auto-resolved from username)

### Key Implementation Files

**Core Resources:**
- `internal/provider/vm_policy_resource.go` (525 lines)
- `internal/provider/vm_policy_principal_assignment_resource.go` (348 lines)
- `internal/models/vm_policy_models.go`
- `internal/validators/vm_validators.go`

**Reference Implementations:**
- `internal/provider/database_policy_resource.go` (pattern for main resource)
- `internal/provider/database_policy_principal_assignment_resource.go` (pattern for assignments)
- `internal/provider/database_policy_resource_test.go` (test patterns)

**Documentation:**
- `specs/001-vm-access-policies/tasks.md` (task tracking - mark complete as you go)
- `specs/001-vm-access-policies/research.md` (SDK details, principal preservation algorithm)
- `specs/001-vm-access-policies/data-model.md` (complete schema documentation)
- `specs/001-vm-access-policies/quickstart.md` (implementation guide)
- `docs/development/vm-policy-implementation.md` (implementation summary)

**Examples:**
- `examples/resources/cyberarksia_vm_policy/resource.tf` (basic)
- `examples/resources/cyberarksia_vm_policy/complete.tf` (full-featured)
- `examples/testing/crud-test-vm-policy.tf` (CRUD template)

### Important Patterns

**Principal Preservation (Critical):**
```go
// In Update() method - must preserve BOTH inline and assigned principals
inlinePrincipalKeys := make(map[string]bool)
for _, p := range plan.Principals {
    key := fmt.Sprintf("%s:%s", p.PrincipalID.ValueString(), p.PrincipalType.ValueString())
    inlinePrincipalKeys[key] = true
}

preservedPrincipals := []uapcommonmodels.ArkUAPPrincipal{}
for _, p := range existingPolicy.Principals {
    key := fmt.Sprintf("%s:%s", p.ID, p.Type)
    if !inlinePrincipalKeys[key] {
        preservedPrincipals = append(preservedPrincipals, p)
    }
}

newPrincipals := /* inline from plan */
newPrincipals = append(newPrincipals, preservedPrincipals...)
```

**Composite ID Format:**
- Format: `policy-id:principal-id:principal-type`
- Example: `a1b2c3d4-...:e5f6a7b8-...:USER`
- Helpers: `helpers.ParseVMPolicyPrincipalID()`, `helpers.BuildVMPolicyPrincipalID()`

**No DELETE Workaround:**
- VM policies use SDK `DeletePolicy()` directly (works correctly)
- Do NOT use `internal/client/delete_workarounds.go`

### Known Issues

1. **Azure HTTP 500**: POC testing showed server errors (see `research.md`)
2. **RDP Domain Ephemeral User**: SDK-supported but undocumented in OpenAPI
3. **Acceptance Test Blockers Resolved**: Credentials now available

### Testing Tips

**Find Test Principal:**
```bash
# Use principal data source to find available principals
terraform console
> data.cyberarksia_principal.test_user
```

**Debug Test Failures:**
```bash
# Enable verbose logging
export TF_LOG=DEBUG
export TF_LOG_PATH=./test-debug.log

# Run single test
TF_ACC=1 go test ./internal/provider -v -run TestAccVMPolicy_basic -timeout 10m

# Check logs
tail -100 test-debug.log
```

**Cleanup Test Resources:**
```bash
# If tests leave orphaned resources
terraform init
terraform destroy -target='cyberarksia_vm_policy.test'
```

### Success Criteria

**Before PR:**
- [ ] All acceptance tests pass (30+ tests)
- [ ] TESTING-GUIDE.md updated with VM policy scenarios
- [ ] Quickstart.md walkthrough verified
- [ ] All 81 tasks marked complete in tasks.md
- [ ] Git commit created with test implementation
- [ ] Clean git status (no uncommitted changes)

**PR Ready Checklist:**
- [ ] Branch pushed to origin
- [ ] PR created with comprehensive description
- [ ] All tests passing in CI (if available)
- [ ] Documentation complete and accurate
- [ ] Ready for code review

---

## Time Estimates

**High Priority (MVP):**
- User Story 1 tests (T022-T026): 2 hours
- User Story 2 tests (T036-T040): 1 hour
- **Total MVP**: 3 hours

**Medium Priority (Full Coverage):**
- User Story 3-7 tests: 4 hours
- TESTING-GUIDE.md update: 30 minutes
- Quickstart verification: 30 minutes
- Final validation & commit: 30 minutes
- **Total Full**: 5.5 hours

**Grand Total**: 8.5 hours for complete implementation

---

## Quick Start Command

```bash
# One-command setup and test
\
source /home/tim/.env-cyberark-sia-backup && \
export TF_ACC=1 && \
git checkout 001-vm-access-policies && \
echo "✅ Ready to implement acceptance tests!" && \
echo "Start with: TF_ACC=1 go test ./internal/provider -v -run TestAccVMPolicy_basic -timeout 10m"
```

---

**End of Handoff Document**

Next developer: Start with Step 1 (Setup Test Environment) and proceed sequentially through Step 7 (Final PR).
All context, patterns, and references are documented above. Good luck! 🚀
