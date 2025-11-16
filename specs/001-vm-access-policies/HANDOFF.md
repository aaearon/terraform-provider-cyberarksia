# VM Access Policy Implementation - Handoff Document

**Date**: 2025-11-16  
**Branch**: `001-vm-access-policies` (commit: b2da840)  
**Status**: Core implementation complete, acceptance tests pending

---

## Current State

### ✅ Completed (43/81 tasks)

**Implementation:**
- ✅ Two resources fully implemented with CRUD lifecycle
  - `cyberarksia_vm_policy` (main policy resource)
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
- ✅ Git commit created (clean working tree)

**Files Modified:** 15 files, 1,769 insertions

### ⏳ Remaining Work (38 tasks)

**Primary Blocker Resolved:** Live CyberArk SIA tenant credentials available

**Pending Tasks:**
1. **Acceptance Tests** (26 tests total):
   - T022-T026: User Story 1 (FQDN/IP basic policies) - 5 tests
   - T036-T039: User Story 2 (principal assignments) - 4 tests
   - T044-T046: User Story 3 (AWS cloud) - 3 tests
   - T048-T055: User Story 4 (RDP behavior) - 8 tests
   - T060-T061: User Story 5 (Azure/GCP) - 2 tests
   - T064-T068: User Story 6 (update tests) - 5 tests
   - T069-T071: User Story 7 (delete tests) - 3 tests

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

### Step 2: Implement User Story 1 Acceptance Tests (T022-T026) - Priority: HIGH

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
