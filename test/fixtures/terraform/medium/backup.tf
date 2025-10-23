# Backup Resources
# SOC2 Controls: SO2

resource "aws_backup_vault" "main" {
  name        = "main-backup-vault"
  kms_key_arn = aws_kms_key.backup.arn

  tags = {
    Name = "main-backup-vault"
  }
}

resource "aws_backup_plan" "daily" {
  name = "daily-backup-plan"

  rule {
    rule_name         = "daily-backup"
    target_vault_name = aws_backup_vault.main.name
    schedule          = "cron(0 5 * * ? *)"

    lifecycle {
      delete_after = 30
    }
  }

  tags = {
    Name = "daily-backup-plan"
  }
}

resource "aws_kms_key" "backup" {
  description             = "Backup encryption key"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Name = "backup-key"
  }
}

resource "aws_s3_bucket" "logs" {
  bucket = "company-logs-bucket"

  tags = {
    Name = "logs-bucket"
  }
}
