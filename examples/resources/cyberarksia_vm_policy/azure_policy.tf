# Azure Cloud VM Access Policy Example
# Demonstrates Azure-specific target criteria
#
# Azure Target Format Requirements:
#   subscriptions:   UUID format (e.g., "759a039e-dc44-4762-9f40-2696323c2fa5")
#   resource_groups: Full ARM path (e.g., "/subscriptions/<id>/resourceGroups/<name>")
#   vnet_ids:        Full ARM path (e.g., "/subscriptions/<id>/resourceGroups/<rg>/providers/Microsoft.Network/virtualNetworks/<name>")

data "cyberarksia_principal" "azure_admin" {
  name = "azure-admin@example.com"
  type = "USER"
}

resource "cyberarksia_vm_policy" "azure_production" {
  name          = "Azure-Production-VMs"
  location_type = "Azure"
  status        = "Active"
  description   = "Access policy for Azure production virtual machines"

  principals {
    principal_id          = data.cyberarksia_principal.azure_admin.id
    principal_name        = data.cyberarksia_principal.azure_admin.name
    principal_type        = data.cyberarksia_principal.azure_admin.principal_type
    source_directory_name = data.cyberarksia_principal.azure_admin.directory_name
    source_directory_id   = data.cyberarksia_principal.azure_admin.directory_id
  }

  # SSH for Linux VMs
  behavior {
    ssh {
      username = "azureuser"
    }
  }

  # Azure target criteria
  azure_targets {
    regions = ["eastus", "westus2", "northeurope"]

    # Azure resource tags
    tags {
      key   = "Environment"
      value = ["production"]
    }

    tags {
      key   = "CostCenter"
      value = ["Engineering", "Operations"]
    }

    # Resource groups (MUST use full ARM path format)
    resource_groups = [
      "/subscriptions/759a039e-dc44-4762-9f40-2696323c2fa5/resourceGroups/rg-production-east",
      "/subscriptions/759a039e-dc44-4762-9f40-2696323c2fa5/resourceGroups/rg-production-west"
    ]

    # Virtual networks
    vnet_ids = [
      "/subscriptions/sub-123/resourceGroups/rg-prod/providers/Microsoft.Network/virtualNetworks/vnet-prod-east",
      "/subscriptions/sub-123/resourceGroups/rg-prod/providers/Microsoft.Network/virtualNetworks/vnet-prod-west"
    ]

    # Azure subscriptions (UUID format)
    subscriptions = ["759a039e-dc44-4762-9f40-2696323c2fa5", "a1b2c3d4-5678-90ab-cdef-1234567890ab"]
  }

  max_session_duration = 4
  idle_time            = 20

  # Business hours
  access_window {
    days_of_the_week = [1, 2, 3, 4, 5]
    from_hour        = "09:00"
    to_hour          = "17:00"
  }

  time_zone = "UTC"

  tags = ["azure", "production", "linux"]
}

output "azure_policy_id" {
  value       = cyberarksia_vm_policy.azure_production.policy_id
  description = "Azure VM policy ID"
}
