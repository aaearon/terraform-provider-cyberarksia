# VM Policy Resource Testing

**Part of**: [Comprehensive CRUD Testing Guide](../TESTING-GUIDE.md)
**Last Updated**: See git history

---

## VM Policy Resource Testing

### Overview

VM access policies define **WHO** can access (principals), **WHAT** they access (targets), **WHEN** they can access (conditions), and **HOW** they connect (behavior). This section covers CRUD testing for `cyberarksia_vm_policy` and `cyberarksia_vm_policy_principal_assignment` resources.

**Template**: [`crud-test-vm-policy.tf`](../crud-test-vm-policy.tf)

**Test Scope**: Full VM policy lifecycle with all location types (FQDN/IP, AWS, Azure, GCP)
**Duration**: 15-20 minutes per location type
**Prerequisites**: Valid CyberArk SIA credentials, at least one principal for assignment

### Resources Validated

1. `cyberarksia_vm_policy` - VM access policy with targets, behavior, and conditions
2. `cyberarksia_vm_policy_principal_assignment` - Dynamic principal assignments to existing policies

### Key Features Tested

**Location Types** (oneOf - exactly one per policy):
- **FQDN/IP**: On-premises servers with hostname/IP matching
- **AWS**: Cloud VMs with region, VPC, account, tag filtering
- **Azure**: Cloud VMs with resource group, VNET, subscription, tag filtering
- **GCP**: Cloud VMs with region, project, VPC, label filtering

**Connection Behavior**:
- **SSH**: Username-based SSH access
- **RDP Local Ephemeral**: Windows RDP with local group membership
- **RDP Domain Ephemeral**: Windows RDP with domain group membership
- **SSH + RDP**: Combined access profiles

**Critical Behaviors**:
- Minimum 1 principal required at policy creation
- ForceNew on `name` and `location_type` attributes
- Principal preservation during updates (inline + assigned principals)
- Azure policies use HTTP workaround (ARK SDK v1.5.0 bug)
- SetType for `fqdn_rule` and `assign_groups` (no ordering drift)

---

## Testing Workflow (20-30 minutes)

### Phase 1: Setup (2 minutes)

```bash
# 1. Create timestamped working directory
export TEST_DIR="/tmp/sia-crud-validation-vm-policy-$(date +%Y%m%d-%H%M%S)"
mkdir -p $TEST_DIR
cd $TEST_DIR

# 2. Copy VM policy template
cp ~/terraform-provider-cyberarksia/examples/testing/crud-test-vm-policy.tf .
cp ~/terraform-provider-cyberarksia/examples/testing/crud-test-provider.tf .

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
- [ ] Templates copied successfully
- [ ] Environment variables set
- [ ] Provider built without errors
- [ ] Terraform initialized successfully

### Phase 2: CREATE - Basic FQDN/IP Policy with SSH (2-3 minutes)

**Edit Template**: Update principal data source with actual test user/group

```hcl
data "cyberarksia_principal" "test_user" {
  name = "your-test-user@example.com"  # REPLACE WITH ACTUAL USER
  type = "USER"
}
```

**Create Policy**:
```bash
terraform apply -auto-approve
```

**Expected Output**: Policy created with unique timestamped name

**Validation**:
- [ ] Policy created successfully (no errors)
- [ ] `policy_id` populated (UUID format)
- [ ] `status` = "Active"
- [ ] `location_type` = "FQDN/IP"
- [ ] At least 1 principal assigned
- [ ] `max_session_duration` = 2 (from template)
- [ ] `delegation_classification` computed
- [ ] `created_by` metadata populated
- [ ] FQDN target rules present

### Phase 3: READ - State Refresh (1 minute)

```bash
# Refresh state from API
terraform refresh

# Verify no changes detected
terraform plan
```

**Expected Output**: `No changes. Your infrastructure matches the configuration.`

**Validation**:
- [ ] `terraform plan` shows 0 to add, 0 to change, 0 to destroy
- [ ] All computed fields populated
- [ ] No drift detected between state and API
- [ ] FQDN rules order-independent (SetType, no reordering drift)

### Phase 4: PRINCIPAL ASSIGNMENT - Add Group (2 minutes)

**Edit Template**: Add group assignment after policy creation

```hcl
data "cyberarksia_principal" "test_group" {
  name = "Test-Group"  # REPLACE WITH ACTUAL GROUP
  type = "GROUP"
}

