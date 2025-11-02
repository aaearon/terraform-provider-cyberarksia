# ============================================================================
# Azure Infrastructure Outputs
# ============================================================================

output "azure_resource_group" {
  description = "Azure resource group name"
  value       = azurerm_resource_group.sia_test.name
}

output "azure_postgres_server_name" {
  description = "Azure PostgreSQL server name"
  value       = azurerm_postgresql_flexible_server.sia_test.name
}

output "azure_postgres_fqdn" {
  description = "Azure PostgreSQL server FQDN"
  value       = azurerm_postgresql_flexible_server.sia_test.fqdn
}

output "azure_postgres_admin_username" {
  description = "PostgreSQL administrator username"
  value       = var.postgres_admin_username
}

# ============================================================================
# SIA Resource Outputs
# ============================================================================

output "secret_id" {
  description = "SIA secret ID for PostgreSQL admin credentials"
  value       = cyberarksia_database_secret.admin.id
}

output "database_workspace_id" {
  description = "SIA database workspace ID"
  value       = cyberarksia_database_workspace.azure_postgres.id
}

output "database_workspace_name" {
  description = "SIA database workspace name"
  value       = cyberarksia_database_workspace.azure_postgres.name
}

output "policy_id" {
  description = "SIA database policy ID (UUID)"
  value       = cyberarksia_database_policy.test.policy_id
}

output "policy_name" {
  description = "SIA database policy name"
  value       = cyberarksia_database_policy.test.name
}

output "tim_assignment_id" {
  description = "Tim Schindler's principal assignment composite ID"
  value       = cyberarksia_database_policy_principal_assignment.tim_schindler.id
}

# ============================================================================
# Validation Summary
# ============================================================================

output "validation_summary" {
  description = "Summary of test resources for validation"
  sensitive   = true
  value = {
    azure_infrastructure = {
      resource_group  = azurerm_resource_group.sia_test.name
      postgres_server = azurerm_postgresql_flexible_server.sia_test.name
      postgres_fqdn   = azurerm_postgresql_flexible_server.sia_test.fqdn
      test_database   = azurerm_postgresql_flexible_server_database.testdb.name
      firewall_rules  = ["AllowAzureServices", "AllowAll"]
      region          = var.azure_region
    }
    sia_resources = {
      secret_id                      = cyberarksia_database_secret.admin.id
      database_workspace_id          = cyberarksia_database_workspace.azure_postgres.id
      database_name                  = cyberarksia_database_workspace.azure_postgres.name
      certificate_validation_enabled = cyberarksia_database_workspace.azure_postgres.enable_certificate_validation
      policy_id                      = cyberarksia_database_policy.test.policy_id
      policy_name                    = cyberarksia_database_policy.test.name
      policy_status                  = cyberarksia_database_policy.test.status
    }
    principal_assignments = {
      service_account = {
        type  = "USER (inline)"
        email = var.sia_username
      }
      tim_schindler = {
        type          = "USER (separate resource)"
        email         = var.test_principal_email
        assignment_id = cyberarksia_database_policy_principal_assignment.tim_schindler.id
      }
    }
    policy_conditions = {
      max_session_duration = cyberarksia_database_policy.test.conditions.max_session_duration
      idle_time            = cyberarksia_database_policy.test.conditions.idle_time
      access_window = {
        days = cyberarksia_database_policy.test.conditions.access_window.days_of_the_week
        from = cyberarksia_database_policy.test.conditions.access_window.from_hour
        to   = cyberarksia_database_policy.test.conditions.access_window.to_hour
      }
    }
  }
}

output "next_steps" {
  description = "Next steps for manual verification"
  sensitive   = true
  value       = <<-EOT

  ✅ TERRAFORM APPLY COMPLETE!

  Next Steps for Manual Verification:

  1. SIA UI Verification:
     - Navigate to SIA UI and find policy: "${cyberarksia_database_policy.test.name}"
     - Check "Assigned To" section:
       * Should show service account: ${var.sia_username}
       * Should show Tim Schindler: ${var.test_principal_email}
     - Check "Targets" section:
       * Should show Azure PostgreSQL database: ${cyberarksia_database_workspace.azure_postgres.name}
       * FQDN: ${azurerm_postgresql_flexible_server.sia_test.fqdn}
       * Authentication: db_auth with roles ["pg_read_all_settings"]

  2. Test Database Access:
     - Log into SIA portal as Tim Schindler
     - Request access to policy: "${cyberarksia_database_policy.test.name}"
     - Verify access is granted during: Monday-Friday, 09:00-17:00 GMT
     - Verify session limits: 4 hours max, 10 min idle timeout

  3. Azure Verification:
     - Check resource group: ${azurerm_resource_group.sia_test.name}
     - Verify PostgreSQL server: ${azurerm_postgresql_flexible_server.sia_test.name}
     - Check FQDN resolves: ${azurerm_postgresql_flexible_server.sia_test.fqdn}

  4. Cost Verification:
     - Check Azure costs (should be < $0.01 for test duration)
     - B1ms SKU: ~$0.017/hour

  ⚠️  WHEN READY TO CLEAN UP:
  - Run: terraform destroy
  - Verify all resources removed from both SIA UI and Azure Portal

  Policy ID: ${cyberarksia_database_policy.test.policy_id}
  Database Workspace ID: ${cyberarksia_database_workspace.azure_postgres.id}
  EOT
}
