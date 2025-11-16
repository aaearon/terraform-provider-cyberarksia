# Assign additional principal to existing VM policy
resource "cyberarksia_vm_policy_principal_assignment" "example" {
  policy_id             = "policy-uuid-here"
  principal_id          = "user-uuid-here"
  principal_name        = "developer@example.com"
  principal_type        = "USER"
  source_directory_name = "CyberArk"
  source_directory_id   = "directory-uuid-here"
}