resource "cyberarksia_vm_policy_principal_assignment" "test_group" {
  policy_id             = cyberarksia_vm_policy.crud_test.policy_id
  principal_id          = data.cyberarksia_principal.test_group.principal_id
  principal_name        = data.cyberarksia_principal.test_group.principal_name
  principal_type        = data.cyberarksia_principal.test_group.principal_type
  source_directory_name = data.cyberarksia_principal.test_group.source_directory_name
  source_directory_id   = data.cyberarksia_principal.test_group.source_directory_id
}
```

**Apply Assignment**:
```bash
terraform apply -auto-approve
```

**Validation**:
- [ ] Assignment resource created successfully
- [ ] Composite ID format: `{policy_id}:{principal_id}:{principal_type}`
- [ ] Policy now has 2 principals (inline + assigned)
- [ ] No duplicate principal error

### Phase 5: UPDATE - Session Duration and Tags (2 minutes)

**Edit Template**: Modify updateable fields

```hcl
resource "cyberarksia_vm_policy" "crud_test" {
  # ... keep other fields same ...
  max_session_duration = 4              # CHANGED from 2
  tags = ["crud-test", "terraform", "updated"]  # ADDED "updated"
}
```

**Apply Update**:
```bash
terraform apply -auto-approve
```

**Expected Output**: `Plan: 0 to add, 1 to change, 0 to destroy`

**Validation**:
- [ ] `max_session_duration` changed from 2 to 4
- [ ] Tags updated with "updated" tag
- [ ] All other fields preserved
- [ ] Principal count unchanged (both inline + assigned preserved)
- [ ] `updated_by` metadata updated
- [ ] `terraform plan` shows no further changes

### Phase 6: UPDATE - Add FQDN Rule (2 minutes)

**Test SetType Behavior**: Add second FQDN rule (no ordering drift)

```hcl
fqdn_ip_targets {
  fqdn_rule {
    operator             = "SUFFIX"
    computername_pattern = "-test"
    domain               = "example.com"
  }
  fqdn_rule {
    operator             = "PREFIX"        # NEW RULE
    computername_pattern = "web-"
    domain               = "example.com"
  }
}
```

**Apply Update**:
```bash
terraform apply -auto-approve
```

**Validation**:
- [ ] Second FQDN rule added successfully
- [ ] No resource recreation (in-place update)
- [ ] Both rules present in state
- [ ] Rule ordering doesn't cause drift (SetType)
- [ ] `terraform plan` shows no changes after apply

### Phase 7: NEGATIVE TEST - ForceNew on Name Change (1 minute)

**Test ForceNew Behavior**: Changing name forces replacement

```hcl
resource "cyberarksia_vm_policy" "crud_test" {
  name = "RENAMED-Policy-${formatdate("YYYYMMDD-hhmm", timestamp())}"  # CHANGED
  # ... rest unchanged ...
}
```

**Plan Only** (don't apply):
```bash
terraform plan
```

**Expected Output**: `Plan: 1 to add, 0 to change, 1 to destroy` (ForceNew)

**Validation**:
- [ ] Plan shows "must be replaced" for name change
- [ ] No apply (revert name change for cleanup)

**Revert**: Change name back to original before continuing

### Phase 8: IMPORT - Test Import Functionality (2 minutes)

```bash
# Get policy ID
POLICY_ID=$(terraform output -raw validation_summary | jq -r '.policy_id')

# Remove from state
terraform state rm cyberarksia_vm_policy.crud_test

# Import by policy_id
terraform import cyberarksia_vm_policy.crud_test "$POLICY_ID"

# Verify no changes after import
terraform plan
```

**Expected Output**: `No changes. Your infrastructure matches the configuration.`

**Validation**:
- [ ] Import succeeded with policy_id
- [ ] All attributes populated correctly after import
- [ ] Principals restored (inline configuration matches)
- [ ] Targets, behavior, conditions all restored
- [ ] `terraform plan` shows no changes

### Phase 9: DELETE - Cleanup (2 minutes)

**Delete in Dependency Order**:

```bash
# Delete assignment first (depends on policy)
terraform destroy -target=cyberarksia_vm_policy_principal_assignment.test_group -auto-approve

