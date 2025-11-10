# Target Set Resource Testing

**Part of**: [Comprehensive CRUD Testing Guide](../TESTING-GUIDE.md)  
**Last Updated**: See git history

---

## Target Set Resource Testing

### Overview

Target sets define logical groupings of virtual machines and servers that share common access credentials for Just-In-Time (JIT) privileged access. This section covers CRUD testing for the `cyberarksia_target_set` resource.

**Template**: [`crud-test-target-set.tf`](./crud-test-target-set.tf)

**Test Scope**: Target set lifecycle with all matching patterns (Domain, Suffix, Target)
**Duration**: 10-15 minutes
**Prerequisites**: Existing VM secret resource

### Resources Validated
1. `cyberarksia_virtual_machine_secret` - VM credentials (ProvisionerUser or PCloudAccount)
2. `cyberarksia_target_set` - VM/server target set with matching patterns

### Key Features Tested

**Matching Patterns**:
- **Domain**: Matches all servers in a domain (e.g., `*.example.com`)
- **Suffix**: Matches servers with hostname suffix (e.g., `*.dc1.example.com`)
- **Target**: Matches specific server hostname

**Critical Behaviors**:
- ✅ Type changes (Domain ↔ Suffix ↔ Target) without resource recreation
- ✅ provision_format clearing prevention (plan-time error)
- ✅ Forward slash warning in names (deletion will fail with 403)
- ✅ In-place rename with ID following name
- ✅ Certificate validation toggle

### Testing Workflow (15-20 minutes)

#### Phase 1: Setup (2 minutes)

```bash
# 1. Create timestamped working directory
export TEST_DIR="/tmp/sia-crud-validation-target-set-$(date +%Y%m%d-%H%M%S)"
mkdir -p $TEST_DIR
cd $TEST_DIR

# 2. Copy target set template
cp ~/terraform-provider-cyberarksia/examples/testing/crud-test-target-set.tf .

# 3. Export environment variables (recommended)
export CYBERARK_USERNAME="your-username@cyberark.cloud.XXXX"
export CYBERARK_PASSWORD="<your-password-here>"
export TF_ACC=1

# Verify environment
cd ~/terraform-provider-cyberarksia
make check-env

# 4. Build and install provider
make build && make install

# 5. Initialize Terraform
cd $TEST_DIR
terraform init
```

**Validation**:
- [ ] Working directory created with timestamp
- [ ] Template copied successfully
- [ ] Environment variables set
- [ ] Provider built without errors
- [ ] Terraform initialized successfully

#### Phase 2: CREATE - VM Secret (< 1 minute)

**IMPORTANT**: Target sets require an existing VM secret. Create one first or reference an existing secret.

```bash
# Option A: Create test secret (recommended for isolated testing)
terraform apply -target=cyberarksia_virtual_machine_secret.test -auto-approve

# Option B: Use existing secret by updating secret_id in template
# Edit crud-test-target-set.tf and set secret_id to existing secret ID
```

**Validation**:
- [ ] VM secret created successfully
- [ ] Secret ID is UUID format
- [ ] secret_type is "ProvisionerUser" or "PCloudAccount"

#### Phase 3: CREATE - Target Set (< 1 minute)

**Edit Template**: Update `local.test_suffix` for unique naming (e.g., "01", "02", "03")

```hcl
locals {
  test_suffix = "01"  # Change this for each test run
}
```

**Create Target Set**:
```bash
terraform apply -auto-approve
```

**Validation**:
- [ ] Target set created successfully
- [ ] Name format: `crud-test-{suffix}.example.com`
- [ ] Type: "Domain"
- [ ] ID equals name (name-as-ID pattern)
- [ ] secret_id matches VM secret
- [ ] provision_format: `<user>-test-<session-guid>` (custom value)
- [ ] enable_certificate_validation: true (default)
- [ ] description populated correctly

#### Phase 4: READ - State Refresh (< 1 minute)

```bash
# Refresh state from API
terraform refresh

# Verify no changes detected
terraform plan
```

**Expected Output**: `No changes. Your infrastructure matches the configuration.`

