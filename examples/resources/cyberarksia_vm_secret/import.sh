#!/bin/bash
# Import an existing VM secret by its secret_id (UUID)
#
# Usage:
#   ./import.sh <secret_id>
#
# Example:
#   ./import.sh abc-123-def-456
#
# Prerequisites:
# 1. Define the resource in your Terraform configuration:
#
#    resource "cyberarksia_vm_secret" "imported" {
#      secret_name = "placeholder"  # Will be updated from API
#      secret_type = "ProvisionerUser"
#
#      provisioner_username = "placeholder"
#      provisioner_password = "temporary"  # Must be provided in config
#    }
#
# 2. Run this script with the secret_id from SIA
# 3. After import, update the config with actual values from state

if [ -z "$1" ]; then
  echo "Error: secret_id required"
  echo "Usage: $0 <secret_id>"
  exit 1
fi

SECRET_ID="$1"

terraform import cyberarksia_vm_secret.imported "$SECRET_ID"

echo ""
echo "Import complete! Next steps:"
echo "1. Run: terraform state show cyberarksia_vm_secret.imported"
echo "2. Update your config with actual values from state"
echo "3. Run: terraform plan (should show no changes if config matches)"
echo ""
echo "NOTE: Passwords are NOT imported (API doesn't return them)."
echo "      You must know the actual password and add it to your config."
