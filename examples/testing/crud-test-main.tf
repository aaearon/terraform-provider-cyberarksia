# ==============================================================================
# CyberArk SIA CRUD Testing Template
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
# CRUD Validation Test Configuration - Complete Resource Suite
# ==============================================================================
# Tests all four resources: certificate, secret, database_workspace,
# policy_database_assignment
#
# Dependencies:
# - certificate (standalone)
# - secret (standalone)
# - database_workspace (requires certificate + secret)
# - policy_database_assignment (requires database_workspace + existing policy)
# ==============================================================================

# ==============================================================================
# DATA SOURCES
# ==============================================================================

# Lookup existing access policy for policy assignment testing
data "cyberarksia_database_policy" "test_policy" {
  name = "Terraform-Test-Policy"
}

# ==============================================================================
# RESOURCES
# ==============================================================================

# 1. Certificate Resource - Base dependency for TLS/mTLS
resource "cyberarksia_certificate" "test_cert" {
  cert_name        = "crud-test-cert-${formatdate("YYYYMMDDhhmmss", timestamp())}"
  cert_description = "CRUD validation test certificate - Full suite"
  cert_body        = file("${path.module}/test-cert.pem")
  cert_type        = "PEM"

  labels = {
    environment = "test"
    purpose     = "crud-validation"
    suite       = "full"
    created_at  = formatdate("YYYY-MM-DD", timestamp())
  }
}

# 2. Secret Resource - Required for database workspace authentication
resource "cyberarksia_database_secret" "test_secret" {
  name                = "crud-test-secret-${formatdate("YYYYMMDDhhmmss", timestamp())}"
  authentication_type = "local"
  username            = "testuser"
  password            = "TestPassword123!"

  tags = {
    environment = "test"
    purpose     = "crud-validation"
    suite       = "full"
  }
}

# 3. Database Workspace Resource - References certificate and secret
resource "cyberarksia_database_workspace" "test_db" {
  name                          = "crud-test-db-${formatdate("YYYYMMDDhhmmss", timestamp())}"
  database_type                 = "postgres"
  cloud_provider                = "on_premise"
  address                       = "test.postgres.local"
  port                          = 5432
  secret_id                     = cyberarksia_database_secret.test_secret.id
  certificate_id                = cyberarksia_certificate.test_cert.id
  enable_certificate_validation = false

  tags = {
    environment = "test"
    purpose     = "crud-validation"
    suite       = "full"
  }
}

# 4. Policy Database Assignment Resource - Assigns database to access policy
resource "cyberarksia_database_policy_database_assignment" "test_assignment" {
  policy_id             = data.cyberarksia_database_policy.test_policy.id
  database_workspace_id = cyberarksia_database_workspace.test_db.id
  authentication_method = "db_auth"

  db_auth_profile {
    roles = ["crud_test_reader", "crud_test_writer"]
  }

  # Ensure database is created before assignment
  depends_on = [cyberarksia_database_workspace.test_db]
}
