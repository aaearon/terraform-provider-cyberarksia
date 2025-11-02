# Quickstart: Virtual Machine Secret Management

**Feature**: Virtual Machine Secret Management
**Branch**: `003-virtual-machine-secret`
**Date**: 2025-11-02
**Resource**: `cyberarksia_virtual_machine_secret`

## Overview

This quickstart guide provides example configurations and testing workflows for the `cyberarksia_virtual_machine_secret` Terraform resource. Use these examples as templates for implementing and testing VM secret management.

## Basic Examples

### Example 1: ProvisionerUser Secret (Self-Contained Credentials)

Create a VM secret with username and password stored in SIA:

```hcl
resource "cyberarksia_virtual_machine_secret" "app_server" {
  secret_name = "app-server-admin"
  secret_type = "ProvisionerUser"

  provisioner_username = "admin"
  provisioner_password = "SecurePassword123!"
}

output "secret_id" {
  description = "UUID of the created VM secret"
  value       = cyberarksia_virtual_machine_secret.app_server.secret_id
}

output "secret_name" {
  description = "Name of the VM secret"
  value       = cyberarksia_virtual_machine_secret.app_server.secret_name
}
```

**Usage**:
```bash
terraform apply
# Outputs:
# secret_id = "abc-123-def-456"
# secret_name = "app-server-admin"
```

### Example 2: PCloudAccount Secret (PAM Vault Reference)

Create a VM secret that references an existing PAM vault account:

```hcl
resource "cyberarksia_virtual_machine_secret" "vault_ref" {
  secret_name = "production-db-admin"
  secret_type = "PCloudAccount"

  pcloud_safe_name    = "Production-Safe"
  pcloud_account_name = "db-admin-account"
}

output "vault_reference" {
  value = {
    secret_id = cyberarksia_virtual_machine_secret.vault_ref.secret_id
    safe      = cyberarksia_virtual_machine_secret.vault_ref.pcloud_safe_name
    account   = cyberarksia_virtual_machine_secret.vault_ref.pcloud_account_name
  }
}
```

**Prerequisites**:
- PAM safe "Production-Safe" must exist
- Account "db-admin-account" must exist in that safe
- Service account must have read permission on the safe

### Example 3: Multiple Secrets (Mixed Types)

```hcl
# Self-contained credentials for dev environment
resource "cyberarksia_virtual_machine_secret" "dev_server" {
  secret_name = "dev-server-admin"
  secret_type = "ProvisionerUser"

  provisioner_username = "devadmin"
  provisioner_password = var.dev_admin_password  # From variables
}

# PAM vault reference for production
resource "cyberarksia_virtual_machine_secret" "prod_server" {
  secret_name = "prod-server-admin"
  secret_type = "PCloudAccount"

  pcloud_safe_name    = var.prod_safe_name
  pcloud_account_name = var.prod_account_name
}

# Locals for reference
locals {
  dev_secret_id  = cyberarksia_virtual_machine_secret.dev_server.secret_id
  prod_secret_id = cyberarksia_virtual_machine_secret.prod_server.secret_id
}
```

**variables.tf**:
```hcl
variable "dev_admin_password" {
  description = "Password for dev server admin"
  type        = string
  sensitive   = true
}

variable "prod_safe_name" {
  description = "PAM safe name for production"
  type        = string
  default     = "Production-Safe"
}

variable "prod_account_name" {
  description = "PAM account name for production"
  type        = string
  default     = "prod-admin-account"
}
```

## Advanced Examples

### Example 4: Secret with Name Updates

Demonstrate in-place name updates:

```hcl
resource "cyberarksia_virtual_machine_secret" "app_server" {
  secret_name = "old-server-name"  # Can be changed without recreating
  secret_type = "ProvisionerUser"

  provisioner_username = "admin"
  provisioner_password = "SecurePassword123!"
}
```

**Update workflow**:
```bash
# Initial creation
terraform apply

# Change name in config
# secret_name = "new-server-name"

# Apply update (in-place, no destroy/recreate)
terraform apply
# Plan will show:
# ~ secret_name = "old-server-name" -> "new-server-name"
```

### Example 5: Password Rotation

Demonstrate password updates:

```hcl
resource "cyberarksia_virtual_machine_secret" "rotated_secret" {
  secret_name = "app-server-admin"
  secret_type = "ProvisionerUser"

  provisioner_username = "admin"
  provisioner_password = var.current_password  # Rotate via variable
}
```

**Rotation workflow**:
```bash
# Set initial password
export TF_VAR_current_password="OldPassword123!"
terraform apply

# Rotate password
export TF_VAR_current_password="NewPassword456!"
terraform apply
# Updates password in-place, marked sensitive in plan output
```

### Example 6: ForceNew on Type Change