**Validation**:
- [ ] `terraform plan` shows 0 to add, 0 to change, 0 to destroy
- [ ] All computed fields populated (id, provision_format defaults)
- [ ] No drift detected between state and API

#### Phase 5: UPDATE - Type Change (< 1 minute)

**Test Type Mutability**: Change from Domain → Suffix (in-place update, no recreation)

Edit `crud-test-target-set.tf`:
```hcl
resource "cyberarksia_target_set" "test" {
  name        = "crud-test-${local.test_suffix}.example.com"
  type        = "Suffix"  # CHANGED from "Domain"
  secret_id   = cyberarksia_virtual_machine_secret.test.id
  secret_type = "ProvisionerUser"

  description                   = "UPDATED: CRUD validation test - modified"  # CHANGED
  provision_format              = "jit-<user>-<session-guid>"                  # CHANGED
  enable_certificate_validation = false                                        # CHANGED
}
```

Apply changes:
```bash
terraform apply -auto-approve
```

**Expected Output**: `Plan: 0 to add, 1 to change, 0 to destroy`

**Validation**:
- [ ] Type changed to "Suffix" (no resource recreation)
- [ ] Description updated
- [ ] provision_format changed successfully
- [ ] enable_certificate_validation toggled to false
- [ ] ID still equals name (unchanged)
- [ ] `terraform plan` shows no further changes

#### Phase 6: UPDATE - Rename Test (< 1 minute)

**Test In-Place Rename**: Verify ID follows name

Edit `crud-test-target-set.tf`:
```hcl
resource "cyberarksia_target_set" "test" {
  name        = "crud-test-${local.test_suffix}-renamed.example.com"  # CHANGED
  type        = "Suffix"
  secret_id   = cyberarksia_virtual_machine_secret.test.id
  secret_type = "ProvisionerUser"

  description                   = "UPDATED: CRUD validation test - modified"
  provision_format              = "jit-<user>-<session-guid>"
  enable_certificate_validation = false
}
```

Apply rename:
```bash
terraform apply -auto-approve
```

**Expected Output**: `Plan: 0 to add, 1 to change, 0 to destroy`

**Validation**:
- [ ] `terraform plan` shows update in-place (NOT replacement)
- [ ] After apply, ID equals new name
- [ ] Old name would return 404 if queried manually
- [ ] `terraform plan` shows no changes after rename

#### Phase 7: NEGATIVE TEST - provision_format Clearing (< 1 minute)

**Test Clearing Prevention**: Attempt to remove provision_format (should fail at plan time)

Edit `crud-test-target-set.tf` and **remove** the `provision_format` line:
```hcl
resource "cyberarksia_target_set" "test" {
  name        = "crud-test-${local.test_suffix}-renamed.example.com"
  type        = "Suffix"
  secret_id   = cyberarksia_virtual_machine_secret.test.id
  secret_type = "ProvisionerUser"

  description                   = "UPDATED: CRUD validation test - modified"
  # provision_format = ""  # REMOVED - should ERROR
  enable_certificate_validation = false
}
```

Try to plan:
```bash
terraform plan
```

**Expected Output**: Error at plan time

**Validation**:
- [ ] `terraform plan` shows error: "Cannot Clear Attribute"
- [ ] Error message explains: "cannot be removed once set due to API limitations"
- [ ] Plan does NOT proceed (blocked at plan phase)

**Restore provision_format** before continuing:
```hcl
  provision_format = "jit-<user>-<session-guid>"
```

#### Phase 8: NEGATIVE TEST - Forward Slash Warning (< 1 minute)

**Test Forward Slash Detection**: Create target set with forward slashes (warning, not error)

**WARNING**: This test creates a resource that CANNOT be deleted via Terraform. Manual SIA UI deletion required.

Create separate test configuration:
```hcl
resource "cyberarksia_target_set" "forward_slash_test" {
  name        = "env/test/servers-${local.test_suffix}"  # Contains forward slashes
  type        = "Domain"
  secret_id   = cyberarksia_virtual_machine_secret.test.id
  secret_type = "ProvisionerUser"
}
```

