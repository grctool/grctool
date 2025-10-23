# Backup Configuration 1
# SOC2 Controls: SO2

resource "aws_backup_vault" "vault_1" {
  name        = "backup-vault-1"
  kms_key_arn = aws_kms_key.backup.arn

  tags = {
    Name = "backup-vault-1"
  }
}

resource "aws_backup_plan" "plan_1" {
  name = "backup-plan-1"

  rule {
    rule_name         = "hourly-backup-1"
    target_vault_name = aws_backup_vault.vault_1.name
    schedule          = "cron(0 * * * ? *)"

    lifecycle {
      delete_after = 7
    }
  }

  rule {
    rule_name         = "daily-backup-1"
    target_vault_name = aws_backup_vault.vault_1.name
    schedule          = "cron(0 5 * * ? *)"

    lifecycle {
      cold_storage_after = 30
      delete_after       = 365
    }
  }

  tags = {
    Name = "backup-plan-1"
  }
}

resource "aws_backup_selection" "selection_1" {
  name         = "backup-selection-1"
  plan_id      = aws_backup_plan.plan_1.id
  iam_role_arn = aws_iam_role.backup.arn

  selection_tag {
    type  = "STRINGEQUALS"
    key   = "Backup"
    value = "true"
  }
}
