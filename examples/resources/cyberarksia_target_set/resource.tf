# Example 1: Domain-Based Target Set (Production Servers)
#
# Matches all servers in a domain (most common use case)

resource "cyberarksia_vm_secret" "admin" {
  secret_name          = "windows-admin-credentials"
  secret_type          = "ProvisionerUser"
  provisioner_username = "Administrator"
  provisioner_password = var.admin_password
}

resource "cyberarksia_target_set" "production" {
  name        = "prod.example.com"
  type        = "Domain"
  secret_id   = cyberarksia_vm_secret.admin.id
  secret_type = cyberarksia_vm_secret.admin.secret_type

  description = "Production environment servers"
}

# Example 2: Suffix-Based Target Set (Datacenter Grouping)
#
# Matches servers with a specific hostname suffix

resource "cyberarksia_target_set" "datacenter_east" {
  name        = "dc-east.example.com"
  type        = "Suffix"
  secret_id   = cyberarksia_vm_secret.admin.id
  secret_type = "ProvisionerUser"

  description                   = "East Coast Datacenter Servers"
  enable_certificate_validation = true
}

# Example 3: Target-Based Target Set (Individual System)
#
# Matches a specific server (exact hostname)

resource "cyberarksia_target_set" "critical_database" {
  name        = "db01.prod.example.com"
  type        = "Target"
  secret_id   = cyberarksia_vm_secret.admin.id
  secret_type = "ProvisionerUser"

  description      = "Critical Production Database Server"
  provision_format = "jit-<user>-<session-guid>"
}
