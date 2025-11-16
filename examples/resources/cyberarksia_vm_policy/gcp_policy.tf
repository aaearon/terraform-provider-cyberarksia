# GCP Cloud VM Access Policy Example
# Demonstrates GCP-specific target criteria (note: uses "labels" not "tags")

data "cyberarksia_principal" "gcp_admin" {
  name = "gcp-admin@example.com"
  type = "USER"
}

resource "cyberarksia_vm_policy" "gcp_production" {
  name          = "GCP-Production-VMs"
  location_type = "GCP"
  status        = "Active"
  description   = "Access policy for GCP production compute instances"

  principal {
    principal_id          = data.cyberarksia_principal.gcp_admin.principal_id
    principal_name        = data.cyberarksia_principal.gcp_admin.principal_name
    principal_type        = data.cyberarksia_principal.gcp_admin.principal_type
    source_directory_name = data.cyberarksia_principal.gcp_admin.source_directory_name
    source_directory_id   = data.cyberarksia_principal.gcp_admin.source_directory_id
  }

  # SSH for GCE instances
  behavior {
    ssh {
      username = "gcpuser"
    }
  }

  # GCP target criteria (NOTE: uses "labels" not "tags")
  gcp_targets {
    regions = ["us-central1", "us-east1", "europe-west1"]

    # GCP labels (NOT tags)
    labels {
      key   = "environment"
      value = ["production"]
    }

    labels {
      key   = "team"
      value = ["platform", "sre"]
    }

    # VPC networks
    vpc_ids = [
      "projects/my-project/global/networks/vpc-prod-us",
      "projects/my-project/global/networks/vpc-prod-eu"
    ]

    # GCP projects
    projects = ["my-project-prod-123", "my-project-shared-456"]
  }

  max_session_duration = 3
  idle_time            = 10

  # 24/7 access for SRE team
  time_zone = "UTC"

  tags = ["gcp", "production", "sre"]
}

output "gcp_policy_id" {
  value       = cyberarksia_vm_policy.gcp_production.policy_id
  description = "GCP VM policy ID"
}
