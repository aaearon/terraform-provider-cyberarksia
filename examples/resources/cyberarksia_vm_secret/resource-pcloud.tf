resource "cyberarksia_vm_secret" "pcloud_example" {
  secret_name = "production-db-admin"
  secret_type = "PCloudAccount"

  pcloud_safe_name    = "Production-Safe"
  pcloud_account_name = "db-admin-account"
}

output "vault_reference" {
  value = {
    secret_id = cyberarksia_vm_secret.pcloud_example.secret_id
    safe      = cyberarksia_vm_secret.pcloud_example.pcloud_safe_name
    account   = cyberarksia_vm_secret.pcloud_example.pcloud_account_name
  }
}