Apply:
```bash
terraform apply -target=cyberarksia_target_set.forward_slash_test -auto-approve
```

**Validation**:
- [ ] `terraform plan` shows WARNING (not error)
- [ ] Warning mentions "forward slashes which will cause deletion failures"
- [ ] Resource CAN be created (warning doesn't block)

**Cleanup Attempt** (will fail):
```bash
terraform destroy -target=cyberarksia_target_set.forward_slash_test -auto-approve
```

**Expected Output**: Error 403 Forbidden

**Validation**:
- [ ] `terraform destroy` FAILS with 403 Forbidden
- [ ] Manual deletion via SIA UI required
- [ ] Forward slash validator working correctly

**Manual Cleanup**:
1. Log into SIA UI
2. Navigate to Target Sets
3. Manually delete `env/test/servers-{suffix}`

#### Phase 9: IMPORT - Test Import Functionality (< 1 minute)

**Test Import by Name**: Target sets use name as ID

```bash
# Get target set name (ID equals name)
TARGET_SET_ID=$(terraform output -raw target_set_id)

# Remove from state
terraform state rm cyberarksia_target_set.test

# Import by name
terraform import cyberarksia_target_set.test "$TARGET_SET_ID"

# Verify no changes after import
terraform plan
```

**Expected Output**: `No changes. Your infrastructure matches the configuration.`

**Validation**:
- [ ] Import succeeded with name as ID
- [ ] All attributes populated correctly after import
- [ ] name matches ID
- [ ] type, secret_id, provision_format restored
- [ ] No changes detected in `terraform plan`

#### Phase 10: DELETE - Cleanup (< 1 minute)

**Delete in Dependency Order**:

```bash
# Delete target set first
terraform destroy -target=cyberarksia_target_set.test -auto-approve

# Delete VM secret
terraform destroy -target=cyberarksia_virtual_machine_secret.test -auto-approve
```

**Validation**:
- [ ] Target set deleted successfully
- [ ] VM secret deleted successfully
- [ ] No orphaned resources in SIA UI
- [ ] State is clean: `terraform state list` returns empty

### Success Criteria

- ✅ VM secret created successfully (ProvisionerUser or PCloudAccount)
- ✅ Target set created with all attributes
- ✅ ID equals name (name-as-ID pattern)
- ✅ Type changes (Domain → Suffix → Target) without recreation
- ✅ provision_format can be updated but not cleared (plan-time error)
- ✅ Forward slash validator warns but allows creation
- ✅ In-place rename with ID following name
- ✅ Certificate validation can be toggled
- ✅ Credential rotation (update secret_id) works in-place
- ✅ IMPORT works with name as ID
- ✅ DELETE cleans up without errors
- ✅ SIA UI reflects all changes correctly

### Advanced Testing: All 6 Type Change Combinations

**Test All Bidirectional Changes** (Domain ↔ Suffix ↔ Target):

```bash
# Start with Domain
terraform apply -auto-approve
# Verify: type = "Domain"

# Change to Suffix
# Edit: type = "Suffix"
terraform apply -auto-approve
# Verify: type = "Suffix", no recreation

# Change to Target
# Edit: type = "Target"
terraform apply -auto-approve
# Verify: type = "Target", no recreation

# Change back to Domain
# Edit: type = "Domain"
terraform apply -auto-approve
# Verify: type = "Domain", no recreation

# Test direct Domain → Target
# Edit: type = "Target"
terraform apply -auto-approve
# Verify: type = "Target", no recreation

# Test direct Target → Suffix (completes all 6 combinations)
# Edit: type = "Suffix"
terraform apply -auto-approve
# Verify: type = "Suffix", no recreation
```

**Validation**:
- [ ] All 6 type change combinations work (Target↔Domain, Target↔Suffix, Domain↔Suffix)
- [ ] No ForceNew on type attribute
- [ ] ID remains stable across type changes
- [ ] `terraform plan` shows no changes after each apply

### Common Issues

**Issue**: Forward Slash Deletion Failure
```
Error: 403 Forbidden - Cannot delete target set with forward slashes in name
```
**Solution**: Manually delete via SIA UI. Forward slash validator warns about this during creation.

**Issue**: provision_format Clearing Error
```
Error: Cannot Clear Attribute: provision_format cannot be removed once set
```
**Solution**: Expected behavior. You can UPDATE provision_format but not remove it. This is due to API PATCH semantics.

**Issue**: Secret Not Found
```
Error: Failed to read VM secret: 404 Not Found
```
**Solution**: Ensure VM secret exists before creating target set. Create via template or reference existing secret ID.

### Troubleshooting Reference

**Quick Diagnostics**:
1. **Validate environment**: `make check-env` - verifies `CYBERARK_USERNAME` and `CYBERARK_PASSWORD`
2. **Rebuild provider**: `cd ~/terraform-provider-cyberarksia && make build && make install`
3. **Reinitialize Terraform**: `rm -rf .terraform .terraform.lock.hcl && terraform init`
4. **Check template**: Ensure `local.test_suffix` is updated to avoid name conflicts

**Common Patterns**:
- **Name conflicts**: Increment `local.test_suffix` for each test run
- **Drift detection**: Run `terraform refresh` then `terraform plan`
- **State issues**: Use `terraform state list` and `terraform state show` to inspect

### Test Results Documentation

Save test results:

```bash
cat > TARGET-SET-TEST-RESULTS-$(date +%Y%m%d-%H%M%S).md <<'EOF'
# Target Set CRUD Test Results

**Test Date**: $(date)
**Test Directory**: $TEST_DIR
**Duration**: [FILL IN] minutes

## Resources Created
- VM Secret ID: [SECRET_ID]
- Target Set Name: [TARGET_SET_NAME]
- Type Tested: Domain → Suffix → Target (all combinations)

## Test Phases
- [x] Phase 1: Setup
- [x] Phase 2: VM Secret Creation
- [x] Phase 3: Target Set Creation
- [x] Phase 4: READ - State Refresh
- [x] Phase 5: UPDATE - Type Change
- [x] Phase 6: UPDATE - Rename
- [x] Phase 7: NEGATIVE - provision_format Clearing
- [x] Phase 8: NEGATIVE - Forward Slash Warning
- [x] Phase 9: IMPORT
- [x] Phase 10: DELETE

## Validation Results
- All resources created successfully: YES/NO
- No drift detected: YES/NO
- Type changes without recreation: YES/NO
- provision_format clearing prevented: YES/NO
- Forward slash warning displayed: YES/NO
- Rename successful with ID update: YES/NO
- IMPORT successful: YES/NO
- DELETE successful: YES/NO
- SIA UI matches state: YES/NO

## Issues Encountered
[FILL IN]

## Notes
[FILL IN]
EOF
```

---

## See Also

### Documentation
- [`CLAUDE.md`](../../CLAUDE.md) - Development guidelines (references this guide)
- [`docs/testing-framework.md`](../../docs/testing-framework.md) - Conceptual testing framework
- [`docs/resources/database_policy_workspace_assignment.md`](../../docs/resources/database_policy_workspace_assignment.md) - Policy database assignment documentation
- [`docs/resources/database_policy.md`](../../docs/resources/database_policy.md) - Database policy documentation
- [`docs/resources/database_policy_principal_assignment.md`](../../docs/resources/database_policy_principal_assignment.md) - Principal assignment documentation
- [`examples/resources/`](../resources/) - Per-resource usage examples

### Automation
- [`scripts/test-crud-resource.sh`](../../scripts/test-crud-resource.sh) - Automated CRUD testing script
- [`Makefile`](../../Makefile) - Build and test targets (`make help` for all commands)
  - `make test-crud DESC=<description>` - Run automated CRUD validation
  - `make check-env` - Verify environment variables
  - `make build && make install` - Build and install provider
  - `make testacc` - Run acceptance tests

### Test Results
- `/tmp/sia-azure-test-20251027-185657/TEST-RESULTS.md` - Azure PostgreSQL test results (2025-10-27)
- `/tmp/sia-crud-validation-*/` - Automated test directories (timestamped)