**WARNING**: Changing `secret_type` triggers destroy + recreate:

```hcl
resource "cyberarksia_virtual_machine_secret" "changing_type" {
  secret_name = "server-admin"
  secret_type = "ProvisionerUser"  # Changing this triggers ForceNew

  provisioner_username = "admin"
  provisioner_password = "Password123!"
}
```

**What happens**:
```bash
terraform apply  # Creates ProvisionerUser secret

# Change config to PCloudAccount
# secret_type = "PCloudAccount"
# pcloud_safe_name = "Safe"
# pcloud_account_name = "Account"

terraform plan
# Plan will show:
# -/+ (destroy and create replacement)
# Reason: secret_type is immutable (ForceNew)
```

## Import Existing Secrets

### Example 7: Import ProvisionerUser Secret

Import an existing VM secret created outside Terraform:

```bash
# Step 1: Define resource in config (without password initially)
cat > import.tf <<'EOF'
resource "cyberarksia_virtual_machine_secret" "existing" {
  secret_name = "placeholder"  # Will be updated from API
  secret_type = "ProvisionerUser"

  provisioner_username = "placeholder"
  provisioner_password = "temporary"  # Must be provided in config
}
EOF

# Step 2: Run import command
terraform import cyberarksia_virtual_machine_secret.existing abc-123-def-456

# Step 3: Retrieve actual values
terraform state show cyberarksia_virtual_machine_secret.existing
# Output shows:
# secret_id = "abc-123-def-456"
# secret_name = "actual-name-from-sia"
# provisioner_username = "actual-username"
# provisioner_password = <sensitive>  # NOT retrieved from API

# Step 4: Update config with actual values
# Edit import.tf to match actual secret_name and provisioner_username
# Update provisioner_password to match actual password

# Step 5: Verify no drift
terraform plan
# Should show: No changes. Your infrastructure matches the configuration.
```

**Important**: Passwords are NOT retrieved during import. You must know the actual password and add it to your configuration manually.

### Example 8: Import PCloudAccount Secret

```bash
# Step 1: Create resource definition
resource "cyberarksia_virtual_machine_secret" "vault_import" {
  secret_name = "placeholder"
  secret_type = "PCloudAccount"

  pcloud_safe_name    = "placeholder"
  pcloud_account_name = "placeholder"
}

# Step 2: Import
terraform import cyberarksia_virtual_machine_secret.vault_import xyz-789-ghi

# Step 3: Update config with actual values from state
terraform state show cyberarksia_virtual_machine_secret.vault_import
```

## Testing Workflows

### CRUD Validation (per TESTING-GUIDE.md)

**Phase 1: Create**
```bash
# 1. Apply configuration
terraform apply

# 2. Verify in SIA UI
# - Navigate to SIA → Secrets → VM Secrets
# - Confirm secret exists with correct name
# - Verify secret_id matches Terraform output

# 3. Verify Terraform state
terraform state show cyberarksia_virtual_machine_secret.app_server
# Confirm all attributes present (password shows as <sensitive>)
```

**Phase 2: Read (Drift Detection)**
```bash
# 1. No changes scenario
terraform plan
# Expected: No changes. Your infrastructure matches the configuration.

# 2. Manual change scenario (drift detection)
# - In SIA UI: Change secret name from "app-server-admin" to "modified-name"
# - Run: terraform plan
# Expected: Plan shows drift, proposes update to restore original name

# 3. Apply correction
terraform apply
# Restores original name
```

**Phase 3: Update**
```bash
# 1. Update mutable field (secret_name)
# Edit config: secret_name = "updated-server-name"
terraform plan
# Expected: ~ secret_name = "app-server-admin" -> "updated-server-name"

terraform apply
# Verify in SIA UI: Name updated

# 2. Update password (sensitive)
# Edit config: provisioner_password = "NewPassword789!"
terraform plan
# Expected: ~ provisioner_password = (sensitive value) (forces update)

terraform apply
# Password rotated (not visible in output)
```

**Phase 4: Delete**
```bash
# 1. Remove resource from config or run destroy
terraform destroy

# 2. Verify in SIA UI
# - Secret no longer exists
# - Confirm deletion successful

# 3. Verify Terraform state
terraform state list
# Resource should not be listed
```

### Validation Checklist

From `examples/testing/TESTING-GUIDE.md`:

- [ ] **CREATE**: Secret created with correct attributes
- [ ] **CREATE**: secret_id returned and stored in state
- [ ] **CREATE**: Password marked as sensitive in plan output
- [ ] **CREATE**: Secret visible in SIA UI

- [ ] **READ**: terraform plan shows no changes after creation
- [ ] **READ**: Manual name change detected as drift
- [ ] **READ**: Password never appears in plan output

