# ==============================================================================
# CyberArk SIA CRUD Testing Template - Outputs
# ==============================================================================
# This is a CANONICAL TEMPLATE - do not modify directly.
#
# For complete testing workflow, see:
#   examples/testing/TESTING-GUIDE.md
#
# Usage:
#   1. Copy this file to /tmp/sia-crud-validation/
#   2. Follow TESTING-GUIDE.md for complete instructions
# ==============================================================================

# ==============================================================================
# CRUD Validation Outputs
# ==============================================================================

# ----------------------------------------------------------------------------
# Policy Data Source Outputs
# ----------------------------------------------------------------------------
output "policy_id" {
  description = "Looked up policy ID"
  value       = data.cyberarksia_database_policy.test_policy.id
}

output "policy_name" {
  description = "Looked up policy name"
  value       = data.cyberarksia_database_policy.test_policy.name
}

output "policy_status" {
  description = "Looked up policy status"
  value       = data.cyberarksia_database_policy.test_policy.status
}

# ----------------------------------------------------------------------------
# Certificate Outputs
# ----------------------------------------------------------------------------
output "certificate_id" {
  description = "Created certificate ID"
  value       = cyberarksia_certificate.test_cert.id
}

output "certificate_name" {
  description = "Certificate name"
  value       = cyberarksia_certificate.test_cert.cert_name
}

output "certificate_expiration" {
  description = "Certificate expiration date (ISO 8601)"
  value       = cyberarksia_certificate.test_cert.expiration_date
}

output "certificate_metadata_issuer" {
  description = "Certificate issuer from metadata"
  value       = cyberarksia_certificate.test_cert.metadata.issuer
}

output "certificate_metadata_subject" {
  description = "Certificate subject from metadata"
  value       = cyberarksia_certificate.test_cert.metadata.subject
}

# ----------------------------------------------------------------------------
# Secret Outputs
# ----------------------------------------------------------------------------
output "secret_id" {
  description = "Created secret ID (UUID format)"
  value       = cyberarksia_database_secret.test_secret.id
}

output "secret_name" {
  description = "Secret name"
  value       = cyberarksia_database_secret.test_secret.name
}

output "secret_created_at" {
  description = "Secret creation timestamp"
  value       = cyberarksia_database_secret.test_secret.created_at
}

output "secret_authentication_type" {
  description = "Secret authentication type"
  value       = cyberarksia_database_secret.test_secret.authentication_type
}

# ----------------------------------------------------------------------------
# Database Workspace Outputs
# ----------------------------------------------------------------------------
output "database_workspace_id" {
  description = "Created database workspace ID"
  value       = cyberarksia_database_workspace.test_db.id
}

output "database_workspace_name" {
  description = "Database workspace name"
  value       = cyberarksia_database_workspace.test_db.name
}

output "database_workspace_type" {
  description = "Database type (engine)"
  value       = cyberarksia_database_workspace.test_db.database_type
}

output "database_workspace_secret_id" {
  description = "Referenced secret ID"
  value       = cyberarksia_database_workspace.test_db.secret_id
}

output "database_workspace_certificate_id" {
  description = "Referenced certificate ID"
  value       = cyberarksia_database_workspace.test_db.certificate_id
}

# ----------------------------------------------------------------------------
# Policy Database Assignment Outputs
# ----------------------------------------------------------------------------
output "policy_assignment_id" {
  description = "Policy database assignment composite ID (format: policy_id:database_id)"
  value       = cyberarksia_database_policy_workspace_assignment.test_assignment.id
}

output "policy_assignment_policy_id" {
  description = "Assigned policy ID"
  value       = cyberarksia_database_policy_workspace_assignment.test_assignment.policy_id
}

output "policy_assignment_database_id" {
  description = "Assigned database workspace ID"
  value       = cyberarksia_database_policy_workspace_assignment.test_assignment.database_workspace_id
}

output "policy_assignment_auth_method" {
  description = "Authentication method used"
  value       = cyberarksia_database_policy_workspace_assignment.test_assignment.authentication_method
}

output "policy_assignment_last_modified" {
  description = "Last modification timestamp"
  value       = cyberarksia_database_policy_workspace_assignment.test_assignment.last_modified
}

# ----------------------------------------------------------------------------
# Validation Summary
# ----------------------------------------------------------------------------
output "validation_summary" {
  description = "Summary of created resources for validation"
  value = {
    # Policy lookup
    policy_found = data.cyberarksia_database_policy.test_policy.id != ""

    # Resource creation
    certificate_created = cyberarksia_certificate.test_cert.id != ""
    secret_created      = cyberarksia_database_secret.test_secret.id != ""
    database_created    = cyberarksia_database_workspace.test_db.id != ""
    assignment_created  = cyberarksia_database_policy_workspace_assignment.test_assignment.id != ""

    # Dependencies working
    database_has_secret      = cyberarksia_database_workspace.test_db.secret_id == cyberarksia_database_secret.test_secret.id
    database_has_certificate = cyberarksia_database_workspace.test_db.certificate_id == cyberarksia_certificate.test_cert.id
    assignment_has_database  = cyberarksia_database_policy_workspace_assignment.test_assignment.database_workspace_id == cyberarksia_database_workspace.test_db.id
    assignment_has_policy    = cyberarksia_database_policy_workspace_assignment.test_assignment.policy_id == data.cyberarksia_database_policy.test_policy.id

    # Resource count
    total_resources_created = 4
  }
}

output "test_completion_message" {
  description = "Test completion status"
  value       = "✅ All 4 resources created successfully! Review validation_summary for dependency verification."
}
