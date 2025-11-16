# Azure Cloud VM Access Policy Example
# Demonstrates Azure-specific target criteria

data "cyberarksia_principal" "azure_admin" {
  name = "azure-admin@example.com"
  type = "USER"
}

resource "cyberarksia_vm_policy" "azure_production" {
  name          = "Azure-Production-VMs"
  location_type = "Azure"
  status        = "Active"
  description   = "Access policy for Azure production virtual machines"

  principal {
    principal_id          = data.cyberarksia_principal.azure_admin.principal_id
    principal_name        = data.cyberarksia_principal.azure_admin.principal_name
    principal_type        = data.cyberarksia_principal.azure_admin.principal_type
    source_directory_name = data.cyberarksia_principal.azure_admin.source_directory_name
    source_directory_id   = data.cyberarksia_principal.azure_admin.source_directory_id
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

    # Resource groups
    resource_groups = [
      "rg-production-east",
      "rg-production-west"
    ]

    # Virtual networks
    vnet_ids = [
      "/subscriptions/sub-123/resourceGroups/rg-prod/providers/Microsoft.Network/virtualNetworks/vnet-prod-east",
      "/subscriptions/sub-123/resourceGroups/rg-prod/providers/Microsoft.Network/virtualNetworks/vnet-prod-west"
    ]

    # Azure subscriptions
    subscriptions = ["subscription-id-123", "subscription-id-456"]
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
