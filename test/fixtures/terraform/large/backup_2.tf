# Backup Configuration 2
# SOC2 Controls: SO2

resource "aws_backup_vault" "vault_2" {
  name        = "backup-vault-2"
  kms_key_arn = aws_kms_key.backup.arn

  tags = {
    Name = "backup-vault-2"
  }
}

resource "aws_backup_plan" "plan_2" {
  name = "backup-plan-2"

  rule {
    rule_name         = "hourly-backup-2"
    target_vault_name = aws_backup_vault.vault_2.name
    schedule          = "cron(0 * * * ? *)"

    lifecycle {
      delete_after = 7
    }
  }

  rule {
    rule_name         = "daily-backup-2"
    target_vault_name = aws_backup_vault.vault_2.name
    schedule          = "cron(0 5 * * ? *)"

    lifecycle {
      cold_storage_after = 30
      delete_after       = 365
    }
  }

  tags = {
    Name = "backup-plan-2"
  }
}

resource "aws_backup_selection" "selection_2" {
  name         = "backup-selection-2"
  plan_id      = aws_backup_plan.plan_2.id
  iam_role_arn = aws_iam_role.backup.arn

  selection_tag {
    type  = "STRINGEQUALS"
    key   = "Backup"
    value = "true"
  }
}
