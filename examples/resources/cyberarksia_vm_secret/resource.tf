resource "cyberarksia_vm_secret" "example" {
  secret_name = "app-server-admin"
  secret_type = "ProvisionerUser"

  provisioner_username = "admin"
  provisioner_password = "SecurePassword123!"
}

output "secret_id" {
  description = "UUID of the created VM secret"
  value       = cyberarksia_vm_secret.example.secret_id
}

output "secret_name" {
  description = "Name of the VM secret"
  value       = cyberarksia_vm_secret.example.secret_name
}