- [ ] **UPDATE**: secret_name update performed in-place
- [ ] **UPDATE**: provisioner_password update performed in-place
- [ ] **UPDATE**: pcloud_* fields update performed in-place
- [ ] **UPDATE**: secret_type change triggers ForceNew

- [ ] **DELETE**: Secret removed from SIA
- [ ] **DELETE**: Resource removed from Terraform state
- [ ] **DELETE**: Idempotent (no error if already deleted)

- [ ] **IMPORT**: Existing secret imported by secret_id
- [ ] **IMPORT**: Metadata populated correctly in state
- [ ] **IMPORT**: terraform plan shows no changes after import (excluding password)

## Configuration Patterns

### Pattern 1: Sensitive Variable Management

Store passwords securely:

```hcl
# variables.tf
variable "vm_admin_password" {
  description = "VM administrator password"
  type        = string
  sensitive   = true
}

# secrets.tf
resource "cyberarksia_virtual_machine_secret" "admin" {
  secret_name = "vm-admin"
  secret_type = "ProvisionerUser"

  provisioner_username = "admin"
  provisioner_password = var.vm_admin_password
}
```

**Usage**:
```bash
# Option 1: Environment variable
export TF_VAR_vm_admin_password="SecurePass123!"
terraform apply

# Option 2: terraform.tfvars (add to .gitignore!)
echo 'vm_admin_password = "SecurePass123!"' > terraform.tfvars
terraform apply

# Option 3: Command-line (not recommended - visible in shell history)
terraform apply -var='vm_admin_password=SecurePass123!'
```

### Pattern 2: Multiple Environments

Manage secrets per environment:

```hcl
# locals.tf
locals {
  environment = terraform.workspace  # or var.environment

  secrets = {
    dev = {
      name     = "dev-server-admin"
      username = "devadmin"
      password = var.dev_password
    }
    prod = {
      name     = "prod-server-admin"
      username = "prodadmin"
      password = var.prod_password
    }
  }
}

# secrets.tf
resource "cyberarksia_virtual_machine_secret" "server_admin" {
  secret_name = local.secrets[local.environment].name
  secret_type = "ProvisionerUser"

  provisioner_username = local.secrets[local.environment].username
  provisioner_password = local.secrets[local.environment].password
}
```

### Pattern 3: Conditional Secret Type

Create different secret types based on input:

```hcl
variable "use_pam_vault" {
  description = "Use PAM vault reference instead of standalone credentials"
  type        = bool
  default     = false
}

resource "cyberarksia_virtual_machine_secret" "conditional" {
  secret_name = "server-admin"
  secret_type = var.use_pam_vault ? "PCloudAccount" : "ProvisionerUser"

  # ProvisionerUser fields (used when use_pam_vault = false)
  provisioner_username = var.use_pam_vault ? null : var.admin_username
  provisioner_password = var.use_pam_vault ? null : var.admin_password

  # PCloudAccount fields (used when use_pam_vault = true)
  pcloud_safe_name    = var.use_pam_vault ? var.safe_name : null
  pcloud_account_name = var.use_pam_vault ? var.account_name : null
}
```

## Troubleshooting

### Issue: Validation Error on Create

**Error**:
```
Error: Invalid configuration
provisioner_username is required when secret_type is "ProvisionerUser"
```

**Solution**: Ensure all required conditional fields are provided based on secret_type.

### Issue: ForceNew on Minor Change

**Error**: Terraform proposes destroy + recreate when you only changed name.

**Solution**: Verify you didn't accidentally change `secret_type` (immutable field).

### Issue: Password Drift Detected

**Error**: Terraform always shows password as changed.

**Solution**: This is not normal - passwords are write-only and should never trigger drift. Check for configuration issues.

### Issue: Import Shows Password Missing

**Behavior**: After import, password is not in state.

**Expected**: This is correct behavior. API never returns passwords. You must add the actual password to your configuration manually.

### Issue: PAM Safe/Account Not Found

**Error**:
```
Error: API returned error
Safe "Production-Safe" not found or you don't have access
```

**Solution**:
1. Verify safe exists in PAM
2. Verify service account has read permission on safe
3. Check account name spelling

## Next Steps

1. **Review Data Model**: See [data-model.md](./data-model.md) for detailed attribute definitions
2. **Review SDK Contracts**: See [contracts/sdk-methods.md](./contracts/sdk-methods.md) for API details
3. **Run CRUD Tests**: Follow validation checklist above
4. **Implement Acceptance Tests**: Create `virtual_machine_secret_resource_test.go`
5. **Generate Documentation**: Run `tfplugindocs generate` after implementation

---

**Document Version**: 1.0
**Last Updated**: 2025-11-02
**Status**: Ready for implementation
