# SOC2 CC8.1 - Change Tracking and Audit Logging
# Test fixture for audit and logging evidence collection

terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# CloudTrail for comprehensive audit logging
resource "aws_cloudtrail" "organization_trail" {
  name                         = "${var.environment}-organization-audit-trail"
  s3_bucket_name              = aws_s3_bucket.cloudtrail_logs.bucket
  s3_key_prefix               = "cloudtrail-logs/"
  include_global_service_events = true
  is_multi_region_trail       = true
  enable_logging              = true
  
  # Enable log file validation for integrity
  enable_log_file_validation = true
  
  # KMS encryption for logs
  kms_key_id = aws_kms_key.audit_key.arn
  
  # SNS notifications for log delivery
  sns_topic_name = aws_sns_topic.audit_notifications.name

  # Comprehensive event logging
  event_selector {
    read_write_type                 = "All"
    include_management_events       = true
    exclude_management_event_sources = []

    data_resource {
      type   = "AWS::S3::Object"
      values = ["arn:aws:s3:::*/*"]
    }

    data_resource {
      type   = "AWS::Lambda::Function"
      values = ["arn:aws:lambda:*"]
    }
  }

  # Advanced event selectors for detailed logging
  advanced_event_selector {
    name = "Log all management events"
    field_selector {
      field  = "eventCategory"
      equals = ["Management"]
    }
  }

  advanced_event_selector {
    name = "Log S3 data events"
    field_selector {
      field  = "eventCategory"
      equals = ["Data"]
    }
    field_selector {
      field  = "resources.type"
      equals = ["AWS::S3::Object"]
    }
  }

  advanced_event_selector {
    name = "Log Lambda data events"
    field_selector {
      field  = "eventCategory"
      equals = ["Data"]
    }
    field_selector {
      field  = "resources.type"
      equals = ["AWS::Lambda::Function"]
    }
  }

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC8.1"
    Environment = var.environment
  }
}

# S3 bucket for CloudTrail logs
resource "aws_s3_bucket" "cloudtrail_logs" {
  bucket = "${var.environment}-cloudtrail-logs-${random_string.bucket_suffix.result}"

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC8.1"
    Environment = var.environment
  }
}

# CloudTrail bucket policy
resource "aws_s3_bucket_policy" "cloudtrail_logs" {
  bucket = aws_s3_bucket.cloudtrail_logs.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AWSCloudTrailAclCheck"
        Effect = "Allow"
        Principal = {
          Service = "cloudtrail.amazonaws.com"
        }
        Action   = "s3:GetBucketAcl"
        Resource = aws_s3_bucket.cloudtrail_logs.arn
        Condition = {
          StringEquals = {
            "AWS:SourceArn" = "arn:aws:cloudtrail:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:trail/${var.environment}-organization-audit-trail"
          }
        }
      },
      {
        Sid    = "AWSCloudTrailWrite"
        Effect = "Allow"
        Principal = {
          Service = "cloudtrail.amazonaws.com"
        }
        Action   = "s3:PutObject"
        Resource = "${aws_s3_bucket.cloudtrail_logs.arn}/*"
        Condition = {
          StringEquals = {
            "s3:x-amz-acl" = "bucket-owner-full-control"
            "AWS:SourceArn" = "arn:aws:cloudtrail:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:trail/${var.environment}-organization-audit-trail"
          }
        }
      }
    ]
  })
}

# Lifecycle policy for CloudTrail logs
resource "aws_s3_bucket_lifecycle_configuration" "cloudtrail_logs" {
  bucket = aws_s3_bucket.cloudtrail_logs.id

  rule {
    id     = "audit_log_lifecycle"
    status = "Enabled"

    # Keep current version for 7 years (SOC2 requirement)
    expiration {
      days = 2555 # 7 years
    }

    # Transition to cheaper storage classes
    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }

    transition {
      days          = 90
      storage_class = "GLACIER"
    }

    transition {
      days          = 365
      storage_class = "DEEP_ARCHIVE"
    }

    noncurrent_version_expiration {
      noncurrent_days = 30
    }
  }
}

# VPC Flow Logs for network monitoring
resource "aws_flow_log" "vpc_flow_logs" {
  iam_role_arn    = aws_iam_role.flow_log_role.arn
  log_destination = aws_cloudwatch_log_group.vpc_flow_logs.arn
  traffic_type    = "ALL"
  vpc_id          = var.vpc_id

  tags = {
    Name        = "${var.environment}-vpc-flow-logs"
    Purpose     = "SOC2 Compliance"
    Control     = "CC8.1"
    Environment = var.environment
  }
}

