# Password Rotation Example for Virtual Machine Secret
#
# This example demonstrates how to securely rotate passwords for ProvisionerUser
# secrets using Terraform variables and best practices.
#
# SECURITY BEST PRACTICES:
# 1. NEVER hardcode passwords in configuration files
# 2. ALWAYS use sensitive variables for password management
# 3. ALWAYS add terraform.tfvars to .gitignore
# 4. Consider using external secret management tools (Vault, AWS Secrets Manager)
#
# ROTATION WORKFLOW:
# Step 1: Set initial password via environment variable or terraform.tfvars
# Step 2: Apply configuration to create secret
# Step 3: Update password value in variable source
# Step 4: Apply again - Terraform performs in-place password update
#
# Expected behavior on password rotation:
# - Terraform plan shows: ~ provisioner_password = (sensitive value)
# - No resource recreation (in-place update only)
# - Password never appears in plan output or state file in plaintext

# ==============================================================================
# VARIABLES (Define these in variables.tf or inline for this example)
# ==============================================================================

variable "vm_admin_password" {
  description = "Current password for VM administrator account (rotate by changing this value)"
  type        = string
  sensitive   = true

  validation {
    condition     = length(var.vm_admin_password) >= 12
    error_message = "Password must be at least 12 characters long."
  }
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, staging, prod."
  }
}

# ==============================================================================
# VM SECRET RESOURCE (ProvisionerUser with Password Rotation)
# ==============================================================================

resource "cyberarksia_vm_secret" "rotated_admin" {
  # Secret name remains constant across password rotations
  secret_name = "${var.environment}-vm-admin"
  secret_type = "ProvisionerUser"

  # Username typically remains constant
  provisioner_username = "admin"

  # Password is sourced from variable - rotate by updating the variable
  provisioner_password = var.vm_admin_password
}

# ==============================================================================
# OUTPUTS (Safe to expose - password is not included)
# ==============================================================================

output "rotated_secret_id" {
  description = "UUID of the VM secret (use for workspace references)"
  value       = cyberarksia_vm_secret.rotated_admin.secret_id
}

output "rotated_secret_name" {
  description = "Name of the VM secret"
  value       = cyberarksia_vm_secret.rotated_admin.secret_name
}

output "rotation_instructions" {
  description = "Instructions for rotating the password"
  value       = <<-EOT
    To rotate the password:

    Method 1 - Environment Variable:
      export TF_VAR_vm_admin_password="NewSecurePassword456!"
      terraform apply

    Method 2 - terraform.tfvars (remember to add to .gitignore):
      echo 'vm_admin_password = "NewSecurePassword456!"' >> terraform.tfvars
      terraform apply

    Method 3 - Interactive prompt:
      terraform apply
      # Terraform will prompt for vm_admin_password

    Expected plan output:
      ~ provisioner_password = (sensitive value)

    The password is updated in-place without recreating the resource.
  EOT
}

# ==============================================================================
# ROTATION WORKFLOW EXAMPLE
# ==============================================================================

# INITIAL SETUP:
# --------------
# 1. Create terraform.tfvars (ensure it's in .gitignore):
#    cat > terraform.tfvars <<EOF
#    vm_admin_password = "InitialPassword123!"
#    environment = "dev"
#    EOF
#
# 2. Initialize and apply:
#    terraform init
#    terraform apply
#
# 3. Verify creation:
#    terraform state show cyberarksia_vm_secret.rotated_admin
#
# Expected output:
#   secret_id = "abc-123-def-456"
#   secret_name = "dev-vm-admin"
#   provisioner_username = "admin"
#   provisioner_password = <sensitive>

# PASSWORD ROTATION (Method 1 - Environment Variable):
# -----------------------------------------------------
# 1. Set new password:
#    export TF_VAR_vm_admin_password="RotatedPassword456!"
#
# 2. Preview changes:
#    terraform plan
#
# Expected plan output:
#   Terraform will perform the following actions:
#
#   # cyberarksia_vm_secret.rotated_admin will be updated in-place
#   ~ resource "cyberarksia_vm_secret" "rotated_admin" {
#       ~ provisioner_password = (sensitive value)
#       # (other attributes unchanged)
#     }
#
# 3. Apply rotation:
#    terraform apply
#
# 4. Verify no other changes:
#    terraform plan
#    # Should show: No changes. Your infrastructure matches the configuration.

# PASSWORD ROTATION (Method 2 - terraform.tfvars):
# -------------------------------------------------
# 1. Update terraform.tfvars:
#    sed -i 's/InitialPassword123!/RotatedPassword789!/' terraform.tfvars
#
# 2. Apply rotation:
#    terraform apply
#
# Expected behavior: Same as Method 1

# PASSWORD ROTATION (Method 3 - External Secret Manager):
# --------------------------------------------------------
# If using Vault, AWS Secrets Manager, or similar:
#
# data "aws_secretsmanager_secret_version" "vm_password" {
#   secret_id = "vm-admin-password"
# }
#
# resource "cyberarksia_vm_secret" "rotated_admin" {
#   secret_name = "vm-admin"
#   secret_type = "ProvisionerUser"
#
#   provisioner_username = "admin"
#   provisioner_password = data.aws_secretsmanager_secret_version.vm_password.secret_string
# }
#
# Rotation workflow:
#   1. Update password in AWS Secrets Manager
#   2. Run: terraform apply
#   3. Terraform detects change and updates SIA secret

# VERIFICATION AFTER ROTATION:
# -----------------------------
# 1. Check Terraform state (password still shows as sensitive):
#    terraform state show cyberarksia_vm_secret.rotated_admin
#
# 2. Verify in SIA UI:
#    - Navigate to SIA → Secrets → VM Secrets
#    - Confirm secret exists with same secret_id
#    - Password is not visible in UI (stored securely)
#
# 3. Test with dependent resources (if applicable):
#    - VM workspace using this secret should continue working
#    - No connection interruption during rotation

# COMMON PITFALLS:
# ----------------
# ❌ DON'T hardcode passwords:
#    provisioner_password = "HardcodedPassword123!"  # NEVER DO THIS
#
# ❌ DON'T commit terraform.tfvars with passwords:
#    # Always add to .gitignore:
#    echo "terraform.tfvars" >> .gitignore
#
# ❌ DON'T use command-line variables (visible in shell history):
#    terraform apply -var='vm_admin_password=Secret123!'  # Avoid this
#
# ✅ DO use environment variables (TF_VAR_*):
#    export TF_VAR_vm_admin_password="SecurePassword123!"
#
# ✅ DO use external secret managers:
#    data "vault_generic_secret" "password" { ... }
#
# ✅ DO validate password complexity:
#    Use validation blocks in variable definitions
#
# ✅ DO mark variables as sensitive:
#    variable "password" { sensitive = true }