# Delete policy
terraform destroy -target=cyberarksia_vm_policy.crud_test -auto-approve
```

**Validation**:
- [ ] Principal assignment deleted successfully
- [ ] VM policy deleted successfully
- [ ] No orphaned resources in SIA UI
- [ ] State is clean: `terraform state list` returns empty

---

## Location Type-Specific Testing

### AWS Policy Testing

**Template Section**: Modify location_type and targets

```hcl
resource "cyberarksia_vm_policy" "aws_test" {
  name          = "AWS-Test-Policy-${formatdate("YYYYMMDD-hhmm", timestamp())}"
  location_type = "AWS"
  status        = "Active"

  principal {
    # ... principal config ...
  }

  behavior {
    ssh {
      username = "ec2-user"
    }
  }

  aws_targets {
    regions     = ["us-east-1", "us-west-2"]
    account_ids = ["123456789012"]
    vpc_ids     = ["vpc-12345678"]

    tags {
      key   = "Environment"
      value = "production"
    }
  }

  max_session_duration = 2
}
```

**Validation**:
- [ ] AWS policy created with cloud targets
- [ ] Regions array preserved correctly
- [ ] Tags with key/value structure stored
- [ ] VPC IDs and account IDs optional but work when provided

### Azure Policy Testing (HTTP Workaround)

**Important**: Azure policies use direct HTTP workaround due to ARK SDK v1.5.0 serialization bug.

```hcl
resource "cyberarksia_vm_policy" "azure_test" {
  name          = "Azure-Test-Policy-${formatdate("YYYYMMDD-hhmm", timestamp())}"
  location_type = "Azure"
  status        = "Active"

  principal {
    # ... principal config ...
  }

  behavior {
    ssh {
      username = "azureuser"
    }
  }

  azure_targets {
    regions         = ["eastus", "westus2"]
    resource_groups = ["production-rg"]
    subscriptions   = ["12345678-1234-1234-1234-123456789012"]
    vnet_ids        = ["vnet-12345678"]

    tags {
      key   = "Team"
      value = "Platform"
    }
  }

  max_session_duration = 2
}
```

**Validation**:
- [ ] Azure policy created successfully (workaround working)
- [ ] `location_type` normalized to "Azure" in state (not "AZURE")
- [ ] Resource groups and subscriptions preserved
- [ ] Tags with key/value structure stored
- [ ] CRUD operations all working despite SDK bug

### GCP Policy Testing

```hcl
resource "cyberarksia_vm_policy" "gcp_test" {
  name          = "GCP-Test-Policy-${formatdate("YYYYMMDD-hhmm", timestamp())}"
  location_type = "GCP"
  status        = "Active"

  principal {
    # ... principal config ...
  }

  behavior {
    ssh {
      username = "gcp-user"
    }
  }

  gcp_targets {
    regions  = ["us-central1", "us-east1"]
    projects = ["my-project-id"]
    vpc_ids  = ["default"]

    labels {
      key   = "environment"  # GCP uses labels, not tags
      value = "production"
    }
  }

  max_session_duration = 2
}
```

**Validation**:
- [ ] GCP policy created successfully
- [ ] Labels (not tags) preserved correctly
- [ ] Projects and regions stored
- [ ] SDK path works without workaround

---

## RDP Behavior Testing

### RDP Local Ephemeral User

```hcl
behavior {
  rdp {
    local_ephemeral_user {
      assign_groups                  = ["Administrators", "Remote Desktop Users"]
      enable_ephemeral_user_reconnect = true
    }
  }
}
```

**Validation**:
- [ ] RDP-only policy creates successfully (no SSH required)
- [ ] assign_groups stored as Set (no ordering drift)
- [ ] enable_ephemeral_user_reconnect preserved
- [ ] Default false when not specified

### RDP Domain Ephemeral User

```hcl
behavior {
  rdp {
    domain_ephemeral_user {
      assign_groups                  = ["Local Admins"]
      assign_domain_groups           = ["Domain Admins", "Server Operators"]
      enable_ephemeral_user_reconnect = false
    }
  }
}
```

**Validation**:
- [ ] Domain ephemeral user config stored
- [ ] Both local and domain groups preserved
- [ ] SetType prevents ordering drift

### Combined SSH + RDP

```hcl
behavior {
  ssh {
    username = "admin"
  }
  rdp {
    local_ephemeral_user {
      assign_groups = ["Administrators"]
    }
  }
}
```

**Validation**:
- [ ] Both SSH and RDP profiles stored
- [ ] Either can be used independently
- [ ] No validation error for having both

---

## Common Issues

### Issue: Azure Policy Creation Fails

```
Error: HTTP 500 - Internal Server Error
```

**Cause**: ARK SDK v1.5.0 serialization bug (uses "AZURE" instead of "Azure")

**Solution**: The provider automatically uses HTTP workaround for Azure policies. Ensure you're using the latest provider build (`make build && make install`).

### Issue: Duplicate Principal Error

```
Error: Duplicate principal assignment
```

**Cause**: Trying to assign a principal that's already in the policy's principals list

**Solution**: Check if principal is already assigned (inline or via assignment resource). Remove duplicate.

### Issue: FQDN Rule Ordering Drift

```
Plan: 0 to add, 1 to change, 0 to destroy
```

**Cause**: Old provider version using ListType for fqdn_rule

**Solution**: Update to latest provider. fqdn_rule now uses SetType (no ordering sensitivity).

### Issue: ForceNew on location_type

```
must be replaced (location_type changed)
```

**Cause**: location_type is immutable after creation

**Solution**: Expected behavior. Create new policy with different location type if needed.

### Issue: Principal Preservation Failure

```
Error: Assigned principals were removed during update
```

**Cause**: Update logic not preserving assigned principals

**Solution**: Bug in provider. Check Update() method uses Read-Modify-Write pattern with principal preservation.

---

## Success Criteria

- [ ] FQDN/IP policy created with all target rules
- [ ] AWS policy created with cloud targets
- [ ] Azure policy created (workaround functional)
- [ ] GCP policy created with labels
- [ ] SSH behavior working
- [ ] RDP local ephemeral working
- [ ] RDP domain ephemeral working
- [ ] Combined SSH + RDP working
- [ ] Principal assignment resource working
- [ ] Duplicate detection working
- [ ] ForceNew on name/location_type confirmed
- [ ] Update preserves assigned principals
- [ ] SetType prevents ordering drift (fqdn_rule, assign_groups)
- [ ] Import by policy_id working
- [ ] Delete cleans up all resources
- [ ] SIA UI reflects all changes correctly

---

## Test Results Documentation

Save test results:

```bash
cat > VM-POLICY-TEST-RESULTS-$(date +%Y%m%d-%H%M%S).md <<'EOF'
# VM Policy CRUD Test Results

