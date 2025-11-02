# Outputs for Azure + SIA integration test

# Azure PostgreSQL outputs
output "azure_resource_group" {
  description = "Azure resource group name"
  value       = azurerm_resource_group.sia_test.name
}

output "postgres_server_name" {
  description = "PostgreSQL server name"
  value       = azurerm_postgresql_flexible_server.sia_test.name
}

output "postgres_server_fqdn" {
  description = "PostgreSQL server FQDN (for connection testing)"
  value       = azurerm_postgresql_flexible_server.sia_test.fqdn
}

output "postgres_server_id" {
  description = "PostgreSQL server Azure resource ID"
  value       = azurerm_postgresql_flexible_server.sia_test.id
}

output "postgres_version" {
  description = "PostgreSQL version"
  value       = azurerm_postgresql_flexible_server.sia_test.version
}

# SIA resource outputs
output "sia_secret_id" {
  description = "CyberArk SIA secret ID"
  value       = cyberarksia_database_secret.postgres_admin.id
}

output "sia_secret_name" {
  description = "CyberArk SIA secret name"
  value       = cyberarksia_database_secret.postgres_admin.name
}

output "sia_database_workspace_id" {
  description = "CyberArk SIA database workspace ID"
  value       = cyberarksia_database_workspace.azure_postgres.id
}

output "sia_database_workspace_name" {
  description = "CyberArk SIA database workspace name"
  value       = cyberarksia_database_workspace.azure_postgres.name
}

output "sia_certificate_id" {
  description = "CyberArk SIA certificate ID (Microsoft RSA Root CA 2017)"
  value       = cyberarksia_certificate.azure_postgres_cert.id
}

output "sia_certificate_name" {
  description = "CyberArk SIA certificate name"
  value       = cyberarksia_certificate.azure_postgres_cert.cert_name
}

# Connection testing commands
output "test_commands" {
  description = "Commands to test connectivity"
  value = {
    psql_direct = "psql 'host=${azurerm_postgresql_flexible_server.sia_test.fqdn} port=5432 dbname=testdb user=${var.postgres_admin_username} password=${var.postgres_admin_password} sslmode=require'"

    dns_lookup = "nslookup ${azurerm_postgresql_flexible_server.sia_test.fqdn}"

    port_check = "nc -zv ${azurerm_postgresql_flexible_server.sia_test.fqdn} 5432"
  }
  sensitive = true
}

# Cost estimation
output "estimated_cost" {
  description = "Estimated cost for this infrastructure"
  value = {
    hourly_rate   = "$0.017 USD (B1ms compute)"
    daily_rate    = "$0.41 USD"
    monthly_rate  = "$12.41 USD (if running continuously)"
    storage_cost  = "$0.138/GB/month (32GB = $4.42/month)"
    total_monthly = "~$16.83 USD if running 24/7"
    note          = "Stop server when not in use to eliminate compute costs (only storage remains)"
  }
}

# Validation summary
output "validation_summary" {
  description = "Summary of created resources for validation"
  value = {
    azure_resources_created = {
      resource_group  = azurerm_resource_group.sia_test.name
      postgres_server = azurerm_postgresql_flexible_server.sia_test.name
      database        = azurerm_postgresql_flexible_server_database.testdb.name
      firewall_rules  = ["AllowAzureServices", "AllowAll_TestOnly"]
    }
    sia_resources_created = {
      certificate        = cyberarksia_certificate.azure_postgres_cert.cert_name
      secret             = cyberarksia_database_secret.postgres_admin.name
      database_workspace = cyberarksia_database_workspace.azure_postgres.name
    }
    test_status = {
      azure_provisioned     = "✅ PostgreSQL B1ms server created"
      sia_certificate_added = "✅ Microsoft RSA Root CA 2017 certificate"
      sia_secret_created    = "✅ Secret with admin credentials"
      sia_workspace_created = "✅ Database workspace with TLS/SSL certificate"
      cloud_provider        = cyberarksia_database_workspace.azure_postgres.cloud_provider
      region                = cyberarksia_database_workspace.azure_postgres.region
    }
  }
}
