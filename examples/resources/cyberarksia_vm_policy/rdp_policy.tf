# RDP connection behavior example
# Demonstrates Windows server access with ephemeral user configuration

data "cyberarksia_principal" "windows_admin" {
  name = "windows-admin@example.com"
  type = "USER"
}

resource "cyberarksia_vm_policy" "windows_servers" {
  name          = "Windows-Servers-RDP"
  location_type = "FQDN/IP"
  status        = "Active"
  description   = "RDP access to Windows domain servers"

  # Initial principal (at least one required)
  principals {
    principal_id          = data.cyberarksia_principal.windows_admin.id
    principal_name        = data.cyberarksia_principal.windows_admin.name
    principal_type        = data.cyberarksia_principal.windows_admin.principal_type
    source_directory_name = data.cyberarksia_principal.windows_admin.directory_name
    source_directory_id   = data.cyberarksia_principal.windows_admin.directory_id
  }

  # RDP behavior with domain ephemeral user
  behavior {
    rdp {
      domain_ephemeral_user {
        assign_groups                   = ["Remote Desktop Users"] # Local groups
        assign_domain_groups            = ["Domain Admins"]        # Domain groups
        enable_ephemeral_user_reconnect = true
      }
    }
  }

  # Target: Windows servers by FQDN
  fqdn_ip_targets {
    fqdn_rule {
      operator             = "PREFIX"
      computername_pattern = "WIN-"
      domain               = "corp.example.com"
    }
  }

  max_session_duration = 8
  idle_time            = 30

  tags = ["windows", "rdp", "domain-joined"]
}

# Alternative: Local ephemeral user (for workgroup servers)
resource "cyberarksia_vm_policy" "workgroup_servers" {
  name          = "Workgroup-Servers-RDP"
  location_type = "FQDN/IP"
  status        = "Active"
  description   = "RDP access to standalone Windows servers"

  principals {
    principal_id          = data.cyberarksia_principal.windows_admin.id
    principal_name        = data.cyberarksia_principal.windows_admin.name
    principal_type        = data.cyberarksia_principal.windows_admin.principal_type
    source_directory_name = data.cyberarksia_principal.windows_admin.directory_name
    source_directory_id   = data.cyberarksia_principal.windows_admin.directory_id
  }

  # RDP with local ephemeral user
  behavior {
    rdp {
      local_ephemeral_user {
        assign_groups                   = ["Administrators", "Remote Desktop Users"]
        enable_ephemeral_user_reconnect = false
      }
    }
  }

  fqdn_ip_targets {
    ip_rule {
      operator     = "EXACTLY"
      ip_addresses = ["192.168.1.10", "192.168.1.11"]
      logical_name = "Workgroup Servers"
    }
  }

  max_session_duration = 4
  idle_time            = 15

  tags = ["windows", "rdp", "workgroup"]
}

output "domain_policy_id" {
  value = cyberarksia_vm_policy.windows_servers.policy_id
}

output "workgroup_policy_id" {
  value = cyberarksia_vm_policy.workgroup_servers.policy_id
}
