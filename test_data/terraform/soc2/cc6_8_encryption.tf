# SOC2 CC6.8 - Encryption at Rest and in Transit
# Test fixture for encryption-related evidence collection

terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# KMS Key with proper rotation
resource "aws_kms_key" "main_encryption_key" {
  description             = "Main encryption key for SOC2 compliance"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region           = true
  
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
          "kms:DescribeKey"
        ]
        Resource = "*"
      }
    ]
  })

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.8"
    Environment = var.environment
  }
}

resource "aws_kms_alias" "main_encryption_key" {
  name          = "alias/${var.environment}-main-encryption"
  target_key_id = aws_kms_key.main_encryption_key.key_id
}

# S3 Bucket with KMS encryption
resource "aws_s3_bucket" "encrypted_data" {
  bucket = "${var.environment}-encrypted-data-${random_string.bucket_suffix.result}"

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.8"
    Environment = var.environment
  }
}

resource "aws_s3_bucket_encryption" "encrypted_data" {
  bucket = aws_s3_bucket.encrypted_data.id

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        kms_master_key_id = aws_kms_key.main_encryption_key.arn
        sse_algorithm     = "aws:kms"
      }
      bucket_key_enabled = true
    }
  }
}

resource "aws_s3_bucket_public_access_block" "encrypted_data" {
  bucket = aws_s3_bucket.encrypted_data.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# RDS with encryption
resource "aws_db_instance" "encrypted_database" {
  identifier     = "${var.environment}-encrypted-db"
  engine         = "postgres"
  engine_version = "14.9"
  instance_class = "db.t3.micro"
  
  allocated_storage     = 20
  max_allocated_storage = 100
  storage_encrypted     = true
  kms_key_id           = aws_kms_key.main_encryption_key.arn
  
  db_name  = "compliancedb"
  username = var.db_username
  password = var.db_password
  
  # Encryption in transit
  ca_cert_identifier = "rds-ca-2019"
  
  # Backup encryption
  backup_retention_period = 7
  backup_window          = "03:00-04:00"
  
  # Monitoring
  monitoring_interval = 60
  monitoring_role_arn = aws_iam_role.rds_monitoring.arn
  
  # Security
  vpc_security_group_ids = [aws_security_group.database.id]
  db_subnet_group_name   = aws_db_subnet_group.main.name
  
  # Prevent deletion
  deletion_protection = true
  skip_final_snapshot = false
  final_snapshot_identifier = "${var.environment}-encrypted-db-final-snapshot"

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.8"
    Environment = var.environment
  }
}

# EBS volumes with encryption
resource "aws_ebs_volume" "encrypted_volume" {
  availability_zone = data.aws_availability_zones.available.names[0]
  size             = 10
  type             = "gp3"
  encrypted        = true
  kms_key_id       = aws_kms_key.main_encryption_key.arn

  tags = {
    Name        = "${var.environment}-encrypted-volume"
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.8"
    Environment = var.environment
  }
}

# EFS with encryption
resource "aws_efs_file_system" "encrypted_efs" {
  creation_token = "${var.environment}-encrypted-efs"
  encrypted      = true
  kms_key_id     = aws_kms_key.main_encryption_key.arn
  
  performance_mode = "generalPurpose"
  throughput_mode  = "provisioned"
  provisioned_throughput_in_mibps = 500

  tags = {
    Name        = "${var.environment}-encrypted-efs"
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.8"
    Environment = var.environment
  }
}

# SNS Topic with encryption
resource "aws_sns_topic" "encrypted_notifications" {
  name              = "${var.environment}-encrypted-notifications"
  kms_master_key_id = aws_kms_key.main_encryption_key.id

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.8"
    Environment = var.environment
  }
}

# SQS Queue with encryption
resource "aws_sqs_queue" "encrypted_queue" {
  name                       = "${var.environment}-encrypted-queue"
  kms_master_key_id         = aws_kms_key.main_encryption_key.id
  kms_data_key_reuse_period_seconds = 300

  tags = {
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.8"
    Environment = var.environment
  }
}

# Supporting resources
data "aws_caller_identity" "current" {}
data "aws_availability_zones" "available" {
  state = "available"
}

resource "random_string" "bucket_suffix" {
  length  = 8
  special = false
  upper   = false
}

# Database subnet group
resource "aws_db_subnet_group" "main" {
  name       = "${var.environment}-db-subnet-group"
  subnet_ids = var.private_subnet_ids

  tags = {
    Name        = "${var.environment}-db-subnet-group"
    Purpose     = "SOC2 Compliance"
    Environment = var.environment
  }
}

# Security group for database
resource "aws_security_group" "database" {
  name_prefix = "${var.environment}-db-"
  vpc_id      = var.vpc_id
  description = "Security group for encrypted database"

  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.application.id]
    description     = "PostgreSQL access from application"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "All outbound traffic"
  }

  tags = {
    Name        = "${var.environment}-database-sg"
    Purpose     = "SOC2 Compliance"
    Control     = "CC6.8"
    Environment = var.environment
  }
}

# Application security group
resource "aws_security_group" "application" {
  name_prefix = "${var.environment}-app-"
  vpc_id      = var.vpc_id
  description = "Security group for application servers"

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = var.allowed_cidr_blocks
    description = "HTTPS traffic"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "All outbound traffic"
  }

  tags = {
    Name        = "${var.environment}-application-sg"
    Purpose     = "SOC2 Compliance"
    Environment = var.environment
  }
}

# RDS monitoring role
resource "aws_iam_role" "rds_monitoring" {
  name = "${var.environment}-rds-monitoring-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "monitoring.rds.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Purpose     = "SOC2 Compliance"
    Environment = var.environment
  }
}

resource "aws_iam_role_policy_attachment" "rds_monitoring" {
  role       = aws_iam_role.rds_monitoring.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonRDSEnhancedMonitoringRole"
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

variable "private_subnet_ids" {
  description = "List of private subnet IDs for database"
  type        = list(string)
}

variable "allowed_cidr_blocks" {
  description = "CIDR blocks allowed to access the application"
  type        = list(string)
  default     = ["10.0.0.0/8"]
}

variable "db_username" {
  description = "Database username"
  type        = string
  default     = "postgres"
}

variable "db_password" {
  description = "Database password"
  type        = string
  sensitive   = true
}

# Outputs
output "kms_key_arn" {
  description = "ARN of the main encryption key"
  value       = aws_kms_key.main_encryption_key.arn
}

output "kms_key_id" {
  description = "ID of the main encryption key"
  value       = aws_kms_key.main_encryption_key.key_id
}

output "encrypted_bucket_name" {
  description = "Name of the encrypted S3 bucket"
  value       = aws_s3_bucket.encrypted_data.bucket
}

output "encrypted_database_endpoint" {
  description = "RDS instance endpoint"
  value       = aws_db_instance.encrypted_database.endpoint
  sensitive   = true
}