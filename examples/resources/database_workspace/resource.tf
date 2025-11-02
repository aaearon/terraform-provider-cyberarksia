# Basic PostgreSQL database workspace
resource "cyberarksia_database_workspace" "postgres_example" {
  name           = "myapp-database"
  database_type  = "postgres"
  address        = "db.example.com"
  port           = 5432
  secret_id      = cyberarksia_database_secret.db_credentials.id
  cloud_provider = "on_premise"

  tags = {
    environment = "production"
    application = "myapp"
  }
}

# AWS RDS PostgreSQL with IAM authentication
resource "cyberarksia_database_workspace" "rds_postgres" {
  name                  = "analytics-db"
  database_type         = "postgres-aws-rds"
  address               = "mydb.abc123.us-east-1.rds.amazonaws.com"
  port                  = 5432
  secret_id             = cyberarksia_database_secret.rds_iam_credentials.id
  authentication_method = "rds_iam_authentication"
  cloud_provider        = "aws"
  region                = "us-east-1" # Required for RDS IAM auth
  network_name          = "production-vpc"

  tags = {
    environment = "production"
    cloud       = "aws"
    auth_method = "rds-iam"
  }
}

# Azure SQL Database with TLS certificate
resource "cyberarksia_database_workspace" "azure_sql" {
  name                          = "customer-db"
  database_type                 = "mssql-azure-managed"
  address                       = "myserver.database.windows.net"
  port                          = 1433
  secret_id                     = cyberarksia_database_secret.sql_credentials.id
  certificate_id                = cyberarksia_certificate.azure_ca.id
  cloud_provider                = "azure"
  region                        = "eastus"
  enable_certificate_validation = true

  tags = {
    environment = "production"
    cloud       = "azure"
  }
}

# MongoDB Atlas
resource "cyberarksia_database_workspace" "mongo_atlas" {
  name           = "app-data"
  database_type  = "mongo-atlas-managed"
  address        = "cluster0.abc123.mongodb.net"
  port           = 27017
  secret_id      = cyberarksia_database_secret.mongo_credentials.id
  cloud_provider = "atlas"
  account        = "my-atlas-account"
  auth_database  = "admin"

  tags = {
    environment = "production"
    database    = "mongodb"
  }
}
