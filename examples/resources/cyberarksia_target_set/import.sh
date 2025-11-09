#!/bin/bash
# Import an existing target set into Terraform state

# Target sets are imported by name (the unique identifier)
# The ID is automatically computed to match the name

# Example: Import a target set named "prod.example.com"
terraform import cyberarksia_target_set.production "prod.example.com"

# After import, create a matching configuration:
# resource "cyberarksia_target_set" "production" {
#   name        = "prod.example.com"
#   type        = "Domain"
#   secret_id   = "aec8cf4b-8012-4efb-9aa2-ca14db5f79c0"
#   secret_type = "ProvisionerUser"
# }

# Verify import was successful:
# terraform plan  # Should show "No changes"

# Note: You'll need to know the secret_id and other attributes
# to create an accurate configuration after import