# CloudWatch Log Group for VPC Flow Logs
resource "aws_cloudwatch_log_group" "vpc_flow_logs" {
  name              = "/aws/vpc/flowlogs/${var.environment}"
  retention_in_days = 2555 # 7 years
  kms_key_id        = aws_kms_key.audit_key.arn

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC8.1"
    Environment = var.environment
  }
}

# IAM role for VPC Flow Logs
resource "aws_iam_role" "flow_log_role" {
  name = "${var.environment}-flow-log-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "vpc-flow-logs.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC8.1"
    Environment = var.environment
  }
}

# IAM policy for VPC Flow Logs
resource "aws_iam_role_policy" "flow_log_policy" {
  name = "${var.environment}-flow-log-policy"
  role = aws_iam_role.flow_log_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogGroups",
          "logs:DescribeLogStreams"
        ]
        Effect   = "Allow"
        Resource = "*"
      }
    ]
  })
}

# AWS Config for configuration change tracking
resource "aws_config_configuration_recorder" "audit_recorder" {
  name     = "${var.environment}-audit-recorder"
  role_arn = aws_iam_role.config_role.arn

  recording_group {
    all_supported                 = true
    include_global_resource_types = true
  }

  depends_on = [aws_config_delivery_channel.audit_delivery_channel]
}

resource "aws_config_delivery_channel" "audit_delivery_channel" {
  name           = "${var.environment}-audit-delivery-channel"
  s3_bucket_name = aws_s3_bucket.config_logs.bucket
  s3_key_prefix  = "config-logs/"
  
  snapshot_delivery_properties {
    delivery_frequency = "Daily"
  }
}

# S3 bucket for AWS Config
resource "aws_s3_bucket" "config_logs" {
  bucket = "${var.environment}-config-logs-${random_string.bucket_suffix.result}"

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC8.1"
    Environment = var.environment
  }
}

# Config bucket policy
resource "aws_s3_bucket_policy" "config_logs" {
  bucket = aws_s3_bucket.config_logs.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AWSConfigBucketPermissionsCheck"
        Effect = "Allow"
        Principal = {
          Service = "config.amazonaws.com"
        }
        Action   = "s3:GetBucketAcl"
        Resource = aws_s3_bucket.config_logs.arn
        Condition = {
          StringEquals = {
            "AWS:SourceAccount" = data.aws_caller_identity.current.account_id
          }
        }
      },
      {
        Sid    = "AWSConfigBucketExistenceCheck"
        Effect = "Allow"
        Principal = {
          Service = "config.amazonaws.com"
        }
        Action   = "s3:ListBucket"
        Resource = aws_s3_bucket.config_logs.arn
        Condition = {
          StringEquals = {
            "AWS:SourceAccount" = data.aws_caller_identity.current.account_id
          }
        }
      },
      {
        Sid    = "AWSConfigBucketDelivery"
        Effect = "Allow"
        Principal = {
          Service = "config.amazonaws.com"
        }
        Action   = "s3:PutObject"
        Resource = "${aws_s3_bucket.config_logs.arn}/*"
        Condition = {
          StringEquals = {
            "s3:x-amz-acl" = "bucket-owner-full-control"
            "AWS:SourceAccount" = data.aws_caller_identity.current.account_id
          }
        }
      }
    ]
  })
}

# IAM role for AWS Config
resource "aws_iam_role" "config_role" {
  name = "${var.environment}-config-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "config.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC8.1"
    Environment = var.environment
  }
}

resource "aws_iam_role_policy_attachment" "config_role_policy" {
  role       = aws_iam_role.config_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/ConfigRole"
}

# CloudWatch alarms for security monitoring
resource "aws_cloudwatch_log_metric_filter" "root_login" {
  name           = "${var.environment}-root-login-filter"
  log_group_name = aws_cloudwatch_log_group.cloudtrail_logs.name
  pattern        = "{ $.userIdentity.type = \"Root\" && $.userIdentity.invokedBy NOT EXISTS && $.eventType != \"AwsServiceEvent\" }"

  metric_transformation {
    name      = "RootLoginCount"
    namespace = "SOC2/Security"
    value     = "1"
  }
}

resource "aws_cloudwatch_alarm" "root_login_alarm" {
  alarm_name          = "${var.environment}-root-login-alarm"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = "1"
  metric_name         = "RootLoginCount"
  namespace           = "SOC2/Security"
  period              = "300"
  statistic           = "Sum"
  threshold           = "1"
  alarm_description   = "Root user login detected"
  alarm_actions       = [aws_sns_topic.security_alerts.arn]

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC8.1"
    Environment = var.environment
  }
}