**Test Date**: $(date)
**Test Directory**: $TEST_DIR
**Duration**: [FILL IN] minutes

## Resources Created
- Policy ID: [POLICY_ID]
- Policy Name: [POLICY_NAME]
- Location Type: [FQDN/IP | AWS | Azure | GCP]

## Test Phases
- [x] Phase 1: Setup
- [x] Phase 2: CREATE - Basic Policy
- [x] Phase 3: READ - State Refresh
- [x] Phase 4: PRINCIPAL ASSIGNMENT
- [x] Phase 5: UPDATE - Session Duration
- [x] Phase 6: UPDATE - Add FQDN Rule
- [x] Phase 7: NEGATIVE - ForceNew
- [x] Phase 8: IMPORT
- [x] Phase 9: DELETE

## Validation Results
- Policy created successfully: YES/NO
- No drift detected: YES/NO
- Updates preserve principals: YES/NO
- SetType prevents ordering drift: YES/NO
- Azure workaround functional: YES/NO
- IMPORT successful: YES/NO
- DELETE successful: YES/NO

## Issues Encountered
[FILL IN]

## Notes
[FILL IN]
EOF
```

---

## See Also

### Documentation
- [`CLAUDE.md`](../../../CLAUDE.md) - Development guidelines
- [`specs/001-vm-access-policies/`](../../../specs/001-vm-access-policies/) - Feature specification
- [`docs/development/vm-policy-implementation.md`](../../../docs/development/vm-policy-implementation.md) - Implementation summary

### Examples
- [`examples/resources/cyberarksia_vm_policy/`](../../resources/cyberarksia_vm_policy/) - Per-resource examples
- [`examples/resources/cyberarksia_vm_policy_principal_assignment/`](../../resources/cyberarksia_vm_policy_principal_assignment/) - Assignment examples

### Automation
- [`Makefile`](../../../Makefile) - Build and test targets (`make testacc` for acceptance tests)
