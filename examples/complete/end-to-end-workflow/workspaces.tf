# Create a TLS certificate for MySQL connections
resource "cyberarksia_certificate" "mysql_ca" {
  cert_name        = "mysql-ca-cert"
  cert_description = "MySQL CA certificate for TLS connections"
  cert_body        = file("${path.module}/certs/mysql-ca.pem")

  labels = {
    purpose     = "mysql-tls"
    environment = "production"
  }
}

# Create a PostgreSQL database workspace
resource "cyberarksia_database_workspace" "production_postgres" {
  name                          = "customers"
  database_type                 = "postgres-aws-rds"
  address                       = "prod-db.abc123.us-east-1.rds.amazonaws.com"
  port                          = 5432
  secret_id                     = cyberarksia_database_secret.postgres_admin.id
  cloud_provider                = "aws"
  region                        = "us-east-1"
  network_name                  = "production-vpc"
  enable_certificate_validation = true

  tags = {
    environment = "production"
    application = "customer-management"
    managed_by  = "terraform"
  }
}

# Create a MySQL database workspace with certificate
resource "cyberarksia_database_workspace" "production_mysql" {
  name                          = "orders"
  database_type                 = "mysql-azure-managed"
  address                       = "prod-mysql.mysql.database.azure.com"
  port                          = 3306
  secret_id                     = cyberarksia_database_secret.postgres_admin.id # Reusing secret for example
  certificate_id                = cyberarksia_certificate.mysql_ca.id
  cloud_provider                = "azure"
  region                        = "eastus"
  enable_certificate_validation = true

  tags = {
    environment = "production"
    application = "order-processing"
    managed_by  = "terraform"
  }
}

# AWS RDS with IAM authentication
resource "cyberarksia_database_workspace" "rds_iam_database" {
  name                  = "analytics"
  database_type         = "postgres-aws-rds"
  address               = "analytics-db.abc123.us-west-2.rds.amazonaws.com"
  port                  = 5432
  secret_id             = cyberarksia_database_secret.rds_iam_user.id
  authentication_method = "rds_iam_authentication"
  cloud_provider        = "aws"
  region                = "us-west-2" # Required for RDS IAM auth

  tags = {
    environment = "production"
    application = "analytics"
    auth_method = "rds-iam"
  }
}
