# IAM roles and policies for security compliance
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Application service role with least privilege
resource "aws_iam_role" "app_service_role" {
  name = "security-app-service-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
        Action = "sts:AssumeRole"
      }
    ]
  })

  tags = {
    Purpose     = "application-service"
    Environment = "production"
    Compliance  = "SOC2"
  }
}

# Custom policy for S3 access (least privilege)
resource "aws_iam_policy" "s3_access_policy" {
  name        = "security-s3-access-policy"
  description = "Least privilege policy for S3 access"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]
        Resource = [
          "${aws_s3_bucket.secure_data.arn}/*"
        ]
        Condition = {
          StringEquals = {
            "s3:x-amz-server-side-encryption" = "aws:kms"
          }
        }
      },
      {
        Effect = "Allow"
        Action = [
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.secure_data.arn
        ]
      }
    ]
  })
}

# Attach S3 policy to service role
resource "aws_iam_role_policy_attachment" "app_s3_access" {
  role       = aws_iam_role.app_service_role.name
  policy_arn = aws_iam_policy.s3_access_policy.arn
}

# CloudWatch logging policy
resource "aws_iam_policy" "cloudwatch_logs_policy" {
  name        = "security-cloudwatch-logs-policy"
  description = "Policy for CloudWatch logs access"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogStreams",
          "logs:DescribeLogGroups"
        ]
        Resource = "arn:aws:logs:*:${data.aws_caller_identity.current.account_id}:*"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "app_cloudwatch_logs" {
  role       = aws_iam_role.app_service_role.name
  policy_arn = aws_iam_policy.cloudwatch_logs_policy.arn
}

# Instance profile for EC2
resource "aws_iam_instance_profile" "app_service_profile" {
  name = "security-app-service-profile"
  role = aws_iam_role.app_service_role.name
}

# Database access role with strict permissions
resource "aws_iam_role" "db_access_role" {
  name = "security-db-access-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "rds.amazonaws.com"
        }
        Action = "sts:AssumeRole"
      }
    ]
  })

  tags = {
    Purpose = "database-access"
  }
}

# Secrets Manager access policy for database credentials
resource "aws_iam_policy" "secrets_access_policy" {
  name        = "security-secrets-access-policy"
  description = "Policy for accessing database credentials in Secrets Manager"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret"
        ]
        Resource = [
          aws_secretsmanager_secret.db_credentials.arn
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "kms:Decrypt"
        ]
        Resource = [
          aws_kms_key.secrets_key.arn
        ]
        Condition = {
          StringEquals = {
            "kms:ViaService" = "secretsmanager.${data.aws_region.current.name}.amazonaws.com"
          }
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "app_secrets_access" {
  role       = aws_iam_role.app_service_role.name
  policy_arn = aws_iam_policy.secrets_access_policy.arn
}

# Lambda execution role for security functions
resource "aws_iam_role" "security_lambda_role" {
  name = "security-lambda-execution-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
        Action = "sts:AssumeRole"
      }
    ]
  })

  tags = {
    Purpose = "lambda-security-functions"
  }
}

# Basic Lambda execution policy
resource "aws_iam_role_policy_attachment" "lambda_basic_execution" {
  role       = aws_iam_role.security_lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# Custom policy for Lambda security functions
resource "aws_iam_policy" "lambda_security_policy" {
  name        = "security-lambda-policy"
  description = "Policy for Lambda security functions"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "cloudtrail:LookupEvents",
          "guardduty:GetDetector",
          "guardduty:ListDetectors",
          "securityhub:GetFindings",
          "config:GetComplianceDetailsByConfigRule"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "sns:Publish"
        ]
        Resource = [
          "arn:aws:sns:*:${data.aws_caller_identity.current.account_id}:security-alerts"
        ]
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_security_policy" {
  role       = aws_iam_role.security_lambda_role.name
  policy_arn = aws_iam_policy.lambda_security_policy.arn
}

# Cross-account access role for auditing
resource "aws_iam_role" "audit_cross_account_role" {
  name = "security-audit-cross-account-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          AWS = var.audit_account_id != "" ? "arn:aws:iam::${var.audit_account_id}:root" : data.aws_caller_identity.current.arn
        }
        Action = "sts:AssumeRole"
        Condition = {
          StringEquals = {
            "sts:ExternalId" = var.external_id
          }
        }
      }
    ]
  })

  tags = {
    Purpose = "cross-account-audit"
  }
}

# Read-only audit policy
resource "aws_iam_policy" "audit_readonly_policy" {
  name        = "security-audit-readonly-policy"
  description = "Read-only policy for security auditing"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "iam:Get*",
          "iam:List*",
          "s3:GetBucketPolicy",
          "s3:GetBucketAcl",
          "s3:GetBucketLocation",
          "s3:GetBucketLogging",
          "s3:GetBucketVersioning",
          "s3:ListAllMyBuckets",
          "cloudtrail:DescribeTrails",
          "cloudtrail:GetTrailStatus",
          "config:DescribeConfigRules",
          "config:GetComplianceSummaryByConfigRule"
        ]
        Resource = "*"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "audit_readonly" {
  role       = aws_iam_role.audit_cross_account_role.name
  policy_arn = aws_iam_policy.audit_readonly_policy.arn
}

# Variables
variable "audit_account_id" {
  description = "AWS Account ID for cross-account audit access"
  type        = string
  default     = ""
}

variable "external_id" {
  description = "External ID for cross-account role assumption"
  type        = string
  default     = "security-audit-2024"
}

# Data sources
data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

# Reference to resources from other files
data "aws_s3_bucket" "secure_data" {
  bucket = "security-compliance-data"
  depends_on = [aws_s3_bucket.secure_data]
}

data "aws_secretsmanager_secret" "db_credentials" {
  name = "production/database/credentials"
  depends_on = [aws_secretsmanager_secret.db_credentials]
}

data "aws_kms_key" "secrets_key" {
  key_id = "alias/secrets-key"
  depends_on = [aws_kms_key.secrets_key]
}