# SOC2 CC6.1 - Logical and Physical Access Controls
# Test fixture for access control evidence collection

terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# IAM Policy for least privilege access
resource "aws_iam_policy" "least_privilege_policy" {
  name        = "${var.environment}-least-privilege-policy"
  description = "Least privilege policy for SOC2 compliance"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "S3ReadOnlyAccess"
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:GetObjectVersion",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.application_data.arn,
          "${aws_s3_bucket.application_data.arn}/*"
        ]
        Condition = {
          StringEquals = {
            "s3:x-amz-server-side-encryption" = "aws:kms"
          }
        }
      },
      {
        Sid    = "CloudWatchLogsAccess"
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogStreams"
        ]
        Resource = "arn:aws:logs:*:${data.aws_caller_identity.current.account_id}:log-group:/aws/lambda/${var.environment}-*"
      },
      {
        Sid    = "KMSDecryptAccess"
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
          "kms:DescribeKey",
          "kms:GenerateDataKey"
        ]
        Resource = [
          aws_kms_key.application_key.arn
        ]
        Condition = {
          StringEquals = {
            "kms:EncryptionContext:Environment" = var.environment
          }
        }
      }
    ]
  })

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.1"
    Environment = var.environment
  }
}

# IAM Role with proper assume role policy
resource "aws_iam_role" "application_role" {
  name = "${var.environment}-application-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
        Condition = {
          StringEquals = {
            "aws:RequestedRegion" = [
              "us-east-1",
              "us-west-2"
            ]
          }
        }
      }
    ]
  })

  max_session_duration = 3600 # 1 hour max session

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.1"
    Environment = var.environment
  }
}

# Attach policy to role
resource "aws_iam_role_policy_attachment" "application_policy" {
  role       = aws_iam_role.application_role.name
  policy_arn = aws_iam_policy.least_privilege_policy.arn
}

# IAM Group for admin users
resource "aws_iam_group" "administrators" {
  name = "${var.environment}-administrators"
  path = "/"
}

# Admin policy with conditions
resource "aws_iam_policy" "admin_policy" {
  name        = "${var.environment}-admin-policy"
  description = "Administrative access with MFA requirement"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AllowFullAccessWithMFA"
        Effect = "Allow"
        Action = "*"
        Resource = "*"
        Condition = {
          Bool = {
            "aws:MultiFactorAuthPresent" = "true"
          }
          NumericLessThan = {
            "aws:MultiFactorAuthAge" = "3600"
          }
        }
      },
      {
        Sid    = "AllowListActionsWithoutMFA"
        Effect = "Allow"
        Action = [
          "iam:ListUsers",
          "iam:ListRoles",
          "iam:ListPolicies",
          "iam:GetAccountSummary",
          "sts:GetCallerIdentity"
        ]
        Resource = "*"
      },
      {
        Sid    = "DenyHighRiskActionsWithoutMFA"
        Effect = "Deny"
        Action = [
          "iam:CreateUser",
          "iam:DeleteUser",
          "iam:CreateRole",
          "iam:DeleteRole",
          "iam:AttachUserPolicy",
          "iam:DetachUserPolicy",
          "iam:AttachRolePolicy",
          "iam:DetachRolePolicy"
        ]
        Resource = "*"
        Condition = {
          Bool = {
            "aws:MultiFactorAuthPresent" = "false"
          }
        }
      }
    ]
  })

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.1"
    Environment = var.environment
  }
}

# Attach admin policy to group
resource "aws_iam_group_policy_attachment" "admin_policy" {
  group      = aws_iam_group.administrators.name
  policy_arn = aws_iam_policy.admin_policy.arn
}

# IAM User with proper configuration
resource "aws_iam_user" "service_account" {
  name = "${var.environment}-service-account"
  path = "/service-accounts/"
  
  # Force password change on first login
  force_destroy = false

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.1"
    Environment = var.environment
    Type        = "ServiceAccount"
  }
}

# IAM Access Key with rotation requirements
resource "aws_iam_access_key" "service_account" {
  user = aws_iam_user.service_account.name
  
  # Note: In production, use AWS Secrets Manager for key rotation
}

# Password policy
resource "aws_iam_account_password_policy" "strict" {
  minimum_password_length        = 14
  require_lowercase_characters   = true
  require_numbers               = true
  require_uppercase_characters   = true
  require_symbols               = true
  allow_users_to_change_password = true
  max_password_age              = 90
  password_reuse_prevention     = 5
  hard_expiry                   = false
}

