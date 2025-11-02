# Create a secret for database authentication
resource "cyberarksia_database_secret" "postgres_admin" {
  name                = "postgres-admin-credentials"
  authentication_type = "local"
  username            = "postgres_admin"
  password            = "SuperSecret123!" # In production, use variables or secret management

  tags = {
    environment = "production"
    managed_by  = "terraform"
    purpose     = "database-access"
  }
}

# Example: AWS RDS IAM authentication secret
resource "cyberarksia_database_secret" "rds_iam_user" {
  name                  = "rds-iam-user-credentials"
  authentication_type   = "aws_iam"
  aws_access_key_id     = "AKIAIOSFODNN7EXAMPLE" # In production, use variables
  aws_secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

  tags = {
    environment = "production"
    cloud       = "aws"
    auth_method = "iam"
  }
}