# Failed login attempts
resource "aws_cloudwatch_log_metric_filter" "failed_logins" {
  name           = "${var.environment}-failed-logins-filter"
  log_group_name = aws_cloudwatch_log_group.cloudtrail_logs.name
  pattern        = "{ ($.errorCode = \"*UnauthorizedOperation\") || ($.errorCode = \"AccessDenied*\") }"

  metric_transformation {
    name      = "FailedLoginCount"
    namespace = "SOC2/Security"
    value     = "1"
  }
}

resource "aws_cloudwatch_alarm" "failed_logins_alarm" {
  alarm_name          = "${var.environment}-failed-logins-alarm"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "FailedLoginCount"
  namespace           = "SOC2/Security"
  period              = "300"
  statistic           = "Sum"
  threshold           = "10"
  alarm_description   = "High number of failed login attempts"
  alarm_actions       = [aws_sns_topic.security_alerts.arn]

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC8.1"
    Environment = var.environment
  }
}

# CloudWatch Log Groups for various services
resource "aws_cloudwatch_log_group" "cloudtrail_logs" {
  name              = "/aws/cloudtrail/${var.environment}"
  retention_in_days = 2555 # 7 years
  kms_key_id        = aws_kms_key.audit_key.arn

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC8.1"
    Environment = var.environment
  }
}

resource "aws_cloudwatch_log_group" "application_logs" {
  name              = "/aws/lambda/${var.environment}-application"
  retention_in_days = 365 # 1 year for application logs
  kms_key_id        = aws_kms_key.audit_key.arn

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC8.1"
    Environment = var.environment
  }
}

# SNS topics for notifications
resource "aws_sns_topic" "audit_notifications" {
  name              = "${var.environment}-audit-notifications"
  kms_master_key_id = aws_kms_key.audit_key.id

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC8.1"
    Environment = var.environment
  }
}

resource "aws_sns_topic" "security_alerts" {
  name              = "${var.environment}-security-alerts"
  kms_master_key_id = aws_kms_key.audit_key.id

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC8.1"
    Environment = var.environment
  }
}

# KMS key for audit encryption
resource "aws_kms_key" "audit_key" {
  description             = "KMS key for audit and logging encryption"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action   = "kms:*"
        Resource = "*"
      },
      {
        Sid    = "Allow CloudTrail encryption"
        Effect = "Allow"
        Principal = {
          Service = "cloudtrail.amazonaws.com"
        }
        Action = [
          "kms:GenerateDataKey*",
          "kms:DescribeKey",
          "kms:Decrypt"
        ]
        Resource = "*"
      },
      {
        Sid    = "Allow CloudWatch Logs encryption"
        Effect = "Allow"
        Principal = {
          Service = "logs.amazonaws.com"
        }
        Action = [
          "kms:Encrypt",
          "kms:Decrypt",
          "kms:ReEncrypt*",
          "kms:GenerateDataKey*",
          "kms:DescribeKey"
        ]
        Resource = "*"
        Condition = {
          ArnEquals = {
            "kms:EncryptionContext:aws:logs:arn" = "arn:aws:logs:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:*"
          }
        }
      }
    ]
  })

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC8.1"
    Environment = var.environment
  }
}

resource "aws_kms_alias" "audit_key" {
  name          = "alias/${var.environment}-audit-encryption"
  target_key_id = aws_kms_key.audit_key.key_id
}

# Supporting resources
data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

resource "random_string" "bucket_suffix" {
  length  = 8
  special = false
  upper   = false
}

# Variables
variable "environment" {
  description = "Environment name"
  type        = string
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be dev, staging, or prod."
  }
}

variable "vpc_id" {
  description = "VPC ID for flow logs"
  type        = string
}

# Outputs
output "cloudtrail_arn" {
  description = "ARN of the CloudTrail"
  value       = aws_cloudtrail.organization_trail.arn
}

output "audit_key_arn" {
  description = "ARN of the audit encryption key"
  value       = aws_kms_key.audit_key.arn
}

output "cloudtrail_bucket" {
  description = "Name of the CloudTrail S3 bucket"
  value       = aws_s3_bucket.cloudtrail_logs.bucket
}

output "config_bucket" {
  description = "Name of the AWS Config S3 bucket"
  value       = aws_s3_bucket.config_logs.bucket
}

output "log_group_arns" {
  description = "ARNs of CloudWatch log groups"
  value = {
    cloudtrail   = aws_cloudwatch_log_group.cloudtrail_logs.arn
    vpc_flow     = aws_cloudwatch_log_group.vpc_flow_logs.arn
    application  = aws_cloudwatch_log_group.application_logs.arn
  }
}