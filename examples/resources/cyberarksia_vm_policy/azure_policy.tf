# Azure Cloud VM Access Policy Example
# Demonstrates Azure-specific target criteria
#
# IMPORTANT: Azure target format requirements:
# - subscriptions: UUID format (e.g., "759a039e-dc44-4762-9f40-2696323c2fa5")
# - resource_groups: Full ARM path ("/subscriptions/<id>/resourceGroups/<name>")
# - vnet_ids: Full ARM path ("/subscriptions/<id>/resourceGroups/<rg>/providers/Microsoft.Network/virtualNetworks/<name>")

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

    # Resource groups - MUST use full ARM path format
    resource_groups = [
      "/subscriptions/00000000-0000-0000-0000-000000000001/resourceGroups/rg-production-east",
      "/subscriptions/00000000-0000-0000-0000-000000000001/resourceGroups/rg-production-west"
    ]

    # Virtual networks - full ARM path format
    vnet_ids = [
      "/subscriptions/00000000-0000-0000-0000-000000000001/resourceGroups/rg-prod/providers/Microsoft.Network/virtualNetworks/vnet-prod-east",
      "/subscriptions/00000000-0000-0000-0000-000000000001/resourceGroups/rg-prod/providers/Microsoft.Network/virtualNetworks/vnet-prod-west"
    ]

    # Azure subscriptions - UUID format (32 chars in 5 groups with hyphens)
    subscriptions = ["00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"]
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
