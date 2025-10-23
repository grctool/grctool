# Shared Resources
# SOC2 Controls: CC6.1, CC6.8, CC7.2

resource "aws_s3_bucket" "logs" {
  bucket = "company-central-logs-bucket"

  tags = {
    Name = "central-logs-bucket"
  }
}

resource "aws_s3_bucket_encryption" "logs" {
  bucket = aws_s3_bucket.logs.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.logs.arn
    }
  }
}

resource "aws_kms_key" "logs" {
  description             = "Central logs encryption key"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Name = "logs-key"
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

resource "aws_kms_key" "cloudtrail" {
  description             = "CloudTrail encryption key"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Name = "cloudtrail-key"
  }
}

resource "aws_kms_key" "cloudwatch" {
  description             = "CloudWatch Logs encryption key"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Name = "cloudwatch-key"
  }
}

resource "aws_iam_role" "backup" {
  name = "aws-backup-service-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "backup.amazonaws.com"
      }
    }]
  })

  tags = {
    Name = "backup-service-role"
  }
}