# Network ACL for additional access control
resource "aws_network_acl" "secure_nacl" {
  vpc_id     = var.vpc_id
  subnet_ids = var.private_subnet_ids

  # Inbound rules
  ingress {
    protocol   = "tcp"
    rule_no    = 100
    action     = "allow"
    cidr_block = var.vpc_cidr
    from_port  = 443
    to_port    = 443
  }

  ingress {
    protocol   = "tcp"
    rule_no    = 200
    action     = "allow"
    cidr_block = var.vpc_cidr
    from_port  = 80
    to_port    = 80
  }

  # Deny all other inbound traffic
  ingress {
    protocol   = -1
    rule_no    = 32766
    action     = "deny"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }

  # Outbound rules
  egress {
    protocol   = -1
    rule_no    = 100
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }

  tags = {
    Name        = "${var.environment}-secure-nacl"
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.1"
    Environment = var.environment
  }
}

# Security Group with restrictive rules
resource "aws_security_group" "web_tier" {
  name_prefix = "${var.environment}-web-"
  vpc_id      = var.vpc_id
  description = "Security group for web tier with restricted access"

  # HTTPS from ALB only
  ingress {
    description     = "HTTPS from ALB"
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }

  # HTTP redirect from ALB only
  ingress {
    description     = "HTTP redirect from ALB"
    from_port       = 80
    to_port         = 80
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }

  # SSH from bastion only
  ingress {
    description     = "SSH from bastion host"
    from_port       = 22
    to_port         = 22
    protocol        = "tcp"
    security_groups = [aws_security_group.bastion.id]
  }

  # Outbound to specific services
  egress {
    description = "HTTPS to external services"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    description = "Database access"
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    security_groups = [aws_security_group.database.id]
  }

  tags = {
    Name        = "${var.environment}-web-tier-sg"
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.1"
    Environment = var.environment
  }
}

# ALB Security Group
resource "aws_security_group" "alb" {
  name_prefix = "${var.environment}-alb-"
  vpc_id      = var.vpc_id
  description = "Security group for Application Load Balancer"

  ingress {
    description = "HTTPS from internet"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = var.allowed_cidr_blocks
  }

  ingress {
    description = "HTTP from internet (redirect to HTTPS)"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = var.allowed_cidr_blocks
  }

  egress {
    description = "All outbound to web tier"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    security_groups = [aws_security_group.web_tier.id]
  }

  tags = {
    Name        = "${var.environment}-alb-sg"
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.1"
    Environment = var.environment
  }
}

# Bastion Security Group
resource "aws_security_group" "bastion" {
  name_prefix = "${var.environment}-bastion-"
  vpc_id      = var.vpc_id
  description = "Security group for bastion host"

  ingress {
    description = "SSH from management network"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.management_cidr_blocks
  }

  egress {
    description = "SSH to private networks"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }

  tags = {
    Name        = "${var.environment}-bastion-sg"
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.1"
    Environment = var.environment
  }
}

# Database Security Group
resource "aws_security_group" "database" {
  name_prefix = "${var.environment}-db-"
  vpc_id      = var.vpc_id
  description = "Security group for database tier"

  ingress {
    description     = "PostgreSQL from web tier"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.web_tier.id]
  }

  # No outbound rules - database should not initiate connections

  tags = {
    Name        = "${var.environment}-database-sg"
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.1"
    Environment = var.environment
  }
}

# Supporting resources
resource "aws_s3_bucket" "application_data" {
  bucket = "${var.environment}-app-data-${random_string.bucket_suffix.result}"

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.1"
    Environment = var.environment
  }
}

resource "aws_s3_bucket_public_access_block" "application_data" {
  bucket = aws_s3_bucket.application_data.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_kms_key" "application_key" {
  description             = "Application encryption key"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.1"
    Environment = var.environment
  }
}

data "aws_caller_identity" "current" {}

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
  description = "VPC ID where resources will be created"
  type        = string
}

variable "vpc_cidr" {
  description = "CIDR block of the VPC"
  type        = string
}

variable "private_subnet_ids" {
  description = "List of private subnet IDs"
  type        = list(string)
}

variable "allowed_cidr_blocks" {
  description = "CIDR blocks allowed to access public services"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "management_cidr_blocks" {
  description = "CIDR blocks for management access"
  type        = list(string)
  default     = ["10.0.0.0/8"]
}

# Outputs
output "application_role_arn" {
  description = "ARN of the application IAM role"
  value       = aws_iam_role.application_role.arn
}

output "service_account_arn" {
  description = "ARN of the service account user"
  value       = aws_iam_user.service_account.arn
}

output "access_key_id" {
  description = "Access key ID for service account"
  value       = aws_iam_access_key.service_account.id
  sensitive   = true
}

output "security_group_ids" {
  description = "Map of security group IDs"
  value = {
    web_tier = aws_security_group.web_tier.id
    alb      = aws_security_group.alb.id
    bastion  = aws_security_group.bastion.id
    database = aws_security_group.database.id
  }
}